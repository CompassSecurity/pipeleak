package scanner

import (
	"bytes"
	"compress/gzip"
	"context"
	"crypto/tls"
	"database/sql"
	"io"
	"net/http"
	"net/url"
	"os"
	"path"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/CompassSecurity/pipeleak/helper"
	_ "github.com/mattn/go-sqlite3"
	"github.com/rs/zerolog/log"
	"github.com/xanzy/go-gitlab"

	"github.com/maragudk/goqite"
	"github.com/maragudk/goqite/jobs"
)

var queue *goqite.Queue
var queueCancelFn context.CancelFunc
var queueFileName string

type ScanOptions struct {
	GitlabUrl          string
	GitlabApiToken     string
	GitlabCookie       string
	ProjectSearchQuery string
	Artifacts          bool
	Owned              bool
	Member             bool
	JobLimit           int
	Verbose            bool
	ConfidenceFilter   []string
	MaxArtifactSize    int64
	MaxScanGoRoutines  int
}

func ScanGitLabPipelines(options *ScanOptions) {
	log.Debug().Msg("Setting up queue on disk")
	tmpfile, err := os.CreateTemp("", "pipeleak-queue-db")
	if err != nil {
		log.Fatal().Err(err).Msg("Creating Temp DB file failed")
	}
	defer os.Remove(tmpfile.Name())
	queueFileName = tmpfile.Name()

	setupQueue(tmpfile.Name())
	helper.RegisterGracefulShutdownHandler(cleanUp)

	r := jobs.NewRunner(jobs.NewRunnerOpts{
		Limit:        4,
		Log:          nil,
		PollInterval: 10 * time.Millisecond,
		Queue:        queue,
	})

	InitRules(options.ConfidenceFilter)

	go fetchProjects(options)

	r.Register("pipeleak-job", func(ctx context.Context, m []byte) error {
		analyzeQueueItem(m, options.MaxScanGoRoutines)
		return nil
	})

	queueCtx, cancelFunc := context.WithCancel(context.Background())
	queueCancelFn = cancelFunc
	r.Start(queueCtx)
}

func setupQueue(fileName string) {
	sqlUri := `file://` + fileName + `:?_journal=WAL&_timeout=5000&_fk=true`
	db, err := sql.Open("sqlite3", sqlUri)
	log.Debug().Str("file", sqlUri).Msg("Using DB file")
	if err != nil {
		log.Fatal().Err(err).Str("file", fileName).Msg("Opening Temp DB file failed")
	}
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)

	if err := goqite.Setup(context.Background(), db); err != nil {
		log.Fatal().Err(err).Msg("Goqite setup failed")
	}

	queue = goqite.New(goqite.NewOpts{
		DB:         db,
		Name:       "jobs",
		MaxReceive: 100,
	})
}

func cleanUp() {
	log.Debug().Str("file", queueFileName).Msg("Graceful Shutdown, removing queue database")
	os.Remove(queueFileName)
}

func fetchProjects(options *ScanOptions) {
	log.Info().Msg("Fetching projects")

	git, err := gitlab.NewClient(options.GitlabApiToken, gitlab.WithBaseURL(options.GitlabUrl))
	if err != nil {
		log.Fatal().Stack().Err(err)
	}

	if len(options.GitlabCookie) > 0 {
		SessionValid(options.GitlabUrl, options.GitlabCookie)
	}

	if len(options.ProjectSearchQuery) > 0 {
		log.Info().Str("query", options.ProjectSearchQuery).Msg("Filtering scanned projects by")
	}

	projectOpts := &gitlab.ListProjectsOptions{
		ListOptions: gitlab.ListOptions{
			PerPage: 100,
			Page:    1,
		},
		Owned:      gitlab.Ptr(options.Owned),
		Membership: gitlab.Ptr(options.Member),
		Search:     gitlab.Ptr(options.ProjectSearchQuery),
		OrderBy:    gitlab.Ptr("last_activity_at"),
	}

	for {
		projects, resp, err := git.Projects.ListProjects(projectOpts)
		if err != nil {
			log.Error().Stack().Err(err).Msg("Failed fetching projects")
		}

		for _, project := range projects {
			log.Debug().Str("url", project.WebURL).Msg("Fetch Project jobs for")
			getAllJobs(git, project, options)
		}

		if resp.NextPage == 0 {
			break
		}
		projectOpts.Page = resp.NextPage
		log.Info().Int("total", projectOpts.Page*projectOpts.PerPage).Msg("Fetched projects")

	}
	queueCancelFn()
}

func getAllJobs(git *gitlab.Client, project *gitlab.Project, options *ScanOptions) {

	opts := &gitlab.ListJobsOptions{
		ListOptions: gitlab.ListOptions{
			PerPage: 100,
			Page:    1,
		},
	}

	currentJobCtr := 0

jobOut:
	for {
		jobs, resp, err := git.Jobs.ListProjectJobs(project.ID, opts)

		if err != nil {
			log.Debug().Stack().Err(err).Msg("Failed fetching project jobs")
		}

		for _, job := range jobs {
			currentJobCtr += 1
			hitMeta := HitMetaInfo{JobId: job.ID, ProjectId: project.ID, JobWebUrl: getJobUrl(git, project, job)}
			enqueueItem(nil, queue, QueueItemJobTrace, hitMeta)

			getJobTrace(git, project, job, hitMeta)

			if options.Artifacts {
				getJobArtifacts(git, project, job, options, hitMeta)
			}

			if options.JobLimit > 0 && currentJobCtr >= options.JobLimit {
				break jobOut
			}
		}

		if resp.NextPage == 0 {
			break
		}

		opts.Page = resp.NextPage
	}

}

func getJobTrace(git *gitlab.Client, project *gitlab.Project, job *gitlab.Job, hitMeta HitMetaInfo) {
	reader, _, err := git.Jobs.GetTraceFile(project.ID, job.ID)
	if err != nil {
		log.Error().Stack().Err(err).Msg("Failed fetching job trace")
		return
	}
	trace, err := io.ReadAll(reader)
	if err != nil {
		log.Error().Stack().Err(err).Msg("Failed reading trace reader into byte array")
		return
	}
	enqueueItem(trace, queue, QueueItemJobTrace, hitMeta)
}

func getJobArtifacts(git *gitlab.Client, project *gitlab.Project, job *gitlab.Job, options *ScanOptions, hitMeta HitMetaInfo) {
	log.Debug().Str("url", getJobUrl(git, project, job)).Msg("Check for artifacts")

	artifactsReader, _, err := git.Jobs.GetJobArtifacts(project.ID, job.ID)
	if err != nil {
		return
	}

	if artifactsReader.Size() > options.MaxArtifactSize {
		log.Debug().Str("url", getJobUrl(git, project, job)).Int64("bytes", artifactsReader.Size()).Int64("maxBytes", options.MaxArtifactSize).Msg("Skipped large artifact Zip")
		return
	}

	data, err := io.ReadAll(artifactsReader)
	if err != nil {
		log.Error().Int("projectId", project.ID).Int("jobId", job.ID).Msg("Failed reading artifacts stream")
		return
	}

	enqueueItem(data, queue, QueueItemArtifact, hitMeta)

	if len(options.GitlabCookie) > 1 {
		envTxt := DownloadEnvArtifact(options.GitlabCookie, options.GitlabUrl, project.PathWithNamespace, job.ID)
		enqueueItem(envTxt, queue, QueueItemDotenv, hitMeta)
	} else {
		log.Debug().Msg("No cookie provided skipping .env.gz artifact")
	}

}

func getJobUrl(git *gitlab.Client, project *gitlab.Project, job *gitlab.Job) string {
	return git.BaseURL().Host + "/" + project.PathWithNamespace + "/-/jobs/" + strconv.Itoa(job.ID)
}

// .env artifacts are not accessible over the API thus we must use session cookie and use the UI path
// however this is where the treasure is - my precious
func DownloadEnvArtifact(cookieVal string, gitlabUrl string, prjectPath string, jobId int) []byte {

	dotenvUrl, _ := url.JoinPath(gitlabUrl, prjectPath, "/-/jobs/", strconv.Itoa(jobId), "/artifacts/download")

	req, err := http.NewRequest("GET", dotenvUrl, nil)
	if err != nil {
		log.Debug().Stack().Err(err)
		return []byte{}
	}

	q := req.URL.Query()
	q.Add("file_type", "dotenv")
	req.URL.RawQuery = q.Encode()

	req.AddCookie(&http.Cookie{Name: "_gitlab_session", Value: cookieVal})

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		log.Debug().Stack().Err(err).Msg("Failed requesting dotenv artifact")
		return []byte{}
	}
	defer resp.Body.Close()

	statCode := resp.StatusCode

	// means no dotenv exists
	if statCode == 404 {
		return []byte{}
	}

	if statCode != 200 {
		log.Error().Stack().Err(err).Int("HTTP", statCode).Msg("Invalid _gitlab_session detected")
		return []byte{}
	} else {
		log.Debug().Msg("Checking .env.gz artifact")
	}

	body, err := io.ReadAll(resp.Body)

	reader := bytes.NewReader(body)
	gzreader, e1 := gzip.NewReader(reader)
	if e1 != nil {
		log.Debug().Msg(err.Error())
		return []byte{}
	}

	envText, err := io.ReadAll(gzreader)
	if err != nil {
		log.Debug().Stack().Err(err)
		return []byte{}
	}

	return envText
}

func SessionValid(gitlabUrl string, cookieVal string) {
	gitlabSessionsUrl, _ := url.JoinPath(gitlabUrl, "-/user_settings/active_sessions")

	req, err := http.NewRequest("GET", gitlabSessionsUrl, nil)
	if err != nil {
		log.Fatal().Stack().Err(err).Msg("Failed GitLab sessions request")
		return
	}
	req.AddCookie(&http.Cookie{Name: "_gitlab_session", Value: cookieVal})
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		log.Fatal().Stack().Err(err).Msg("Failed GitLab session test")
	}
	defer resp.Body.Close()

	statCode := resp.StatusCode

	if statCode != 200 {
		log.Fatal().Int("http", statCode).Msg("Negative _gitlab_session test")
	} else {
		log.Info().Msg("Provided GitLab session cookie is valid")
	}
}

type runnerResult struct {
	runner  *gitlab.Runner
	project *gitlab.Project
	group   *gitlab.Group
}

func ListAllAvailableRunners(gitlabUrl string, apiToken string) {
	git, err := gitlab.NewClient(apiToken, gitlab.WithBaseURL(gitlabUrl))
	if err != nil {
		log.Fatal().Stack().Err(err)
	}
	runnerMap := make(map[int]runnerResult)
	runnerMap = listProjectRunners(git, runnerMap)
	runnerMap = listGroupRunners(git, runnerMap)

	log.Info().Msg("Listing avaialable runenrs: Runners are only shown once, even when available by multiple source e,g, group or project")

	for _, entry := range runnerMap {
		details, _, err := git.Runners.GetRunnerDetails(entry.runner.ID)
		if err != nil {
			log.Error().Stack().Err(err)
			continue
		}

		if entry.project != nil {
			log.Info().Str("project", entry.project.Name).Str("runner", details.Name).Str("description", details.Description).Str("type", details.RunnerType).Bool("paused", details.Paused).Str("tags", strings.Join(details.TagList, ",")).Msg("project runner")
		}

		if entry.group != nil {
			log.Info().Str("name", entry.group.Name).Str("runner", details.Name).Str("description", details.Description).Str("type", details.RunnerType).Bool("paused", details.Paused).Str("tags", strings.Join(details.TagList, ",")).Msg("group runner")
		}

	}
}

func listProjectRunners(git *gitlab.Client, runnerMap map[int]runnerResult) map[int]runnerResult {
	projectOpts := &gitlab.ListProjectsOptions{
		ListOptions: gitlab.ListOptions{
			PerPage: 100,
			Page:    1,
		},
		MinAccessLevel: gitlab.Ptr(gitlab.MaintainerPermissions),
	}

	for {
		projects, resp, err := git.Projects.ListProjects(projectOpts)
		if err != nil {
			log.Error().Stack().Err(err).Msg("Failed fetching projects")
		}

		for _, project := range projects {
			log.Debug().Str("name", project.Name).Int("id", project.ID).Msg("List runners for")
			runnerOpts := &gitlab.ListProjectRunnersOptions{
				ListOptions: gitlab.ListOptions{
					PerPage: 100,
					Page:    1,
				},
			}
			runners, _, _ := git.Runners.ListProjectRunners(project.ID, runnerOpts)
			for _, runner := range runners {
				runnerMap[runner.ID] = runnerResult{runner: runner, project: project, group: nil}
			}
		}

		if resp.NextPage == 0 {
			break
		}
		projectOpts.Page = resp.NextPage
	}

	return runnerMap

}

func listGroupRunners(git *gitlab.Client, runnerMap map[int]runnerResult) map[int]runnerResult {
	log.Debug().Msg("Logging available groups with at least developer access")

	listGroupsOpts := &gitlab.ListGroupsOptions{
		ListOptions: gitlab.ListOptions{
			PerPage: 100,
			Page:    1,
		},
		AllAvailable:   gitlab.Ptr(true),
		MinAccessLevel: gitlab.Ptr(gitlab.DeveloperPermissions),
	}

	var availableGroups []*gitlab.Group

	for {
		groups, resp, err := git.Groups.ListGroups(listGroupsOpts)
		if err != nil {
			log.Error().Stack().Err(err)
		}

		for _, group := range groups {
			log.Debug().Str("name", group.Name).Msg("List runners for")
			availableGroups = append(availableGroups, group)
		}

		if resp.NextPage == 0 {
			break
		}
		listGroupsOpts.Page = resp.NextPage
	}

	listRunnerOpts := &gitlab.ListGroupsRunnersOptions{
		ListOptions: gitlab.ListOptions{
			PerPage: 100,
			Page:    1,
		},
	}

	for _, group := range availableGroups {
		for {
			runners, resp, err := git.Runners.ListGroupsRunners(group.ID, listRunnerOpts)
			if err != nil {
				log.Error().Stack().Err(err)
			}
			for _, runner := range runners {
				runnerMap[runner.ID] = runnerResult{runner: runner, project: nil, group: group}
			}

			if resp.NextPage == 0 {
				break
			}
			listRunnerOpts.Page = resp.NextPage
		}
	}

	return runnerMap
}

func DetermineVersion(gitlabUrl string, apiToken string) *gitlab.Version {
	if len(apiToken) > 0 {
		git, err := gitlab.NewClient(apiToken, gitlab.WithBaseURL(gitlabUrl))
		if err != nil {
			return &gitlab.Version{Version: "none", Revision: "none"}
		}

		version, _, err := git.Version.GetVersion()
		if err != nil {
			return &gitlab.Version{Version: "none", Revision: "none"}
		}
		return version
	} else {
		u, err := url.Parse(gitlabUrl)
		if err != nil {
			return &gitlab.Version{Version: "none", Revision: "none"}
		}
		u.Path = path.Join(u.Path, "/help")

		tr := &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		}
		client := &http.Client{Transport: tr, Timeout: 15 * time.Second}
		response, err := client.Get(u.String())

		if err != nil {
			log.Warn().Msg(gitlabUrl)
			return &gitlab.Version{Version: "none", Revision: "none"}
		}

		responseData, err := io.ReadAll(response.Body)
		if err != nil {
			return &gitlab.Version{Version: "none", Revision: "none"}
		}

		extractLineR := regexp.MustCompile(`instance_version":"\d*.\d*.\d*"`)
		fullLine := extractLineR.Find(responseData)
		versionR := regexp.MustCompile(`\d+.\d+.\d+`)
		versionNumber := versionR.Find(fullLine)

		if len(versionNumber) > 3 {
			return &gitlab.Version{Version: string(versionNumber), Revision: "none"}
		}
		return &gitlab.Version{Version: "none", Revision: "none"}
	}
}

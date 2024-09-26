package scanner

import (
	"bytes"
	"compress/gzip"
	"context"
	"database/sql"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
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

	setupQueue(tmpfile.Name(), options.MaxScanGoRoutines)
	helper.RegisterGracefulShutdownHandler(cleanUp)

	r := jobs.NewRunner(jobs.NewRunnerOpts{
		Limit:        options.MaxScanGoRoutines,
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

func setupQueue(fileName string, maxReceive int) {
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
		MaxReceive: maxReceive,
	})
}

func cleanUp() {
	log.Info().Msg("Graceful Shutdown, cleaning up")
	files, err := filepath.Glob(queueFileName + "*")
	if err != nil {
		log.Fatal().Err(err).Msg("Error removing database files")
	}
	for _, f := range files {
		err := os.Remove(f)
		if err != nil {
			log.Fatal().Err(err).Str("file", f).Msg("Error deleting database file")
		}
		log.Debug().Str("file", f).Msg("Deleted")
	}
	os.Remove(queueFileName)
}

func fetchProjects(options *ScanOptions) {
	log.Info().Msg("Fetching projects")

	git, err := gitlab.NewClient(options.GitlabApiToken, gitlab.WithBaseURL(options.GitlabUrl))
	if err != nil {
		log.Fatal().Stack().Err(err)
	}

	if len(options.GitlabCookie) > 0 {
		helper.CookieSessionValid(options.GitlabUrl, options.GitlabCookie)
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
				getDotenvArtifact(git, project, job, options, hitMeta)
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

	extractedZipSize := helper.CalculateZipFileSize(data)
	if extractedZipSize > uint64(options.MaxArtifactSize) {
		log.Debug().Str("url", getJobUrl(git, project, job)).Int64("zipBytes", artifactsReader.Size()).Uint64("bytesExtracted", extractedZipSize).Int64("maxBytes", options.MaxArtifactSize).Msg("Skipped large extracted Zip artifact")
		return
	}

	if len(data) > 1 {
		enqueueItem(data, queue, QueueItemArtifact, hitMeta)
	}
}

// dotenv artifacts are not listed in the API thus a request must always be made
func getDotenvArtifact(git *gitlab.Client, project *gitlab.Project, job *gitlab.Job, options *ScanOptions, hitMeta HitMetaInfo) {
	if len(options.GitlabCookie) > 1 {
		envTxt := DownloadEnvArtifact(options.GitlabCookie, options.GitlabUrl, project.PathWithNamespace, job.ID)
		if len(envTxt) > 1 {
			enqueueItem(envTxt, queue, QueueItemDotenv, hitMeta)
		}
	}
}

func getJobUrl(git *gitlab.Client, project *gitlab.Project, job *gitlab.Job) string {
	return git.BaseURL().Host + "/" + project.PathWithNamespace + "/-/jobs/" + strconv.Itoa(job.ID)
}

// .env artifacts are not accessible over the API thus we must use session cookie and use the UI path
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

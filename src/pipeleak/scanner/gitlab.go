package scanner

import (
	"archive/zip"
	"bytes"
	"compress/gzip"
	"context"
	"crypto/tls"
	"io"
	"net/http"
	"net/url"
	"path"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/h2non/filetype"
	"github.com/rs/zerolog/log"
	"github.com/wandb/parallel"
	"github.com/xanzy/go-gitlab"
)

func ScanGitLabPipelines(gitlabUrl string, apiToken string, cookie string, scanArtifacts bool, scanOwnedOnly bool, query string, jobLimit int, member bool, confidenceFilter []string) {
	GetRules(confidenceFilter)
	log.Info().Msg("Fetching projects")
	git, err := gitlab.NewClient(apiToken, gitlab.WithBaseURL(gitlabUrl))
	if err != nil {
		log.Fatal().Stack().Err(err)
	}

	if len(query) > 0 {
		log.Info().Str("query", query).Msg("Filtering scanned projects by")
	}

	projectOpts := &gitlab.ListProjectsOptions{
		ListOptions: gitlab.ListOptions{
			PerPage: 100,
			Page:    1,
		},
		Owned:      gitlab.Ptr(scanOwnedOnly),
		Membership: gitlab.Ptr(member),
		Search:     gitlab.Ptr(query),
		OrderBy:    gitlab.Ptr("last_activity_at"),
	}

	for {
		projects, resp, err := git.Projects.ListProjects(projectOpts)

		// regularily test cookie liveness
		if len(cookie) > 0 {
			SessionValid(gitlabUrl, cookie)
		}

		if err != nil {
			log.Error().Stack().Err(err).Msg("Failed fetching projects")
		}

		for _, project := range projects {
			log.Debug().Str("name", project.Name).Msg("Scan Project jobs for")
			getAllJobs(git, project, scanArtifacts, cookie, gitlabUrl, jobLimit)
		}

		if resp.NextPage == 0 {
			break
		}
		projectOpts.Page = resp.NextPage
		log.Info().Int("total", projectOpts.Page*projectOpts.PerPage).Msg("Scanned projects")
	}
}

func getAllJobs(git *gitlab.Client, project *gitlab.Project, scanArtifacts bool, cookie string, gitlabUrl string, jobLimit int) {

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
			getJobTrace(git, project, job)

			if scanArtifacts {
				getJobArtifacts(git, project, job, cookie, gitlabUrl)
			}

			if jobLimit > 0 && currentJobCtr >= jobLimit {
				log.Debug().Msg("Skipping jobs as job-limit is reached")
				break jobOut
			}
		}

		if resp.NextPage == 0 {
			break
		}

		opts.Page = resp.NextPage
	}

}

func getJobTrace(git *gitlab.Client, project *gitlab.Project, job *gitlab.Job) {
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
	findings := DetectHits(trace)

	for _, finding := range findings {
		log.Warn().Str("confidence", finding.Pattern.Pattern.Confidence).Str("name", finding.Pattern.Pattern.Name).Str("value", finding.Text).Str("url", getJobUrl(git, project, job)).Msg("HIT")
	}
}

func getJobArtifacts(git *gitlab.Client, project *gitlab.Project, job *gitlab.Job, cookie string, gitlabUrl string) {
	log.Debug().Int("projectId", project.ID).Int("jobId", job.ID).Msg("extract artifacts")

	artifactsReader, _, err := git.Jobs.GetJobArtifacts(project.ID, job.ID)
	if err != nil {
		return
	}

	zipListing, err := zip.NewReader(artifactsReader, artifactsReader.Size())
	if err != nil {
		log.Warn().Int("project", project.ID).Int("job", job.ID).Msg("Unable to unzip artifacts for")
		return
	}

	for _, file := range zipListing.File {
		ctx := context.Background()
		group := parallel.Unlimited(ctx)
		group.Go(func(ctx context.Context) {
			fc, err := file.Open()
			if err != nil {
				log.Error().Stack().Err(err).Msg("Unable to open raw artifact zip file")
				return
			}

			content, err := io.ReadAll(fc)
			if err != nil {
				log.Error().Stack().Err(err).Msg("Unable to readAll artifact zip file")
				return
			}

			kind, _ := filetype.Match(content)
			// do not scan https://pkg.go.dev/github.com/h2non/filetype#readme-supported-types
			if kind == filetype.Unknown {
				findings := DetectHits(content)
				for _, finding := range findings {
					log.Warn().Str("confidence", finding.Pattern.Pattern.Confidence).Str("name", finding.Pattern.Pattern.Name).Str("value", finding.Text).Str("url", job.WebURL).Str("file", file.Name).Msg("HIT Artifact")
				}
			}
			fc.Close()
		})
	}

	zipListing = &zip.Reader{}
	artifactsReader = &bytes.Reader{}

	if len(cookie) > 1 {
		envTxt := DownloadEnvArtifact(cookie, gitlabUrl, project.PathWithNamespace, job.ID)
		findings := DetectHits(envTxt)
		artifactsBaseUrl, _ := url.JoinPath(project.WebURL, "/-/artifacts")
		for _, finding := range findings {
			log.Warn().Str("confidence", finding.Pattern.Pattern.Confidence).Str("name", finding.Pattern.Pattern.Name).Str("value", finding.Text).Str("artifactUrl", artifactsBaseUrl).Int("jobId", job.ID).Msg("HIT DOTENV: Check artifacts page which is the only place to download the dotenv file")
		}

	} else {
		log.Debug().Msg("No cookie provided skipping .env.gz artifact")
	}

}

func getJobUrl(git *gitlab.Client, project *gitlab.Project, job *gitlab.Job) string {
	return git.BaseURL().Host + "/" + project.PathWithNamespace + "/-/jobs/" + strconv.Itoa(job.ID)
}

func StreamToString(stream io.Reader) string {
	buf := new(bytes.Buffer)
	_, err := buf.ReadFrom(stream)
	if err != nil {
		log.Error().Stack().Err(err).Msg("Unable to read job trace buffer")
		return ""
	}
	return buf.String()
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

func ListAllAvailableRunners(gitlabUrl string, apiToken string) {
	git, err := gitlab.NewClient(apiToken, gitlab.WithBaseURL(gitlabUrl))
	if err != nil {
		log.Fatal().Stack().Err(err)
	}

	log.Info().Msg("Logging available groups with at least developer access")

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
			log.Info().Str("name", group.Name).Str("fullName", group.FullName).Int("groupId", group.ID).Str("url", group.WebURL)
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
				if runner.Active {
					details, _, err := git.Runners.GetRunnerDetails(runner.ID)
					if err != nil {
						log.Error().Stack().Err(err)
						continue
					}
					log.Info().Str("name", group.Name).Str("runner", details.Name).Str("description", details.Description).Str("type", details.RunnerType).Bool("paused", details.Paused).Str("tags", strings.Join(details.TagList, ","))
				}
			}

			if resp.NextPage == 0 {
				break
			}
			listRunnerOpts.Page = resp.NextPage
		}
	}
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

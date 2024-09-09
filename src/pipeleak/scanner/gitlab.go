package scanner

import (
	"archive/zip"
	"bytes"
	"compress/gzip"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/h2non/filetype"
	"github.com/rs/zerolog/log"
	"github.com/xanzy/go-gitlab"
)

func ScanGitLabPipelines(gitlabUrl string, apiToken string, cookie string, scanArtifacts bool, scanOwnedOnly bool, query string, jobLimit int, member bool) {
	log.Info().Msg("Fetching projects")
	git, err := gitlab.NewClient(apiToken, gitlab.WithBaseURL(gitlabUrl))
	if err != nil {
		log.Fatal().Msg(err.Error())
	}

	if len(query) > 0 {
		log.Info().Msg("Filtering scanned projects by query: " + query)
	}

	projectOpts := &gitlab.ListProjectsOptions{
		ListOptions: gitlab.ListOptions{
			PerPage: 100,
			Page:    1,
		},
		Owned:            gitlab.Ptr(scanOwnedOnly),
		Membership:       gitlab.Ptr(member),
		Search:           gitlab.Ptr(query),
		OrderBy:          gitlab.Ptr("last_activity_at"),
		SearchNamespaces: gitlab.Ptr(true),
	}

	for {
		projects, resp, err := git.Projects.ListProjects(projectOpts)

		// regularily test cookie liveness
		if len(cookie) > 0 {
			SessionValid(gitlabUrl, cookie)
		}

		if err != nil {
			log.Error().Msg(err.Error())
		}

		for _, project := range projects {
			log.Debug().Msg("Scan Project jobs: " + project.Name)
			getAllJobs(git, project, scanArtifacts, cookie, gitlabUrl, jobLimit)
		}

		if resp.NextPage == 0 {
			break
		}
		projectOpts.Page = resp.NextPage
		log.Info().Msg("Scanned projects: " + strconv.Itoa(projectOpts.Page*projectOpts.PerPage) + " of total: " + strconv.Itoa(resp.TotalPages*projectOpts.PerPage))
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
			log.Debug().Msg("Failed fetching project jobs " + err.Error())
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
		log.Error().Msg("Failed fetching job trace with: " + err.Error())
		return
	}
	trace, err := io.ReadAll(reader)
	if err != nil {
		log.Error().Msg("Failed reading trace reader into byte array: " + err.Error())
		return
	}
	findings := DetectHits(trace)

	for _, finding := range findings {
		log.Warn().Msg("HIT Confidence: " + finding.Pattern.Pattern.Confidence + " Name:" + finding.Pattern.Pattern.Name + " Value: " + finding.Text + " URL: " + getJobUrl(git, project, job))
	}
}

func getJobArtifacts(git *gitlab.Client, project *gitlab.Project, job *gitlab.Job, cookie string, gitlabUrl string) {
	log.Debug().Msg("extract artifacts for proj " + strconv.Itoa(project.ID) + " job " + strconv.Itoa(job.ID))

	artifactsReader, _, err := git.Jobs.GetJobArtifacts(project.ID, job.ID)
	if err != nil {
		return
	}

	zipListing, err := zip.NewReader(artifactsReader, artifactsReader.Size())
	if err != nil {
		log.Warn().Msg("Unable to unzip artifacts for proj " + strconv.Itoa(project.ID) + " job " + strconv.Itoa(job.ID))
		return
	}

	for _, file := range zipListing.File {
		fc, err := file.Open()
		if err != nil {
			log.Error().Msg("Unable to openRaw artifact zip file: " + err.Error())
			break
		}

		content, err := io.ReadAll(fc)
		if err != nil {
			log.Error().Msg("Unable to readAll artifact zip file: " + err.Error())
			break
		}

		kind, _ := filetype.Match(content)
		// do not scan https://pkg.go.dev/github.com/h2non/filetype#readme-supported-types
		if kind == filetype.Unknown {
			findings := DetectHits(content)
			for _, finding := range findings {
				log.Warn().Msg("HIT Artifact Confidence: " + finding.Pattern.Pattern.Confidence + " Name:" + finding.Pattern.Pattern.Name + " Value: " + finding.Text + " " + job.WebURL + " in file: " + file.Name)
			}
		} else {
			log.Debug().Msg("Skipping non-text artifact file scan for " + file.Name)
		}
		fc.Close()
	}

	zipListing = &zip.Reader{}
	artifactsReader = &bytes.Reader{}

	if len(cookie) > 1 {
		envTxt := DownloadEnvArtifact(cookie, gitlabUrl, project.PathWithNamespace, job.ID)
		findings := DetectHits(envTxt)
		artifactsBaseUrl, _ := url.JoinPath(project.WebURL, "/-/artifacts")
		for _, finding := range findings {
			log.Warn().Msg("HIT DOTENV Confidence: " + finding.Pattern.Pattern.Confidence + " Name:" + finding.Pattern.Pattern.Name + " Value: " + finding.Text + " Check artifacts page which is the only place to download the dotenv file jobId: " + strconv.Itoa(job.ID) + ": " + artifactsBaseUrl)
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
		log.Error().Msg("Unable to read job trace buffer: " + err.Error())
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
		log.Debug().Msg(err.Error())
		return []byte{}
	}

	q := req.URL.Query()
	q.Add("file_type", "dotenv")
	req.URL.RawQuery = q.Encode()

	req.AddCookie(&http.Cookie{Name: "_gitlab_session", Value: cookieVal})

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		log.Debug().Msg("Failed requesting dotenv artifact with: " + err.Error())
		return []byte{}
	}
	defer resp.Body.Close()

	statCode := resp.StatusCode

	// means no dotenv exists
	if statCode == 404 {
		return []byte{}
	}

	if statCode != 200 {
		log.Error().Msg("Invalid _gitlab_session detected, HTTP " + strconv.Itoa(statCode))
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
		log.Debug().Msg(err.Error())
		return []byte{}
	}

	return envText
}

func SessionValid(gitlabUrl string, cookieVal string) {
	gitlabSessionsUrl, _ := url.JoinPath(gitlabUrl, "-/user_settings/active_sessions")

	req, err := http.NewRequest("GET", gitlabSessionsUrl, nil)
	if err != nil {
		log.Fatal().Msg("Failed GitLab sessions request with: " + err.Error())
		return
	}
	req.AddCookie(&http.Cookie{Name: "_gitlab_session", Value: cookieVal})
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		log.Fatal().Msg("Failed GitLab session test with: " + err.Error())
	}
	defer resp.Body.Close()

	statCode := resp.StatusCode

	if statCode != 200 {
		log.Fatal().Msg("Negative _gitlab_session test, HTTP " + strconv.Itoa(statCode))
	} else {
		log.Info().Msg("Provided GitLab session cookie is valid")
	}
}

func ListAllAvailableRunners(gitlabUrl string, apiToken string) {
	git, err := gitlab.NewClient(apiToken, gitlab.WithBaseURL(gitlabUrl))
	if err != nil {
		log.Fatal().Msg(err.Error())
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
			log.Error().Msg(err.Error())
		}

		for _, group := range groups {
			log.Info().Msg("Group name: " + group.Name + " | full name: " + group.FullName + " | group id: " + strconv.Itoa(group.ID) + " | web url: " + group.WebURL)
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
				log.Error().Msg(err.Error())
			}
			for _, runner := range runners {
				if runner.Active {
					details, _, err := git.Runners.GetRunnerDetails(runner.ID)
					if err != nil {
						log.Error().Msg(err.Error())
						continue
					}
					log.Info().Msg("Group " + group.Name + " Runner name: " + details.Name + " | description: " + details.Description + " | type: " + details.RunnerType + " | paused: " + strconv.FormatBool(details.Paused) + " tags: " + strings.Join(details.TagList, ","))
				}
			}

			if resp.NextPage == 0 {
				break
			}
			listRunnerOpts.Page = resp.NextPage
		}
	}
}

package scanner

import (
	"archive/zip"
	"bytes"
	"compress/gzip"
	"io"
	"net/http"
	"net/url"
	"strconv"

	"github.com/h2non/filetype"
	"github.com/rs/zerolog/log"
	"github.com/xanzy/go-gitlab"
)

func ScanGitLabPipelines(gitlabUrl string, apiToken string, cookie string, scanArtifacts bool, scanOwnedOnly bool, query string) {
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
		Owned:  gitlab.Ptr(scanOwnedOnly),
		Search: gitlab.Ptr(query),
	}

	for {
		projects, resp, err := git.Projects.ListProjects(projectOpts)

		if err != nil {
			log.Error().Msg(err.Error())
		}

		for _, project := range projects {
			log.Debug().Msg("Scan Project jobs: " + project.Name)
			getAllJobs(git, project, scanArtifacts, cookie, gitlabUrl)
		}

		if resp.NextPage == 0 {
			break
		}
		projectOpts.Page = resp.NextPage
		log.Info().Msg("Scanned projects: " + strconv.Itoa(projectOpts.Page*projectOpts.PerPage))
	}
}

func getAllJobs(git *gitlab.Client, project *gitlab.Project, scanArtifacts bool, cookie string, gitlabUrl string) {

	opts := &gitlab.ListJobsOptions{
		ListOptions: gitlab.ListOptions{
			PerPage: 100,
			Page:    1,
		},
	}

	for {
		jobs, resp, err := git.Jobs.ListProjectJobs(project.ID, opts)

		if err != nil {
			log.Debug().Msg("Failed fetching project jobs " + err.Error())
		}

		for _, job := range jobs {
			getJobTrace(git, project, job)

			if scanArtifacts {
				getJobArtifacts(git, project, job, cookie, gitlabUrl)
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
		log.Error().Msg(err.Error())
	}
	trace := StreamToString(reader)
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

	}

	for _, file := range zipListing.File {
		fc, err := file.Open()
		if err != nil {
			log.Error().Msg("Unable to openRaw artifact zip file: " + err.Error())
		}

		content, err := io.ReadAll(fc)
		if err != nil {
			log.Error().Msg("Unable to readAll artifact zip file: " + err.Error())
		}

		kind, _ := filetype.Match(content)
		// do not scan https://pkg.go.dev/github.com/h2non/filetype#readme-supported-types
		if kind == filetype.Unknown {
			findings := DetectHits(string(content))
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
			log.Warn().Msg("HIT .ENV Confidence: " + finding.Pattern.Pattern.Confidence + " Name:" + finding.Pattern.Pattern.Name + " Value: " + finding.Text + " Check artifacts page which is the only place to download the dotenv file jobId: " + strconv.Itoa(job.ID) + ": " + artifactsBaseUrl)
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
	}
	return buf.String()
}

// .env artifacts are not accessible over the API thus we must use session cookie and use the UI path
// however this is where the treasure is - my precious
func DownloadEnvArtifact(cookieVal string, gitlabUrl string, prjectPath string, jobId int) string {

	dotenvUrl, _ := url.JoinPath(gitlabUrl, prjectPath, "/-/jobs/", strconv.Itoa(jobId), "/artifacts/download")

	req, err := http.NewRequest("GET", dotenvUrl, nil)
	if err != nil {
		log.Debug().Msg(err.Error())
	}

	q := req.URL.Query()
	q.Add("file_type", "dotenv")
	req.URL.RawQuery = q.Encode()

	req.AddCookie(&http.Cookie{Name: "_gitlab_session", Value: cookieVal})

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		log.Debug().Msg("Failed requesting dotenv artifact with: " + err.Error())
	}
	defer resp.Body.Close()

	statCode := resp.StatusCode

	// means no dotenv exists
	if statCode == 404 {
		return ""
	}

	if statCode != 200 {
		log.Error().Msg("Invalid _gitlab_session detected, HTTP " + strconv.Itoa(statCode))
		return ""
	} else {
		log.Debug().Msg("Checking .env.gz artifact")
	}

	body, err := io.ReadAll(resp.Body)
	resp.Body.Close()

	reader := bytes.NewReader(body)
	gzreader, e1 := gzip.NewReader(reader)
	if e1 != nil {
		log.Debug().Msg(err.Error())
		return ""
	}

	envText, err := io.ReadAll(gzreader)
	if err != nil {
		log.Debug().Msg(err.Error())
		return ""
	}

	return string(envText)
}

package scanner

import (
	"archive/zip"
	"bufio"
	"bytes"
	"compress/gzip"
	"io"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"unicode/utf8"

	"github.com/rs/zerolog/log"
	"github.com/xanzy/go-gitlab"
)

func ScanGitLabPipelines(gitlabUrl string, apiToken string, cookie string, scanArtifacts bool, scanOwnedOnly bool) {
	log.Info().Msg("Fetching projects")
	git, err := gitlab.NewClient(apiToken, gitlab.WithBaseURL(gitlabUrl))
	if err != nil {
		log.Fatal().Msg(err.Error())
	}

	owned := &[]bool{false}[0]
	if scanOwnedOnly {
		log.Info().Msg("Scanning only owend projects")
		owned = &[]bool{true}[0]
	}

	projectOpts := &gitlab.ListProjectsOptions{
		ListOptions: gitlab.ListOptions{
			PerPage: 100,
			Page:    1,
		},
		Owned: owned,
	}

	log.Info().Msg("Start scanning pipeline jobs")
	for {
		projects, resp, err := git.Projects.ListProjects(projectOpts)
		log.Info().Msg("Scanned projects: " + strconv.Itoa(projectOpts.Page*projectOpts.PerPage))

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

	dir, err := os.MkdirTemp("", "pipeleak")
	if err != nil {
		log.Error().Msg(err.Error())
	}
	defer os.RemoveAll(dir)

	log.Debug().Msg("extracting artifacts")

	zipListing, err := zip.NewReader(artifactsReader, artifactsReader.Size())
	if err != nil {
		log.Warn().Msg("Unable to unzip artifacts for proj " + strconv.Itoa(project.ID) + " job " + strconv.Itoa(job.ID))

	}

	for _, file := range zipListing.File {
		fc, err := file.Open()
		if err != nil {
			log.Error().Msg("Unable to openRaw artifact zip file: " + err.Error())
		}

		if isFileTextBased(fc) {
			content := readZipFile(file)
			if err != nil {
				log.Error().Msg(err.Error())
			}

			findings := DetectHits(string(content))
			for _, finding := range findings {
				log.Warn().Msg("HIT Artifact Confidence: " + finding.Pattern.Pattern.Confidence + " Name:" + finding.Pattern.Pattern.Name + " Value: " + finding.Text + " " + job.WebURL + " in file: " + file.Name)
			}
		} else {
			log.Debug().Msg("Skipping non-text artifact file scan for " + file.Name)
		}
	}

	if len(cookie) > 1 {
		log.Debug().Msg("Checking .env.gz artifact")
		envTxt := DownloadEnvArtifact(cookie, gitlabUrl, project.PathWithNamespace, job.ID)
		if err != nil {
			log.Error().Msg(err.Error())
		}

		findings := DetectHits(envTxt)
		artifactsBaseUrl, _ := url.JoinPath(project.WebURL, "/-/artifacts")
		for _, finding := range findings {
			log.Warn().Msg("HIT .ENV Confidence: " + finding.Pattern.Pattern.Confidence + " Name:" + finding.Pattern.Pattern.Name + " Value: " + finding.Text + " Check artifacts page which is the only place to download the dotenv file jobId: " + strconv.Itoa(job.ID) + ": " + artifactsBaseUrl)
		}

	} else {
		log.Debug().Msg("No cookie provided skipping .env.gz artifact")
	}

}

func isFileTextBased(file io.Reader) bool {
	fileScanner := bufio.NewScanner(file)
	fileScanner.Split(bufio.ScanLines)
	fileScanner.Scan()
	return utf8.ValidString(string(fileScanner.Text()))
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

func readZipFile(file *zip.File) []byte {
	fc, err := file.Open()
	if err != nil {
		log.Error().Msg("Unable to open artifact zip file: " + err.Error())
	}

	content, err := io.ReadAll(fc)
	if err != nil {
		log.Error().Msg("Unable to readAll artifact zip file: " + err.Error())
	}

	return content
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
		log.Debug().Msg(err.Error())
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return ""
	}

	body, err := io.ReadAll(resp.Body)

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

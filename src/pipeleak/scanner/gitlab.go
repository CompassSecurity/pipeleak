package scanner

import (
	"bytes"
	"io"
	"strconv"

	"github.com/rs/zerolog/log"
	"github.com/xanzy/go-gitlab"
)

func ScanGitLabPipelines(gitlabUrl string, apiToken string) {
	log.Info().Msg("Fetching projects")
	git, err := gitlab.NewClient(apiToken, gitlab.WithBaseURL(gitlabUrl))
	if err != nil {
		log.Fatal().Msg(err.Error())
	}

	projectOpts := &gitlab.ListProjectsOptions{
		ListOptions: gitlab.ListOptions{
			PerPage: 100,
			Page:    1,
		},
	}

	log.Info().Msg("Start scanning pipeline jobs")
	for {
		projects, resp, err := git.Projects.ListProjects(projectOpts)
		log.Info().Msg("Scanned projects: " + strconv.Itoa(projectOpts.Page*projectOpts.PerPage))

		if err != nil {
			log.Fatal().Msg(err.Error())
		}

		for _, project := range projects {
			log.Debug().Msg("Scan Project jobs: " + project.Name)
			getAllJobs(git, project)
		}

		if resp.NextPage == 0 {
			break
		}
		projectOpts.Page = resp.NextPage
	}
}

func getAllJobs(git *gitlab.Client, project *gitlab.Project) {

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

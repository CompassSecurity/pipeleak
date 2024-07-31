package scanner

import (
	"bytes"
	"io"
	"strconv"

	"github.com/rs/zerolog/log"
	"github.com/xanzy/go-gitlab"
)

func ScanGitLabPipelines(gitlabUrl string, apiToken string) {
	log.Info().Msg("Gathering all projects")
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

	allProjects := make(map[int]*gitlab.Project)

	for {
		projects, resp, err := git.Projects.ListProjects(projectOpts)

		if err != nil {
			log.Fatal().Msg(err.Error())
		}

		for _, project := range projects {
			allProjects[project.ID] = project
			log.Debug().Msg("Scan Project jobs: " + project.Name)
			getAllJobs(git, project)
		}

		if resp.NextPage == 0 {
			break
		}

		projectOpts.Page = resp.NextPage
	}

	log.Info().Msg("Enumerated " + strconv.Itoa(len(allProjects)) + " projects to be scanned")

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
			log.Error().Msg("Failed fetching project jobs " + err.Error())
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
		log.Warn().Msg("HIT Confidence: " + finding.Pattern.Pattern.Confidence + " Name:" + finding.Pattern.Pattern.Name + " Value: " + finding.Text)
		log.Warn().Msg("HIT URL: " + git.BaseURL().Host + "/" + project.PathWithNamespace + "/-/jobs/" + strconv.Itoa(job.ID))
	}
}

func StreamToString(stream io.Reader) string {
	buf := new(bytes.Buffer)
	buf.ReadFrom(stream)
	return buf.String()
}

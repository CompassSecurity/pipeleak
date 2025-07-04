package scan

import (
	"os"
	"path/filepath"
	"strconv"
	"sync"

	"github.com/CompassSecurity/pipeleak/cmd/gitlab/util"
	"github.com/CompassSecurity/pipeleak/helper"
	"github.com/CompassSecurity/pipeleak/scanner"
	"github.com/nsqio/go-diskqueue"
	"github.com/rs/zerolog/log"
	gitlab "gitlab.com/gitlab-org/api/client-go"
)

var globQueue diskqueue.Interface
var waitGroup *sync.WaitGroup
var queueFileName string

type ScanOptions struct {
	GitlabUrl              string
	GitlabApiToken         string
	GitlabCookie           string
	ProjectSearchQuery     string
	Artifacts              bool
	Owned                  bool
	Member                 bool
	JobLimit               int
	Verbose                bool
	ConfidenceFilter       []string
	MaxArtifactSize        int64
	MaxScanGoRoutines      int
	QueueFolder            string
	TruffleHogVerification bool
}

func ScanGitLabPipelines(options *ScanOptions) {
	globQueue, queueFileName = setupQueue(options)
	helper.RegisterGracefulShutdownHandler(cleanUp)

	scanner.InitRules(options.ConfidenceFilter)
	if !options.TruffleHogVerification {
		log.Info().Msg("TruffleHog verification is disabled")
	}

	git, err := util.GetGitlabClient(options.GitlabApiToken, options.GitlabUrl)
	if err != nil {
		log.Fatal().Stack().Err(err).Msg("Failed creating gitlab client")
	}

	// waitgroup is used to coordinate termination
	// dont kill the queue before the jobs have been fetched
	waitGroup = new(sync.WaitGroup)
	waitGroup.Add(1)
	go fetchProjects(git, options, waitGroup)

	go func() {
		queueChan := globQueue.ReadChan()
		for qitem := range queueChan {
			analyzeQueueItem(qitem, git, options, waitGroup)
		}
	}()

	waitGroup.Wait()
	cleanUp()
}

func cleanUp() {
	log.Debug().Msg("Cleaning up")
	err := globQueue.Delete()
	if err != nil {
		log.Fatal().Err(err).Msg("Error deleteing queue on shutdown")
	}

	files, err := filepath.Glob(queueFileName + "*")
	if err != nil {
		log.Fatal().Err(err).Msg("Error removing database files")
	}
	for _, f := range files {
		err := os.Remove(f)
		if err != nil {
			log.Fatal().Err(err).Str("file", f).Msg("Error deleting database file")
		}
		log.Trace().Str("file", f).Msg("Deleted")
	}
	os.Remove(queueFileName)
}

func fetchProjects(git *gitlab.Client, options *ScanOptions, wg *sync.WaitGroup) {
	defer wg.Done()
	log.Info().Msg("Fetching projects")

	if len(options.GitlabCookie) > 0 {
		util.CookieSessionValid(options.GitlabUrl, options.GitlabCookie)
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
			break
		}

		for _, project := range projects {
			log.Debug().Str("url", project.WebURL).Msg("Fetch project jobs")
			getAllJobs(git, project, options)
		}

		if resp.NextPage == 0 {
			break
		}
		projectOpts.Page = resp.NextPage
		log.Info().Int("total", projectOpts.Page*projectOpts.PerPage).Msg("Fetched projects")
	}

	log.Info().Msg("Fetched all projects")
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
			break
		}

		if resp.StatusCode == 403 {
			break
		}

		for _, job := range jobs {
			currentJobCtr += 1
			meta := QueueMeta{JobId: job.ID, ProjectId: project.ID, JobWebUrl: getJobUrl(git, project, job), JobName: job.Name, ProjectPathWithNamespace: project.PathWithNamespace}
			enqueueItem(globQueue, QueueItemJobTrace, meta, waitGroup)

			if options.Artifacts {
				enqueueItem(globQueue, QueueItemArtifact, meta, waitGroup)
				if len(options.GitlabCookie) > 1 {
					enqueueItem(globQueue, QueueItemDotenv, meta, waitGroup)
				}
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

func getJobUrl(git *gitlab.Client, project *gitlab.Project, job *gitlab.Job) string {
	return git.BaseURL().Host + "/" + project.PathWithNamespace + "/-/jobs/" + strconv.Itoa(job.ID)
}

func GetQueueStatus() int {
	return int(globQueue.Depth())
}

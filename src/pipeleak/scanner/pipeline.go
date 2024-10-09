package scanner

import (
	"context"
	"database/sql"
	"os"
	"path"
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
	QueueFolder        string
}

func ScanGitLabPipelines(options *ScanOptions) {
	setupQueue(options)
	helper.RegisterGracefulShutdownHandler(cleanUp)

	r := jobs.NewRunner(jobs.NewRunnerOpts{
		Limit:        options.MaxScanGoRoutines,
		Log:          nil,
		PollInterval: 10 * time.Millisecond,
		Queue:        queue,
		Extend:       30 * time.Second,
	})

	InitRules(options.ConfidenceFilter)

	git, err := helper.GetGitlabClient(options.GitlabApiToken, options.GitlabUrl)
	if err != nil {
		log.Fatal().Stack().Err(err).Msg("failed creating gitlab client")
	}

	go fetchProjects(git, options)

	r.Register("pipeleak-job", func(ctx context.Context, m []byte) error {
		analyzeQueueItem(m, git, options)
		return nil
	})

	// the time the queue waits without any items
	ctx, cancel := context.WithTimeout(context.Background(), 80*time.Second)
	defer cancel()
	r.Start(ctx)
}

func setupQueue(options *ScanOptions) {
	log.Debug().Msg("Setting up queue on disk")

	queueDirectory := options.QueueFolder
	if len(options.QueueFolder) > 0 {
		cwd, err := os.Getwd()
		if err != nil {
			log.Fatal().Err(err).Msg("Could not determine CWD")
		}
		relative := path.Join(cwd, queueDirectory)
		absPath, err := filepath.Abs(relative)
		if err != nil {
			log.Fatal().Err(err).Msg("Failed parsing absolute path")
		}
		queueDirectory = absPath
	}

	tmpfile, err := os.CreateTemp(queueDirectory, "pipeleak-queue-db-")
	if err != nil {
		log.Fatal().Err(err).Msg("Creating Temp DB file failed")
	}
	defer os.Remove(tmpfile.Name())
	queueFileName = tmpfile.Name()

	sqlUri := `file://` + queueFileName + `:?_journal=WAL&_timeout=5000&_fk=true`
	db, err := sql.Open("sqlite3", sqlUri)
	log.Debug().Str("file", sqlUri).Msg("Using DB file")
	if err != nil {
		log.Fatal().Err(err).Str("file", queueFileName).Msg("Opening Temp DB file failed")
	}
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)

	if err := goqite.Setup(context.Background(), db); err != nil {
		log.Fatal().Err(err).Msg("Goqite setup failed")
	}

	queue = goqite.New(goqite.NewOpts{
		DB:         db,
		Name:       "jobs",
		MaxReceive: options.MaxScanGoRoutines,
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

func fetchProjects(git *gitlab.Client, options *ScanOptions) {
	log.Info().Msg("Fetching projects")

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
			break
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
		if resp.StatusCode == 403 {
			break
		}

		if err != nil {
			log.Debug().Stack().Err(err).Msg("Failed fetching project jobs")
			break
		}

		for _, job := range jobs {
			currentJobCtr += 1
			meta := QueueMeta{JobId: job.ID, ProjectId: project.ID, JobWebUrl: getJobUrl(git, project, job), ProjectPathWithNamespace: project.PathWithNamespace}
			enqueueItem(queue, QueueItemJobTrace, meta)

			if options.Artifacts {
				enqueueItem(queue, QueueItemArtifact, meta)
				if len(options.GitlabCookie) > 1 {
					enqueueItem(queue, QueueItemDotenv, meta)
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

package scanner

import (
	"archive/zip"
	"bytes"
	"compress/gzip"
	"context"
	"io"
	"net/http"
	"net/url"
	"os"
	"path"
	"strconv"
	"strings"

	"github.com/h2non/filetype"
	"github.com/rs/zerolog/log"
	"github.com/wandb/parallel"
	"github.com/xanzy/go-gitlab"
	"golift.io/xtractr"
)

func ScanGitLabPipelines(gitlabUrl string, apiToken string, cookie string, scanArtifacts bool, scanOwnedOnly bool, query string, jobLimit int, member bool) {
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
				detectFileHits(content, job, file.Name, "")
			} else if filetype.IsArchive(content) {
				log.Debug().Str("file", file.Name).Msg("Archive in artifact Zip Detected")
				handleArchiveArtifact(file.Name, content, job)
			} else {
				log.Debug().Str("file", file.Name).Msg("Skipping non-text artifact")
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

func handleArchiveArtifact(archivefileName string, content []byte, job *gitlab.Job) {
	fileType, err := filetype.Get(content)
	if err != nil {
		log.Error().Stack().Err(err).Msg("Cannot determine file type")
		return
	}

	tmpArchiveFile, err := os.CreateTemp("", "pipeleak-artifact-archive-*."+fileType.Extension)
	if err != nil {
		log.Error().Stack().Err(err).Msg("Cannot create artifact archive temp file")
		return
	}

	err = os.WriteFile(tmpArchiveFile.Name(), content, 0666)
	if err != nil {
		log.Error().Stack().Err(err).Msg("Failed writing archive to disk")
		return
	}
	defer os.Remove(tmpArchiveFile.Name())

	tmpArchiveFilesDirectory, err := os.MkdirTemp("", "pipeleak-artifact-archive-out-")
	if err != nil {
		log.Error().Stack().Err(err).Msg("Cannot create artifact archive temp directory")
		return
	}
	defer os.RemoveAll(tmpArchiveFilesDirectory)

	x := &xtractr.XFile{
		FilePath:  tmpArchiveFile.Name(),
		OutputDir: tmpArchiveFilesDirectory,
		FileMode:  0o600,
		DirMode:   0o700,
	}

	_, files, _, err := xtractr.ExtractFile(x)
	if err != nil || files == nil {
		log.Error().Stack().Err(err).Msg("Unable to handle archive in artifacts")
		return
	}

	for _, fPath := range files {
		if !isDirectory(fPath) {
			fileBytes, err := os.ReadFile(fPath)
			if err != nil {
				log.Error().Str("file", fPath).Stack().Err(err).Msg("Cannot read temp artifact archive file content")
			}

			kind, _ := filetype.Match(fileBytes)
			if kind == filetype.Unknown {
				detectFileHits(fileBytes, job, path.Base(fPath), archivefileName)
			}
		}
	}
}

func isDirectory(path string) bool {
	fileInfo, err := os.Stat(path)
	if err != nil {
		return true
	}

	return fileInfo.IsDir()
}

func detectFileHits(content []byte, job *gitlab.Job, fileName string, archiveName string) {
	findings := DetectHits(content)
	for _, finding := range findings {
		baseLog := log.Warn().Str("confidence", finding.Pattern.Pattern.Confidence).Str("name", finding.Pattern.Pattern.Name).Str("value", finding.Text).Str("url", job.WebURL).Str("file", fileName)
		if len(archiveName) > 0 {
			baseLog.Str("archive", archiveName).Msg("HIT Artifact (in archive)")
		} else {
			baseLog.Msg("HIT Artifact")
		}
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

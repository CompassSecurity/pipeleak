package scanner

import (
	"archive/zip"
	"bytes"
	"compress/gzip"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"runtime"
	"strconv"

	"github.com/CompassSecurity/pipeleak/helper"
	"github.com/h2non/filetype"
	"github.com/maragudk/goqite"
	"github.com/maragudk/goqite/jobs"
	"github.com/rs/zerolog/log"
	"github.com/wandb/parallel"
	"github.com/xanzy/go-gitlab"
)

type QueueItemType string

const (
	QueueItemJobTrace QueueItemType = "jobTrace"
	QueueItemArtifact QueueItemType = "artifact"
	QueueItemDotenv   QueueItemType = "dotenv"
)

type QueueMeta struct {
	ProjectId                int
	JobId                    int
	JobWebUrl                string
	ProjectPathWithNamespace string
}

type QueueItem struct {
	Type        QueueItemType `json:"type"`
	ScanOptions *ScanOptions  `json:"scanOptions"`
	Meta        QueueMeta     `json:"meta"`
}

func analyzeQueueItem(serializeditem []byte, git *gitlab.Client, options *ScanOptions) {
	var item QueueItem
	err := json.Unmarshal(serializeditem, &item)
	if err != nil {
		log.Error().Err(err).Msg("Failed unmarshalling queue item")
	}

	if item.Type == QueueItemJobTrace {
		log.Debug().Str("url", item.Meta.JobWebUrl).Msg("Scanning Job Trace")
		analyzeJobTrace(git, item, options)
	}

	if item.Type == QueueItemArtifact {
		log.Debug().Str("url", item.Meta.JobWebUrl).Msg("Scanning artifact")
		analyzeJobArtifact(git, item, options)
		runtime.GC()
	}

	if item.Type == QueueItemDotenv {
		log.Debug().Str("url", item.Meta.JobWebUrl).Msg("Scanning Dotenv")
		analyzeDotenvArtifact(git, item, options)
	}

}

func enqueueItem(queue *goqite.Queue, qType QueueItemType, meta QueueMeta) {
	item := &QueueItem{Type: qType, Meta: meta}
	itemBytes, err := json.Marshal(item)
	if err != nil {
		log.Error().Str("type", string(qType)).Err(err).Msg("Failed marshalling job item")
		return
	}

	err = jobs.Create(context.Background(), queue, "pipeleak-job", itemBytes)
	if err != nil {
		log.Error().Str("type", string(qType)).Err(err).Msg("Failed queuing job")
	}
}

func analyzeJobTrace(git *gitlab.Client, item QueueItem, options *ScanOptions) {
	trace := getJobTrace(git, item.Meta.ProjectId, item.Meta.JobId)
	if len(trace) < 1 {
		return
	}

	findings := DetectHits(trace, options.MaxScanGoRoutines)
	for _, finding := range findings {
		log.Warn().Str("confidence", finding.Pattern.Pattern.Confidence).Str("name", finding.Pattern.Pattern.Name).Str("value", finding.Text).Str("url", item.Meta.JobWebUrl).Msg("HIT")
	}
}

func analyzeJobArtifact(git *gitlab.Client, item QueueItem, options *ScanOptions) {
	data := getJobArtifacts(git, item.Meta.ProjectId, item.Meta.JobId, options)
	if data == nil {
		return
	}

	reader := bytes.NewReader(data)
	zipListing, err := zip.NewReader(reader, int64(len(data)))
	if err != nil {
		log.Warn().Int("project", item.Meta.ProjectId).Int("job", item.Meta.JobId).Msg("Unable to unzip artifacts for")
		return
	}

	ctx := context.Background()
	group := parallel.Limited(ctx, options.MaxScanGoRoutines)
	for _, file := range zipListing.File {
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
				// use one to prevent maxThreads^2 which trashes memory
				findings := DetectHits(content, 1)
				for _, finding := range findings {
					log.Warn().Str("confidence", finding.Pattern.Pattern.Confidence).Str("name", finding.Pattern.Pattern.Name).Str("value", finding.Text).Str("url", item.Meta.JobWebUrl).Str("file", file.Name).Msg("HIT Artifact")
				}
			}
			fc.Close()
		})
	}

	group.Wait()
}

func analyzeDotenvArtifact(git *gitlab.Client, item QueueItem, options *ScanOptions) {
	dotenvText := getDotenvArtifact(git, item.Meta.ProjectId, item.Meta.JobId, item.Meta.ProjectPathWithNamespace, options)
	if dotenvText == nil {
		return
	}

	findings := DetectHits(dotenvText, options.MaxScanGoRoutines)
	for _, finding := range findings {
		artifactsBaseUrl, _ := url.JoinPath(item.Meta.JobWebUrl, "/-/artifacts")
		log.Warn().Str("confidence", finding.Pattern.Pattern.Confidence).Str("name", finding.Pattern.Pattern.Name).Str("value", finding.Text).Str("artifactUrl", artifactsBaseUrl).Int("jobId", item.Meta.JobId).Msg("HIT DOTENV: Check artifacts page which is the only place to download the dotenv file")
	}
}

func getJobTrace(git *gitlab.Client, projectId int, jobId int) []byte {
	reader, _, err := git.Jobs.GetTraceFile(projectId, jobId)
	if err != nil {
		log.Error().Stack().Err(err).Msg("Failed fetching job trace")
		return nil
	}
	trace, err := io.ReadAll(reader)
	if err != nil {
		log.Error().Stack().Err(err).Msg("Failed reading trace reader into byte array")
		return nil
	}

	return trace
}

func getJobArtifacts(git *gitlab.Client, projectId int, jobId int, options *ScanOptions) []byte {
	artifactsReader, resp, err := git.Jobs.GetJobArtifacts(projectId, jobId)
	if resp.StatusCode == 404 {
		return nil
	}

	if err != nil {
		log.Error().Err(err).Int("project", projectId).Int("job", jobId).Msg("Failed donloading job artifacts zip")
		return nil
	}

	if artifactsReader.Size() > options.MaxArtifactSize {
		log.Debug().Int("project", projectId).Int("job", jobId).Int64("bytes", artifactsReader.Size()).Int64("maxBytes", options.MaxArtifactSize).Msg("Skipped large artifact Zip")
		return nil
	}

	data, err := io.ReadAll(artifactsReader)
	if err != nil {
		log.Error().Err(err).Int("project", projectId).Int("job", jobId).Msg("Failed reading artifacts stream")
		return nil
	}

	extractedZipSize := helper.CalculateZipFileSize(data)
	if extractedZipSize > uint64(options.MaxArtifactSize) {
		log.Debug().Int("project", projectId).Int("job", jobId).Int64("zipBytes", artifactsReader.Size()).Uint64("bytesExtracted", extractedZipSize).Int64("maxBytes", options.MaxArtifactSize).Msg("Skipped large extracted Zip artifact")
		return nil
	}

	if len(data) > 1 {
		return data
	}

	return nil
}

// dotenv artifacts are not listed in the API thus a request must always be made
func getDotenvArtifact(git *gitlab.Client, projectId int, jobId int, projectPathWithNamespace string, options *ScanOptions) []byte {
	if len(options.GitlabCookie) > 1 {
		envTxt := DownloadEnvArtifact(options.GitlabCookie, options.GitlabUrl, projectPathWithNamespace, jobId)
		if len(envTxt) > 1 {
			return envTxt
		}
	}

	return nil
}

// .env artifacts are not accessible over the API thus we must use session cookie and use the UI path
func DownloadEnvArtifact(cookieVal string, gitlabUrl string, prjectPath string, jobId int) []byte {

	dotenvUrl, _ := url.JoinPath(gitlabUrl, prjectPath, "/-/jobs/", strconv.Itoa(jobId), "/artifacts/download")

	req, err := http.NewRequest("GET", dotenvUrl, nil)
	if err != nil {
		log.Debug().Stack().Err(err).Msg("Failed dotenv GET request")
		return []byte{}
	}

	q := req.URL.Query()
	q.Add("file_type", "dotenv")
	req.URL.RawQuery = q.Encode()

	req.AddCookie(&http.Cookie{Name: "_gitlab_session", Value: cookieVal})

	client := helper.GetNonVerifyingHTTPClient()
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
		log.Debug().Stack().Err(err).Msg("failed uncompressing dotenv archive")
		return []byte{}
	}

	return envText
}

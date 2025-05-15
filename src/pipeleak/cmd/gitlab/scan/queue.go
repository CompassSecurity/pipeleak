package scan

import (
	"archive/zip"
	"bytes"
	"compress/gzip"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"sync"
	"time"

	"github.com/CompassSecurity/pipeleak/helper"
	"github.com/CompassSecurity/pipeleak/scanner"
	"github.com/h2non/filetype"
	"github.com/nsqio/go-diskqueue"
	"github.com/rs/zerolog/log"
	"github.com/wandb/parallel"
	gitlab "gitlab.com/gitlab-org/api/client-go"
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
	JobName                  string
	ProjectPathWithNamespace string
}

type QueueItem struct {
	Type        QueueItemType `json:"type"`
	ScanOptions *ScanOptions  `json:"scanOptions"`
	Meta        QueueMeta     `json:"meta"`
}

func setupQueue(options *ScanOptions) (diskqueue.Interface, string) {
	log.Debug().Msg("Setting up queue on disk")

	queueDirectory := options.QueueFolder
	if len(options.QueueFolder) > 0 {
		cwd, err := os.Getwd()
		if err != nil {
			log.Fatal().Err(err).Msg("Could not determine CWD")
		}
		relative := filepath.Join(cwd, queueDirectory)
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

	return diskqueue.New(tmpfile.Name(), queueDirectory, 512, 0, math.MaxInt32, 2500, 2*time.Second, func(lvl diskqueue.LogLevel, f string, args ...interface{}) {
		log.Trace().Msg("Queue Log: " + fmt.Sprintf(lvl.String()+": "+f, args...))
	}), tmpfile.Name()
}

func analyzeQueueItem(serializeditem []byte, git *gitlab.Client, options *ScanOptions, wg *sync.WaitGroup) {
	defer wg.Done()

	var item QueueItem
	err := json.Unmarshal(serializeditem, &item)
	if err != nil {
		log.Error().Err(err).Msg("Failed unmarshalling queue item")
	}

	if item.Type == QueueItemJobTrace {
		analyzeJobTrace(git, item, options)
	}

	if item.Type == QueueItemArtifact {
		analyzeJobArtifact(git, item, options)
		runtime.GC()
	}

	if item.Type == QueueItemDotenv {
		analyzeDotenvArtifact(git, item, options)
	}
}

func enqueueItem(queue diskqueue.Interface, qType QueueItemType, meta QueueMeta, wg *sync.WaitGroup) {
	item := &QueueItem{Type: qType, Meta: meta}
	itemBytes, err := json.Marshal(item)
	if err != nil {
		log.Error().Str("type", string(qType)).Err(err).Msg("Failed marshalling job item")
		return
	}
	err = queue.Put(itemBytes)
	if err != nil {
		log.Error().Str("type", string(qType)).Err(err).Msg("Failed put'ing the queue item")
		return
	}

	wg.Add(1)
}

func analyzeJobTrace(git *gitlab.Client, item QueueItem, options *ScanOptions) {
	trace := getJobTrace(git, item.Meta.ProjectId, item.Meta.JobId)
	if len(trace) < 1 {
		return
	}

	findings, err := scanner.DetectHits(trace, options.MaxScanGoRoutines, options.TruffleHogVerification)
	if err != nil {
		log.Debug().Err(err).Int("project", item.Meta.ProjectId).Int("job", item.Meta.JobId).Msg("Failed detecting secrets")
		return
	}

	for _, finding := range findings {
		log.Warn().Str("confidence", finding.Pattern.Pattern.Confidence).Str("ruleName", finding.Pattern.Pattern.Name).Str("value", finding.Text).Str("url", item.Meta.JobWebUrl).Str("jobName", item.Meta.JobName).Msg("HIT")
	}
}

func analyzeJobArtifact(git *gitlab.Client, item QueueItem, options *ScanOptions) {
	data := getJobArtifacts(git, item.Meta.ProjectId, item.Meta.JobId, item.Meta.JobWebUrl, options)
	if data == nil {
		return
	}

	reader := bytes.NewReader(data)
	zipListing, err := zip.NewReader(reader, int64(len(data)))
	if err != nil {
		log.Debug().Int("project", item.Meta.ProjectId).Int("job", item.Meta.JobId).Msg("Unable to unzip artifacts for")
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
				scanner.DetectFileHits(content, item.Meta.JobWebUrl, item.Meta.JobName, file.Name, "", options.TruffleHogVerification)
			} else if filetype.IsArchive(content) {
				scanner.HandleArchiveArtifact(file.Name, content, item.Meta.JobWebUrl, item.Meta.JobName, options.TruffleHogVerification)
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

	findings, err := scanner.DetectHits(dotenvText, options.MaxScanGoRoutines, options.TruffleHogVerification)
	if err != nil {
		log.Debug().Err(err).Int("project", item.Meta.ProjectId).Int("job", item.Meta.JobId).Msg("Failed detecting secrets")
		return
	}
	for _, finding := range findings {
		artifactsBaseUrl, _ := url.JoinPath(item.Meta.JobWebUrl, "/-/artifacts")
		log.Warn().Str("confidence", finding.Pattern.Pattern.Confidence).Str("ruleName", finding.Pattern.Pattern.Name).Str("value", finding.Text).Str("artifactUrl", artifactsBaseUrl).Int("jobId", item.Meta.JobId).Str("jobName", item.Meta.JobName).Msg("HIT DOTENV: Check artifacts page which is the only place to download the dotenv file")
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

func getJobArtifacts(git *gitlab.Client, projectId int, jobId int, jobWebUrl string, options *ScanOptions) []byte {
	artifactsReader, resp, err := git.Jobs.GetJobArtifacts(projectId, jobId)
	if resp.StatusCode == 404 {
		return nil
	}

	if err != nil {
		log.Error().Err(err).Str("url", jobWebUrl).Msg("Failed downloading job artifacts zip")
		return nil
	}

	if artifactsReader.Size() > options.MaxArtifactSize {
		log.Debug().Int64("bytes", artifactsReader.Size()).Int64("maxBytes", options.MaxArtifactSize).Str("url", jobWebUrl).Msg("Skipped large artifact Zip")
		return nil
	}

	data, err := io.ReadAll(artifactsReader)
	if err != nil {
		log.Error().Err(err).Str("url", jobWebUrl).Msg("Failed reading artifacts stream")
		return nil
	}

	extractedZipSize := helper.CalculateZipFileSize(data)
	if extractedZipSize > uint64(options.MaxArtifactSize) {
		log.Debug().Str("url", jobWebUrl).Int64("zipBytes", artifactsReader.Size()).Uint64("bytesExtracted", extractedZipSize).Int64("maxBytes", options.MaxArtifactSize).Msg("Skipped large extracted Zip artifact")
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

	client := helper.GetPipeleakHTTPClient()
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

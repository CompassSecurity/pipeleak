package scanner

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/url"
	"runtime"

	"github.com/h2non/filetype"
	"github.com/maragudk/goqite"
	"github.com/maragudk/goqite/jobs"
	"github.com/rs/zerolog/log"
	//"github.com/wandb/parallel"
)

type QueueItemType string

const (
	QueueItemJobTrace QueueItemType = "jobTrace"
	QueueItemArtifact QueueItemType = "artifact"
	QueueItemDotenv   QueueItemType = "dotenv"
)

type HitMetaInfo struct {
	ProjectId int
	JobId     int
	JobWebUrl string
}

type QueueItem struct {
	Type        QueueItemType `json:"type"`
	Data        []byte        `json:"data"`
	ScanOptions *ScanOptions  `json:"scanOptions"`
	HitMetaInfo HitMetaInfo   `json:"hitMetaInfo"`
}

func analyzeQueueItem(serializeditem []byte, maxThreads int) {
	var item QueueItem
	err := json.Unmarshal(serializeditem, &item)
	if err != nil {
		log.Error().Err(err).Msg("Failed unmarshalling queue item")
	}

	if item.Type == QueueItemJobTrace {
		log.Debug().Str("url", item.HitMetaInfo.JobWebUrl).Msg("Scanning Job Trace")
		analyzeJobTrace(item, maxThreads)
	}

	if item.Type == QueueItemArtifact {
		log.Debug().Str("url", item.HitMetaInfo.JobWebUrl).Msg("Scanning artifact")
		analyzeJobArtifact(item, maxThreads)
		runtime.GC()
	}

	if item.Type == QueueItemDotenv {
		log.Debug().Str("url", item.HitMetaInfo.JobWebUrl).Msg("Scanning Dotenv")
		analyzeDotenvArtifact(item, maxThreads)
	}

}

func enqueueItem(trace []byte, queue *goqite.Queue, qType QueueItemType, hitMetaInfo HitMetaInfo) {
	item := &QueueItem{Type: qType, Data: trace, HitMetaInfo: hitMetaInfo}
	itemBytes, err := json.Marshal(item)
	if err != nil {
		log.Error().Str("type", string(qType)).Err(err).Msg("Failed marshalling job item")
		return
	}

	err = jobs.Create(context.Background(), queue, "pipeleak-job", itemBytes)
	if err != nil {
		log.Error().Str("type", string(qType)).Err(err).Msg("Failed queuing jpb")
	}
}

func analyzeJobTrace(item QueueItem, maxThreads int) {
	findings := DetectHits(item.Data, maxThreads)
	for _, finding := range findings {
		log.Warn().Str("confidence", finding.Pattern.Pattern.Confidence).Str("name", finding.Pattern.Pattern.Name).Str("value", finding.Text).Str("url", item.HitMetaInfo.JobWebUrl).Msg("HIT")
	}
}

func analyzeJobArtifact(item QueueItem, maxThreads int) {
	reader := bytes.NewReader(item.Data)
	zipListing, err := zip.NewReader(reader, int64(len(item.Data)))
	if err != nil {
		log.Warn().Int("project", item.HitMetaInfo.ProjectId).Int("job", item.HitMetaInfo.JobId).Msg("Unable to unzip artifacts for")
		return
	}

	for _, file := range zipListing.File {
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
			findings := DetectHits(content, maxThreads)
			for _, finding := range findings {
				log.Warn().Str("confidence", finding.Pattern.Pattern.Confidence).Str("name", finding.Pattern.Pattern.Name).Str("value", finding.Text).Str("url", item.HitMetaInfo.JobWebUrl).Str("file", file.Name).Msg("HIT Artifact")
			}
		}
		fc.Close()
	}
}

func analyzeDotenvArtifact(item QueueItem, maxThreads int) {
	findings := DetectHits(item.Data, maxThreads)
	for _, finding := range findings {
		artifactsBaseUrl, _ := url.JoinPath(item.HitMetaInfo.JobWebUrl, "/-/artifacts")
		log.Warn().Str("confidence", finding.Pattern.Pattern.Confidence).Str("name", finding.Pattern.Pattern.Name).Str("value", finding.Text).Str("artifactUrl", artifactsBaseUrl).Int("jobId", item.HitMetaInfo.JobId).Msg("HIT DOTENV: Check artifacts page which is the only place to download the dotenv file")
	}
}

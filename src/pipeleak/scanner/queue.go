package scanner

import (
	"context"
	"encoding/json"

	"github.com/maragudk/goqite"
	"github.com/maragudk/goqite/jobs"
	"github.com/rs/zerolog/log"
)

type QueueItemType string

const (
	QueueItemJobTrace QueueItemType = "jobTrace"
	QueueItemArtifact QueueItemType = "artifact"
)

type QueueItem struct {
	Type        QueueItemType `json:"type"`
	Data        []byte        `json:"data"`
	ScanOptions *ScanOptions  `json:"scanOptions"`
}

func analyzeQueueItem(serializeditem []byte) {
	var item QueueItem
	err := json.Unmarshal(serializeditem, &item)
	if err != nil {
		log.Error().Err(err).Msg("Failed unmarshalling queue item")
	}

	if item.Type == QueueItemJobTrace {
		analyzeJobTrace(item)
	}

}

func enqueueItem(trace []byte, queue *goqite.Queue, qType QueueItemType) {
	item := &QueueItem{Type: QueueItemJobTrace, Data: trace}
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

func analyzeJobTrace(item QueueItem) {
	findings := DetectHits(item.Data)
	for _, finding := range findings {
		log.Warn().Str("confidence", finding.Pattern.Pattern.Confidence).Str("name", finding.Pattern.Pattern.Name).Str("value", finding.Text).Str("url", "getJobUrl(git, project, job)").Msg("HIT")
	}
}

func analyzeJobArtifact() {

}

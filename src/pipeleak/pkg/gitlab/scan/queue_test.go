package scan

import (
	"bytes"
	"strings"
	"testing"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	gitlab "gitlab.com/gitlab-org/api/client-go"
)

// withCapturedLogs temporarily routes zerolog output to a buffer for assertions.
func withCapturedLogs(t *testing.T, level zerolog.Level, fn func(buf *bytes.Buffer)) {
	t.Helper()
	old := log.Logger
	buf := &bytes.Buffer{}
	logger := zerolog.New(buf).Level(level).With().Timestamp().Logger()
	log.Logger = logger
	defer func() { log.Logger = old }()
	fn(buf)
}

func TestAnalyzeJobArtifact_SkipsLargeArtifactPreDownload(t *testing.T) {
	// Arrange: artifact is larger than the configured max — should skip before any download
	item := QueueItem{Meta: QueueMeta{
		ProjectId:    1,
		JobId:        3000,
		JobWebUrl:    "http://gitlab.local/-/jobs/3000",
		JobName:      "large-artifact-job",
		ArtifactSize: 100 * 1024 * 1024, // 100MB
	}}
	opts := &ScanOptions{MaxArtifactSize: 50 * 1024 * 1024, MaxScanGoRoutines: 1}

	withCapturedLogs(t, zerolog.DebugLevel, func(buf *bytes.Buffer) {
		// Act: pass nil gitlab client since we expect early return (no network calls)
		analyzeJobArtifact((*gitlab.Client)(nil), item, opts)

		// Assert: log contains skip message and job name
		logs := buf.String()
		if !strings.Contains(logs, "Skipped large artifact") {
			t.Fatalf("expected skip log, got: %s", logs)
		}
		if !strings.Contains(logs, "large-artifact-job") {
			t.Fatalf("expected job name in logs, got: %s", logs)
		}
	})
}

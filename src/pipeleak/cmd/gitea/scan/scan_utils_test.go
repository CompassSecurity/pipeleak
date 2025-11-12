package scan

import (
	"archive/zip"
	"bytes"
	"context"
	"testing"

	"code.gitea.io/sdk/gitea"
	"github.com/CompassSecurity/pipeleak/pkg/logging"
	"github.com/CompassSecurity/pipeleak/pkg/scanner"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/stretchr/testify/assert"
)

func TestDetermineFileAction(t *testing.T) {
	tests := []struct {
		name             string
		content          []byte
		displayName      string
		expectedAction   string
		expectedFileType string
	}{
		{
			name:             "text file",
			content:          []byte("plain text content"),
			displayName:      "test.txt",
			expectedAction:   "scan",
			expectedFileType: "",
		},
		{
			name:             "empty file",
			content:          []byte{},
			displayName:      "empty.txt",
			expectedAction:   "scan",
			expectedFileType: "",
		},
		{
			name:             "json file",
			content:          []byte(`{"key": "value"}`),
			displayName:      "config.json",
			expectedAction:   "scan",
			expectedFileType: "",
		},
		{
			name: "zip archive",
			content: func() []byte {
				buf := new(bytes.Buffer)
				w := zip.NewWriter(buf)
				f, _ := w.Create("test.txt")
				_, _ = f.Write([]byte("test"))
				_ = w.Close()
				return buf.Bytes()
			}(),
			displayName:      "archive.zip",
			expectedAction:   "archive",
			expectedFileType: "application/zip",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			action, fileType := determineFileAction(tt.content, tt.displayName)

			assert.Equal(t, tt.expectedAction, action)
			if tt.expectedFileType != "" {
				assert.Equal(t, tt.expectedFileType, fileType)
			}
		})
	}
}

func TestLogFinding(t *testing.T) {
	var buf bytes.Buffer
	oldLogger := log.Logger
	hitWriter := logging.NewHitLevelWriter(&buf)
	log.Logger = zerolog.New(hitWriter).With().Timestamp().Logger()
	logging.SetGlobalHitWriter(hitWriter)
	defer func() { log.Logger = oldLogger }()

	tests := []struct {
		name         string
		finding      scanner.Finding
		repoFullName string
		runID        int64
		jobID        int64
		jobName      string
		url          string
		expectInLog  []string
	}{
		{
			name: "complete finding info",
			finding: scanner.Finding{
				Pattern: scanner.PatternElement{
					Pattern: scanner.PatternPattern{
						Name:       "Test Secret",
						Confidence: "high",
					},
				},
				Text: "secret_value_123",
			},
			repoFullName: "owner/repo",
			runID:        123,
			jobID:        456,
			jobName:      "test-job",
			url:          "https://gitea.example.com/owner/repo/actions/runs/123",
			expectInLog:  []string{"Test Secret", "high", "owner/repo", "test-job", "secret_value_123"},
		},
		{
			name: "finding without job info",
			finding: scanner.Finding{
				Pattern: scanner.PatternElement{
					Pattern: scanner.PatternPattern{
						Name:       "AWS Key",
						Confidence: "medium",
					},
				},
				Text: "AKIAIOSFODNN7EXAMPLE",
			},
			repoFullName: "owner/repo",
			runID:        123,
			jobID:        0,
			jobName:      "",
			url:          "https://gitea.example.com/owner/repo/actions/runs/123",
			expectInLog:  []string{"AWS Key", "medium", "owner/repo", "AKIAIOSFODNN7EXAMPLE"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			buf.Reset()

			assert.NotPanics(t, func() {
				logFinding(tt.finding, tt.repoFullName, tt.runID, tt.jobID, tt.jobName, tt.url)
			})

			output := buf.String()
			for _, expected := range tt.expectInLog {
				assert.Contains(t, output, expected, "Expected log to contain %q", expected)
			}
		})
	}
}

func TestProcessZipArtifact(t *testing.T) {
	tests := []struct {
		name         string
		zipContent   []byte
		artifactName string
		expectError  bool
	}{
		{
			name: "valid zip with text file",
			zipContent: func() []byte {
				buf := new(bytes.Buffer)
				w := zip.NewWriter(buf)
				f, _ := w.Create("test.txt")
				_, _ = f.Write([]byte("test content"))
				_ = w.Close()
				return buf.Bytes()
			}(),
			artifactName: "test-artifact",
			expectError:  false,
		},
		{
			name: "zip with multiple files",
			zipContent: func() []byte {
				buf := new(bytes.Buffer)
				w := zip.NewWriter(buf)
				f1, _ := w.Create("file1.txt")
				_, _ = f1.Write([]byte("content 1"))
				f2, _ := w.Create("file2.txt")
				_, _ = f2.Write([]byte("content 2"))
				_ = w.Close()
				return buf.Bytes()
			}(),
			artifactName: "multi-file-artifact",
			expectError:  false,
		},
		{
			name:         "not a zip file - should scan directly",
			zipContent:   []byte("plain text content"),
			artifactName: "plain-text-artifact",
			expectError:  false,
		},
		{
			name:         "empty content",
			zipContent:   []byte{},
			artifactName: "empty-artifact",
			expectError:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			setupTestScanOptions()
			repo := &gitea.Repository{
				FullName: "owner/repo",
			}
			run := ActionWorkflowRun{
				ID:      123,
				HTMLURL: "https://gitea.example.com/owner/repo/actions/runs/123",
				Name:    "Test Run",
			}

			assert.NotPanics(t, func() {
				processZipArtifact(tt.zipContent, repo, run, tt.artifactName)
			})
		})
	}
}

func TestProcessZipArtifact_NilRepo(t *testing.T) {
	zipContent := []byte("test")
	run := ActionWorkflowRun{ID: 123}

	assert.NotPanics(t, func() {
		processZipArtifact(zipContent, nil, run, "test-artifact")
	})
}

func TestScanLogs_NilRepo(t *testing.T) {
	logBytes := []byte("test log content")
	run := ActionWorkflowRun{ID: 123}

	assert.NotPanics(t, func() {
		scanLogs(logBytes, nil, run, 456, "test-job")
	})
}

func TestScanLogs_EmptyLogs(t *testing.T) {
	setupTestScanOptions()
	logBytes := []byte("")
	repo := &gitea.Repository{
		FullName: "owner/repo",
	}
	run := ActionWorkflowRun{
		ID:      123,
		HTMLURL: "https://gitea.example.com/owner/repo/actions/runs/123",
	}

	assert.NotPanics(t, func() {
		scanLogs(logBytes, repo, run, 456, "test-job")
	})
}

func TestScanArtifactContent(t *testing.T) {
	tests := []struct {
		name         string
		content      []byte
		artifactName string
		fileName     string
		repo         *gitea.Repository
		run          ActionWorkflowRun
	}{
		{
			name:         "scan artifact without filename",
			content:      []byte("test content"),
			artifactName: "test-artifact",
			fileName:     "",
			repo: &gitea.Repository{
				FullName: "owner/repo",
			},
			run: ActionWorkflowRun{
				ID:      123,
				HTMLURL: "https://gitea.example.com/owner/repo/actions/runs/123",
				Name:    "Test Run",
			},
		},
		{
			name:         "scan artifact with filename",
			content:      []byte("test content"),
			artifactName: "test-artifact",
			fileName:     "inner-file.txt",
			repo: &gitea.Repository{
				FullName: "owner/repo",
			},
			run: ActionWorkflowRun{
				ID:      123,
				HTMLURL: "https://gitea.example.com/owner/repo/actions/runs/123",
				Name:    "Test Run",
			},
		},
		{
			name: "scan zip archive",
			content: func() []byte {
				buf := new(bytes.Buffer)
				w := zip.NewWriter(buf)
				f, _ := w.Create("nested.txt")
				_, _ = f.Write([]byte("nested content"))
				_ = w.Close()
				return buf.Bytes()
			}(),
			artifactName: "nested-artifact",
			fileName:     "",
			repo: &gitea.Repository{
				FullName: "owner/repo",
			},
			run: ActionWorkflowRun{
				ID:      123,
				HTMLURL: "https://gitea.example.com/owner/repo/actions/runs/123",
				Name:    "Test Run",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			setupTestScanOptions()

			// Execute - should not panic
			assert.NotPanics(t, func() {
				scanArtifactContent(tt.content, tt.repo, tt.run, tt.artifactName, tt.fileName)
			})
		})
	}
}

func TestProcessZipArtifact_ConcurrentFileProcessing(t *testing.T) {
	buf := new(bytes.Buffer)
	w := zip.NewWriter(buf)

	for i := 1; i <= 10; i++ {
		f, _ := w.Create(string(rune('a'+i-1)) + ".txt")
		_, _ = f.Write([]byte("content " + string(rune('0'+i))))
	}
	_ = w.Close()

	setupTestScanOptions()
	scanOptions.MaxScanGoRoutines = 4
	scanOptions.Context = context.Background()

	repo := &gitea.Repository{
		FullName: "owner/repo",
	}
	run := ActionWorkflowRun{
		ID:      123,
		HTMLURL: "https://gitea.example.com/owner/repo/actions/runs/123",
		Name:    "Test Run",
	}

	assert.NotPanics(t, func() {
		processZipArtifact(buf.Bytes(), repo, run, "concurrent-test-artifact")
	})
}

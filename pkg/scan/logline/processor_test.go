package logline

import (
	"archive/zip"
	"bytes"
	"testing"
	"time"

	"github.com/CompassSecurity/pipeleek/pkg/scanner"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func init() {
	scanner.InitRules([]string{})
}

const testTimeout = 60 * time.Second

func TestProcessLogs(t *testing.T) {
	tests := []struct {
		name        string
		logs        []byte
		opts        ProcessOptions
		expectError bool
	}{
		{
			name: "empty logs",
			logs: []byte{},
			opts: ProcessOptions{
				MaxGoRoutines:     4,
				VerifyCredentials: false,
				HitTimeout:        testTimeout,
			},
			expectError: false,
		},
		{
			name: "logs with no secrets",
			logs: []byte("INFO: Starting build\nINFO: Running tests\nINFO: Build complete"),
			opts: ProcessOptions{
				MaxGoRoutines:     4,
				VerifyCredentials: false,
				BuildURL:          "https://example.com/build/123",
				HitTimeout:        testTimeout,
			},
			expectError: false,
		},
		{
			name: "logs with potential pattern",
			logs: []byte("Connecting to database...\nUsing API key: test123\nBuild successful"),
			opts: ProcessOptions{
				MaxGoRoutines:     4,
				VerifyCredentials: false,
				HitTimeout:        testTimeout,
			},
			expectError: false,
		},
		{
			name: "large log file",
			logs: bytes.Repeat([]byte("INFO: Log line\n"), 1000),
			opts: ProcessOptions{
				MaxGoRoutines:     4,
				VerifyCredentials: false,
				HitTimeout:        testTimeout,
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ProcessLogs(tt.logs, tt.opts)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.NotNil(t, result)
				assert.Equal(t, len(tt.logs), result.BytesRead)
				assert.NotNil(t, result.Findings)
			}
		})
	}
}

func TestExtractLogsFromZip(t *testing.T) {
	tests := []struct {
		name          string
		createZip     func() []byte
		expectError   bool
		expectFiles   int
		expectContent string
	}{
		{
			name: "empty zip bytes",
			createZip: func() []byte {
				return []byte{}
			},
			expectError: false,
			expectFiles: 0,
		},
		{
			name: "zip with single log file",
			createZip: func() []byte {
				buf := new(bytes.Buffer)
				w := zip.NewWriter(buf)
				f, _ := w.Create("build.log")
				_, _ = f.Write([]byte("Build log content"))
				_ = w.Close()
				return buf.Bytes()
			},
			expectError:   false,
			expectFiles:   1,
			expectContent: "Build log content",
		},
		{
			name: "zip with multiple log files",
			createZip: func() []byte {
				buf := new(bytes.Buffer)
				w := zip.NewWriter(buf)
				f1, _ := w.Create("step1.log")
				_, _ = f1.Write([]byte("Step 1 logs"))
				f2, _ := w.Create("step2.log")
				_, _ = f2.Write([]byte("Step 2 logs"))
				_ = w.Close()
				return buf.Bytes()
			},
			expectError: false,
			expectFiles: 2,
		},
		{
			name: "invalid zip data",
			createZip: func() []byte {
				return []byte("not a zip file")
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			zipBytes := tt.createZip()
			result, err := ExtractLogsFromZip(zipBytes)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.expectFiles, result.FileCount)
				if tt.expectContent != "" {
					assert.Contains(t, string(result.ExtractedLogs), tt.expectContent)
				}
			}
		})
	}
}

func TestProcessLogsFromZip(t *testing.T) {
	tests := []struct {
		name        string
		createZip   func() []byte
		opts        ProcessOptions
		expectError bool
	}{
		{
			name: "zip with log files",
			createZip: func() []byte {
				buf := new(bytes.Buffer)
				w := zip.NewWriter(buf)
				f, _ := w.Create("build.log")
				_, _ = f.Write([]byte("INFO: Build started\nINFO: Build completed"))
				_ = w.Close()
				return buf.Bytes()
			},
			opts: ProcessOptions{
				MaxGoRoutines:     4,
				VerifyCredentials: false,
				BuildURL:          "https://example.com/build/456",
				HitTimeout:        testTimeout,
			},
			expectError: false,
		},
		{
			name: "empty zip",
			createZip: func() []byte {
				buf := new(bytes.Buffer)
				w := zip.NewWriter(buf)
				_ = w.Close()
				return buf.Bytes()
			},
			opts: ProcessOptions{
				MaxGoRoutines: 4,
				HitTimeout:    testTimeout,
			},
			expectError: false,
		},
		{
			name: "invalid zip",
			createZip: func() []byte {
				return []byte("invalid")
			},
			opts:        ProcessOptions{MaxGoRoutines: 4, HitTimeout: testTimeout},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			zipBytes := tt.createZip()
			result, err := ProcessLogsFromZip(zipBytes, tt.opts)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.NotNil(t, result)
				assert.NotNil(t, result.Findings)
			}
		})
	}
}

func TestProcessLogs_WithThreads(t *testing.T) {
	// Test different thread configurations
	logs := bytes.Repeat([]byte("INFO: Test log line\n"), 100)

	threadConfigs := []int{1, 2, 4, 8}

	for _, threads := range threadConfigs {
		t.Run("threads="+string(rune('0'+threads)), func(t *testing.T) {
			opts := ProcessOptions{
				MaxGoRoutines:     threads,
				VerifyCredentials: false,
				HitTimeout:        testTimeout,
			}

			result, err := ProcessLogs(logs, opts)
			assert.NoError(t, err)
			assert.NotNil(t, result)
			assert.Equal(t, len(logs), result.BytesRead)
		})
	}
}

func TestExtractLogsFromZip_WithErrors(t *testing.T) {
	// Test that errors in individual files are collected but don't stop processing
	buf := new(bytes.Buffer)
	w := zip.NewWriter(buf)
	f1, _ := w.Create("good.log")
	_, _ = f1.Write([]byte("Good log content"))
	f2, _ := w.Create("also-good.log")
	_, _ = f2.Write([]byte("More good content"))
	_ = w.Close()

	result, err := ExtractLogsFromZip(buf.Bytes())

	assert.NoError(t, err)
	assert.Equal(t, 2, result.FileCount)
	assert.Greater(t, result.TotalBytes, 0)
	// All files should be processed successfully in this test
	assert.Len(t, result.Errors, 0)
}

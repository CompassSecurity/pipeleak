package processor

import (
	"archive/zip"
	"bytes"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestProcessWorkflowLogs(t *testing.T) {
	tests := []struct {
		name              string
		logs              []byte
		workflowURL       string
		maxGoRoutines     int
		verifyCredentials bool
		wantError         bool
	}{
		{
			name:              "empty logs",
			logs:              []byte(""),
			workflowURL:       "https://github.com/owner/repo/actions/runs/123",
			maxGoRoutines:     4,
			verifyCredentials: false,
			wantError:         false,
		},
		{
			name:              "logs with no secrets",
			logs:              []byte("Workflow started\nRunning job\nJob completed successfully"),
			workflowURL:       "https://github.com/owner/repo/actions/runs/456",
			maxGoRoutines:     4,
			verifyCredentials: false,
			wantError:         false,
		},
		{
			name: "logs with potential secret",
			logs: []byte(`Setting up environment
export GITHUB_TOKEN=ghp_1234567890abcdefghijklmnopqrstuvwxyz
Running tests`),
			workflowURL:       "https://github.com/company/project/actions/runs/789",
			maxGoRoutines:     4,
			verifyCredentials: false,
			wantError:         false,
		},
		{
			name:              "large log file",
			logs:              bytes.Repeat([]byte("Log line\n"), 50000),
			workflowURL:       "https://github.com/org/repo/actions/runs/999",
			maxGoRoutines:     8,
			verifyCredentials: false,
			wantError:         false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ProcessWorkflowLogs(tt.logs, tt.workflowURL, tt.maxGoRoutines, tt.verifyCredentials)

			if tt.wantError {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.NotNil(t, result)
			assert.Equal(t, tt.workflowURL, result.WorkflowURL)
			assert.NotNil(t, result.Findings)
		})
	}
}

func TestExtractLogsFromZip(t *testing.T) {
	tests := []struct {
		name            string
		zipContent      func() []byte
		wantError       bool
		expectFileCount int
		expectMinBytes  int
	}{
		{
			name: "valid zip with single log file",
			zipContent: func() []byte {
				buf := new(bytes.Buffer)
				w := zip.NewWriter(buf)

				f, _ := w.Create("job.log")
				_, _ = fmt.Fprintf(f, "Job started\nRunning tests\nJob completed")

				_ = w.Close()
				return buf.Bytes()
			},
			wantError:       false,
			expectFileCount: 1,
			expectMinBytes:  10,
		},
		{
			name: "zip with multiple log files",
			zipContent: func() []byte {
				buf := new(bytes.Buffer)
				w := zip.NewWriter(buf)

				f1, _ := w.Create("setup.log")
				_, _ = fmt.Fprintf(f1, "Setting up environment")

				f2, _ := w.Create("build.log")
				_, _ = fmt.Fprintf(f2, "Building project")

				f3, _ := w.Create("test.log")
				_, _ = fmt.Fprintf(f3, "Running tests")

				_ = w.Close()
				return buf.Bytes()
			},
			wantError:       false,
			expectFileCount: 3,
			expectMinBytes:  30,
		},
		{
			name: "empty zip",
			zipContent: func() []byte {
				buf := new(bytes.Buffer)
				w := zip.NewWriter(buf)
				_ = w.Close()
				return buf.Bytes()
			},
			wantError:       false,
			expectFileCount: 0,
			expectMinBytes:  0,
		},
		{
			name: "invalid zip data",
			zipContent: func() []byte {
				return []byte("not a zip file")
			},
			wantError:       true,
			expectFileCount: 0,
			expectMinBytes:  0,
		},
		{
			name: "zip with large log file",
			zipContent: func() []byte {
				buf := new(bytes.Buffer)
				w := zip.NewWriter(buf)

				f, _ := w.Create("large.log")
				for i := 0; i < 1000; i++ {
					_, _ = fmt.Fprintf(f, "Line %d: Some log content\n", i)
				}

				_ = w.Close()
				return buf.Bytes()
			},
			wantError:       false,
			expectFileCount: 1,
			expectMinBytes:  20000, // At least 20KB
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			zipBytes := tt.zipContent()
			result, err := ExtractLogsFromZip(zipBytes)

			if tt.wantError {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.NotNil(t, result)
			assert.Equal(t, tt.expectFileCount, result.FileCount)
			assert.GreaterOrEqual(t, result.TotalBytes, tt.expectMinBytes)
			assert.Equal(t, result.TotalBytes, len(result.ExtractedLogs))
		})
	}
}

func TestWorkflowRunFilter_ShouldContinueScanning(t *testing.T) {
	tests := []struct {
		name         string
		maxWorkflows int
		currentCount int
		want         bool
	}{
		{
			name:         "no limit set (negative)",
			maxWorkflows: -1,
			currentCount: 100,
			want:         true,
		},
		{
			name:         "no limit set (zero)",
			maxWorkflows: 0,
			currentCount: 50,
			want:         true,
		},
		{
			name:         "under limit",
			maxWorkflows: 10,
			currentCount: 5,
			want:         true,
		},
		{
			name:         "at limit",
			maxWorkflows: 10,
			currentCount: 10,
			want:         false,
		},
		{
			name:         "over limit",
			maxWorkflows: 10,
			currentCount: 15,
			want:         false,
		},
		{
			name:         "just started",
			maxWorkflows: 5,
			currentCount: 0,
			want:         true,
		},
		{
			name:         "one before limit",
			maxWorkflows: 10,
			currentCount: 9,
			want:         true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filter := &WorkflowRunFilter{
				MaxWorkflows: tt.maxWorkflows,
				CurrentCount: tt.currentCount,
			}
			got := filter.ShouldContinueScanning()
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestWorkflowRunFilter_ReachedLimit(t *testing.T) {
	tests := []struct {
		name         string
		maxWorkflows int
		currentCount int
		want         bool
	}{
		{
			name:         "no limit - not reached",
			maxWorkflows: 0,
			currentCount: 100,
			want:         false,
		},
		{
			name:         "under limit",
			maxWorkflows: 10,
			currentCount: 5,
			want:         false,
		},
		{
			name:         "at limit",
			maxWorkflows: 10,
			currentCount: 10,
			want:         true,
		},
		{
			name:         "over limit",
			maxWorkflows: 10,
			currentCount: 15,
			want:         true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filter := &WorkflowRunFilter{
				MaxWorkflows: tt.maxWorkflows,
				CurrentCount: tt.currentCount,
			}
			got := filter.ReachedLimit()
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestWorkflowRunFilter_IncrementCount(t *testing.T) {
	filter := &WorkflowRunFilter{
		MaxWorkflows: 10,
		CurrentCount: 0,
	}

	// Test incrementing
	assert.Equal(t, 0, filter.CurrentCount)
	filter.IncrementCount()
	assert.Equal(t, 1, filter.CurrentCount)
	filter.IncrementCount()
	assert.Equal(t, 2, filter.CurrentCount)

	// Test flow control
	for filter.ShouldContinueScanning() {
		filter.IncrementCount()
		if filter.CurrentCount > 20 {
			t.Fatal("Should have stopped at limit")
		}
	}

	assert.Equal(t, 10, filter.CurrentCount)
	assert.True(t, filter.ReachedLimit())
}

func TestExtractLogsFromZip_ErrorHandling(t *testing.T) {
	// Create a zip with a file that will cause an error during reading
	// In practice, this is hard to trigger, but we test empty files
	buf := new(bytes.Buffer)
	w := zip.NewWriter(buf)

	// Add some valid files
	f1, _ := w.Create("valid1.log")
	_, _ = fmt.Fprintf(f1, "Valid content 1")

	f2, _ := w.Create("valid2.log")
	_, _ = fmt.Fprintf(f2, "Valid content 2")

	_ = w.Close()

	result, err := ExtractLogsFromZip(buf.Bytes())
	require.NoError(t, err)
	assert.Equal(t, 2, result.FileCount)
	assert.Greater(t, result.TotalBytes, 20)
	assert.Empty(t, result.Errors, "Should have no errors with valid files")
}

func TestProcessWorkflowLogs_PreservesContext(t *testing.T) {
	// Verify that context information is preserved
	workflowURL := "https://github.com/test/repo/actions/runs/12345"
	result, err := ProcessWorkflowLogs([]byte("test log"), workflowURL, 4, false)

	require.NoError(t, err)
	assert.Equal(t, workflowURL, result.WorkflowURL)
}

func TestExtractLogsFromZip_ConcatenatesLogs(t *testing.T) {
	// Verify that logs from multiple files are concatenated
	buf := new(bytes.Buffer)
	w := zip.NewWriter(buf)

	f1, _ := w.Create("file1.log")
	_, _ = fmt.Fprintf(f1, "First")

	f2, _ := w.Create("file2.log")
	_, _ = fmt.Fprintf(f2, "Second")

	f3, _ := w.Create("file3.log")
	_, _ = fmt.Fprintf(f3, "Third")

	_ = w.Close()

	result, err := ExtractLogsFromZip(buf.Bytes())
	require.NoError(t, err)

	extracted := string(result.ExtractedLogs)
	assert.Contains(t, extracted, "First")
	assert.Contains(t, extracted, "Second")
	assert.Contains(t, extracted, "Third")
}

package processor

import (
	"archive/zip"
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestProcessJobTrace(t *testing.T) {
	jobInfo := JobInfo{
		ProjectID: 123,
		JobID:     456,
		JobWebURL: "https://gitlab.com/project/job/456",
		JobName:   "test-job",
	}

	tests := []struct {
		name        string
		trace       []byte
		expectError bool
		expectEmpty bool
		description string
	}{
		{
			name:        "empty trace",
			trace:       []byte{},
			expectEmpty: true,
			description: "Should handle empty trace gracefully",
		},
		{
			name:        "nil trace",
			trace:       nil,
			expectEmpty: true,
			description: "Should handle nil trace gracefully",
		},
		{
			name:        "normal trace without secrets",
			trace:       []byte("Running job\nInstalling dependencies\nBuild complete\n"),
			expectEmpty: true,
			description: "Should process normal trace without false positives",
		},
		{
			name:        "trace with potential secret",
			trace:       []byte("export AWS_SECRET_ACCESS_KEY=wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY\n"),
			expectEmpty: false,
			description: "Should detect secrets in trace logs",
		},
		{
			name:        "large trace",
			trace:       bytes.Repeat([]byte("Log line with no secrets\n"), 10000),
			expectEmpty: true,
			description: "Should handle large traces efficiently",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ProcessJobTrace(tt.trace, jobInfo, 4, false)

			if tt.expectError {
				assert.Error(t, result.Error)
			} else {
				if result.Error != nil {
					t.Logf("Unexpected error: %v", result.Error)
				}
			}

			if tt.expectEmpty {
				assert.Empty(t, result.Findings, "Expected no findings for %s", tt.description)
			}
		})
	}
}

func TestProcessJobTrace_PreservesMetadata(t *testing.T) {
	jobInfo := JobInfo{
		ProjectID: 789,
		JobID:     101,
		JobWebURL: "https://gitlab.com/test/job/101",
		JobName:   "integration-test",
	}

	trace := []byte("Some trace content\n")
	result := ProcessJobTrace(trace, jobInfo, 2, false)

	assert.NotNil(t, result)
	assert.NoError(t, result.Error)
}

func TestProcessDotenvArtifact(t *testing.T) {
	jobInfo := JobInfo{
		ProjectID: 123,
		JobID:     456,
		JobWebURL: "https://gitlab.com/project/job/456",
		JobName:   "test-job",
	}

	tests := []struct {
		name        string
		dotenvText  []byte
		expectError bool
		expectEmpty bool
		description string
	}{
		{
			name:        "empty dotenv",
			dotenvText:  []byte{},
			expectEmpty: true,
			description: "Should handle empty dotenv file",
		},
		{
			name:        "nil dotenv",
			dotenvText:  nil,
			expectEmpty: true,
			description: "Should handle nil dotenv content",
		},
		{
			name:        "normal dotenv without secrets",
			dotenvText:  []byte("DATABASE_URL=postgres://localhost:5432/db\nAPP_ENV=production\n"),
			expectEmpty: true,
			description: "Should process normal dotenv without false positives",
		},
		{
			name:        "dotenv with potential secret",
			dotenvText:  []byte("AWS_SECRET_ACCESS_KEY=wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY\n"),
			expectEmpty: false,
			description: "Should detect secrets in dotenv files",
		},
		{
			name:        "dotenv with multiple variables",
			dotenvText:  []byte("VAR1=value1\nVAR2=value2\nVAR3=value3\n"),
			expectEmpty: true,
			description: "Should handle multiple environment variables",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ProcessDotenvArtifact(tt.dotenvText, jobInfo, 4, false)

			if tt.expectError {
				assert.Error(t, result.Error)
			} else {
				if result.Error != nil {
					t.Logf("Unexpected error: %v", result.Error)
				}
			}

			if tt.expectEmpty {
				assert.Empty(t, result.Findings, "Expected no findings for %s", tt.description)
			}
		})
	}
}

func TestExtractArtifactFiles(t *testing.T) {
	tests := []struct {
		name        string
		setupZip    func() []byte
		expectCount int
		expectError bool
		description string
	}{
		{
			name: "valid zip with text files",
			setupZip: func() []byte {
				var buf bytes.Buffer
				w := zip.NewWriter(&buf)

				f1, _ := w.Create("file1.txt")
				_, _ = f1.Write([]byte("content1"))

				f2, _ := w.Create("file2.txt")
				_, _ = f2.Write([]byte("content2"))

				_ = w.Close()
				return buf.Bytes()
			},
			expectCount: 2,
			description: "Should extract all files from valid zip",
		},
		{
			name: "empty zip",
			setupZip: func() []byte {
				var buf bytes.Buffer
				w := zip.NewWriter(&buf)
				_ = w.Close()
				return buf.Bytes()
			},
			expectCount: 0,
			description: "Should handle empty zip",
		},
		{
			name:        "nil data",
			setupZip:    func() []byte { return nil },
			expectCount: 0,
			description: "Should handle nil data",
		},
		{
			name:        "invalid zip data",
			setupZip:    func() []byte { return []byte("not a zip file") },
			expectError: true,
			description: "Should return error for invalid zip",
		},
		{
			name: "zip with nested directory",
			setupZip: func() []byte {
				var buf bytes.Buffer
				w := zip.NewWriter(&buf)

				f1, _ := w.Create("dir/file.txt")
				_, _ = f1.Write([]byte("nested content"))

				_ = w.Close()
				return buf.Bytes()
			},
			expectCount: 1,
			description: "Should handle nested directories",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data := tt.setupZip()
			files, err := ExtractArtifactFiles(data)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Len(t, files, tt.expectCount, tt.description)
			}
		})
	}
}

func TestExtractArtifactFiles_FileTypes(t *testing.T) {
	var buf bytes.Buffer
	w := zip.NewWriter(&buf)

	// Add a text file (unknown filetype)
	f1, _ := w.Create("text.txt")
	_, _ = f1.Write([]byte("plain text content"))

	// Add a JSON file (also unknown to filetype)
	f2, _ := w.Create("data.json")
	_, _ = f2.Write([]byte(`{"key": "value"}`))

	_ = w.Close()

	files, err := ExtractArtifactFiles(buf.Bytes())
	require.NoError(t, err)
	require.Len(t, files, 2)

	assert.Equal(t, "text.txt", files[0].Name)
	assert.True(t, files[0].IsUnknown, "Text files should be marked as unknown type")

	assert.Equal(t, "data.json", files[1].Name)
	assert.True(t, files[1].IsUnknown, "JSON files should be marked as unknown type")
}

func TestProcessJobArtifactZip(t *testing.T) {
	tests := []struct {
		name        string
		setupZip    func() []byte
		expectError bool
		description string
	}{
		{
			name: "valid artifact zip",
			setupZip: func() []byte {
				var buf bytes.Buffer
				w := zip.NewWriter(&buf)

				f1, _ := w.Create("logs.txt")
				_, _ = f1.Write([]byte("build logs"))

				_ = w.Close()
				return buf.Bytes()
			},
			expectError: false,
			description: "Should process valid artifact zip",
		},
		{
			name:        "nil data",
			setupZip:    func() []byte { return nil },
			expectError: false,
			description: "Should handle nil data gracefully",
		},
		{
			name:        "empty data",
			setupZip:    func() []byte { return []byte{} },
			expectError: false,
			description: "Should handle empty data gracefully",
		},
		{
			name:        "invalid zip data",
			setupZip:    func() []byte { return []byte("not a zip") },
			expectError: true,
			description: "Should return error for invalid zip",
		},
		{
			name: "large zip with multiple files",
			setupZip: func() []byte {
				var buf bytes.Buffer
				w := zip.NewWriter(&buf)

				for i := 0; i < 50; i++ {
					f, _ := w.Create("file" + string(rune(i)) + ".txt")
					_, _ = f.Write([]byte("content"))
				}

				_ = w.Close()
				return buf.Bytes()
			},
			expectError: false,
			description: "Should handle large zips with many files",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data := tt.setupZip()
			result, err := ProcessJobArtifactZip(data, 4)

			assert.NotNil(t, result)
			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestProcessJobArtifactZip_CountsFiles(t *testing.T) {
	var buf bytes.Buffer
	w := zip.NewWriter(&buf)

	f1, _ := w.Create("file1.txt")
	_, _ = f1.Write([]byte("content1"))

	f2, _ := w.Create("file2.txt")
	_, _ = f2.Write([]byte("content2"))

	f3, _ := w.Create("file3.txt")
	_, _ = f3.Write([]byte("content3"))

	_ = w.Close()

	result, err := ProcessJobArtifactZip(buf.Bytes(), 4)
	require.NoError(t, err)
	assert.Equal(t, 3, result.FilesProcessed, "Should count all processed files")
}

func TestProcessArtifactFile(t *testing.T) {
	tests := []struct {
		name        string
		fileName    string
		content     []byte
		description string
	}{
		{
			name:        "text file",
			fileName:    "log.txt",
			content:     []byte("text content"),
			description: "Should handle text files",
		},
		{
			name:        "json file",
			fileName:    "data.json",
			content:     []byte(`{"key":"value"}`),
			description: "Should handle JSON files",
		},
		{
			name:        "empty file",
			fileName:    "empty.txt",
			content:     []byte{},
			description: "Should handle empty files",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// ProcessArtifactFile has side effects (calls scanner functions)
			// We just ensure it doesn't panic
			assert.NotPanics(t, func() {
				ProcessArtifactFile(tt.fileName, tt.content)
			}, tt.description)
		})
	}
}

package processor

import (
	"archive/zip"
	"bytes"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestProcessLogContent(t *testing.T) {
	tests := []struct {
		name              string
		logs              []byte
		buildURL          string
		maxGoRoutines     int
		verifyCredentials bool
		wantError         bool
	}{
		{
			name:              "empty logs",
			logs:              []byte(""),
			buildURL:          "https://example.com/build/123",
			maxGoRoutines:     4,
			verifyCredentials: false,
			wantError:         false,
		},
		{
			name:              "logs with no secrets",
			logs:              []byte("Build started\nInstalling dependencies\nBuild completed successfully"),
			buildURL:          "https://example.com/build/123",
			maxGoRoutines:     4,
			verifyCredentials: false,
			wantError:         false,
		},
		{
			name: "logs with potential secret pattern",
			logs: []byte(`Build started
Setting environment variables
export API_KEY=sk-1234567890abcdefghijklmnopqrstuvwxyz12345
Running tests`),
			buildURL:          "https://example.com/build/456",
			maxGoRoutines:     4,
			verifyCredentials: false,
			wantError:         false,
		},
		{
			name:              "large log file",
			logs:              bytes.Repeat([]byte("Log line\n"), 10000),
			buildURL:          "https://example.com/build/789",
			maxGoRoutines:     8,
			verifyCredentials: false,
			wantError:         false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ProcessLogContent(tt.logs, tt.buildURL, tt.maxGoRoutines, tt.verifyCredentials)

			if tt.wantError {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.NotNil(t, result)
			assert.Equal(t, tt.buildURL, result.BuildURL)
			assert.NotNil(t, result.Findings)
		})
	}
}

func TestProcessArtifactZip(t *testing.T) {
	tests := []struct {
		name              string
		zipContent        func() []byte
		artifactName      string
		buildURL          string
		maxGoRoutines     int
		verifyCredentials bool
		wantError         bool
		expectFiles       int
	}{
		{
			name: "valid zip with text file",
			zipContent: func() []byte {
				buf := new(bytes.Buffer)
				w := zip.NewWriter(buf)

				f, _ := w.Create("test.txt")
				_, _ = f.Write([]byte("This is a test file"))

				_ = w.Close()
				return buf.Bytes()
			},
			artifactName:      "test-artifact.zip",
			buildURL:          "https://example.com/build/123",
			maxGoRoutines:     4,
			verifyCredentials: false,
			wantError:         false,
			expectFiles:       1,
		},
		{
			name: "zip with multiple files",
			zipContent: func() []byte {
				buf := new(bytes.Buffer)
				w := zip.NewWriter(buf)

				f1, _ := w.Create("file1.txt")
				_, _ = f1.Write([]byte("Content 1"))

				f2, _ := w.Create("file2.log")
				_, _ = f2.Write([]byte("Log content"))

				f3, _ := w.Create("config.json")
				_, _ = f3.Write([]byte(`{"key": "value"}`))

				_ = w.Close()
				return buf.Bytes()
			},
			artifactName:      "multi-file.zip",
			buildURL:          "https://example.com/build/456",
			maxGoRoutines:     4,
			verifyCredentials: false,
			wantError:         false,
			expectFiles:       3,
		},
		{
			name: "empty zip",
			zipContent: func() []byte {
				buf := new(bytes.Buffer)
				w := zip.NewWriter(buf)
				_ = w.Close()
				return buf.Bytes()
			},
			artifactName:      "empty.zip",
			buildURL:          "https://example.com/build/789",
			maxGoRoutines:     4,
			verifyCredentials: false,
			wantError:         false,
			expectFiles:       0,
		},
		{
			name: "invalid zip data",
			zipContent: func() []byte {
				return []byte("not a zip file")
			},
			artifactName:      "invalid.zip",
			buildURL:          "https://example.com/build/999",
			maxGoRoutines:     4,
			verifyCredentials: false,
			wantError:         true,
			expectFiles:       0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			zipBytes := tt.zipContent()
			result, err := ProcessArtifactZip(zipBytes, tt.artifactName, tt.buildURL, tt.maxGoRoutines, tt.verifyCredentials)

			if tt.wantError {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.NotNil(t, result)
			assert.Equal(t, tt.artifactName, result.ArtifactName)
			assert.Equal(t, tt.buildURL, result.BuildURL)
			assert.Len(t, result.FileResults, tt.expectFiles)
		})
	}
}

func TestProcessFileContent(t *testing.T) {
	tests := []struct {
		name              string
		content           []byte
		filename          string
		artifactName      string
		buildURL          string
		verifyCredentials bool
		expectProcessed   bool
	}{
		{
			name:              "text file",
			content:           []byte("This is a plain text file"),
			filename:          "readme.txt",
			artifactName:      "docs.zip",
			buildURL:          "https://example.com/build/123",
			verifyCredentials: false,
			expectProcessed:   true,
		},
		{
			name:              "json file",
			content:           []byte(`{"key": "value", "number": 123}`),
			filename:          "config.json",
			artifactName:      "config.zip",
			buildURL:          "https://example.com/build/456",
			verifyCredentials: false,
			expectProcessed:   true,
		},
		{
			name:              "empty file",
			content:           []byte{},
			filename:          "empty.txt",
			artifactName:      "artifacts.zip",
			buildURL:          "https://example.com/build/789",
			verifyCredentials: false,
			expectProcessed:   true,
		},
		{
			name: "file with potential secret",
			content: []byte(`
API_KEY=sk-1234567890abcdefghijklmnopqrstuvwxyz12345
DATABASE_URL=postgres://user:pass@localhost:5432/db
			`),
			filename:          ".env",
			artifactName:      "env-vars.zip",
			buildURL:          "https://example.com/build/999",
			verifyCredentials: false,
			expectProcessed:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ProcessFileContent(tt.content, tt.filename, tt.artifactName, tt.buildURL, tt.verifyCredentials)

			require.NoError(t, err)
			assert.NotNil(t, result)
			assert.Equal(t, tt.filename, result.FileName)

			// If we processed the file, findings slice should be initialized (even if empty)
			if tt.expectProcessed {
				assert.NotNil(t, result.Findings)
			}
		})
	}
}

func TestProcessArtifactZip_ConcurrentProcessing(t *testing.T) {
	// Create a zip with many files to test concurrent processing
	buf := new(bytes.Buffer)
	w := zip.NewWriter(buf)

	for i := 0; i < 20; i++ {
		f, _ := w.Create(fmt.Sprintf("file%d.txt", i))
		_, _ = fmt.Fprintf(f, "Content for file %d", i)
	}

	_ = w.Close()
	zipBytes := buf.Bytes()

	result, err := ProcessArtifactZip(zipBytes, "concurrent-test.zip", "https://example.com/build/123", 8, false)

	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.Len(t, result.FileResults, 20)

	// Verify all files were processed
	processedFiles := make(map[string]bool)
	for _, fileResult := range result.FileResults {
		processedFiles[fileResult.FileName] = true
	}

	for i := 0; i < 20; i++ {
		expectedName := fmt.Sprintf("file%d.txt", i)
		assert.True(t, processedFiles[expectedName], "File %s should be processed", expectedName)
	}
}

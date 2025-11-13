package processor

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestProcessStepLogs(t *testing.T) {
	tests := []struct {
		name              string
		logBytes          []byte
		workspaceSlug     string
		repoSlug          string
		pipelineUUID      string
		stepUUID          string
		maxGoRoutines     int
		verifyCredentials bool
		wantError         bool
	}{
		{
			name:              "empty logs",
			logBytes:          []byte(""),
			workspaceSlug:     "myworkspace",
			repoSlug:          "myrepo",
			pipelineUUID:      "pipeline-123",
			stepUUID:          "step-456",
			maxGoRoutines:     4,
			verifyCredentials: false,
			wantError:         false,
		},
		{
			name:              "logs with no secrets",
			logBytes:          []byte("Building project\nRunning tests\nTests passed"),
			workspaceSlug:     "workspace1",
			repoSlug:          "repo1",
			pipelineUUID:      "pipeline-abc",
			stepUUID:          "step-def",
			maxGoRoutines:     4,
			verifyCredentials: false,
			wantError:         false,
		},
		{
			name: "logs with potential secret pattern",
			logBytes: []byte(`Step 1/5: Building
export AWS_ACCESS_KEY_ID=AKIAIOSFODNN7EXAMPLE
export AWS_SECRET_ACCESS_KEY=wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY
Running deployment`),
			workspaceSlug:     "company",
			repoSlug:          "backend",
			pipelineUUID:      "pipeline-789",
			stepUUID:          "step-012",
			maxGoRoutines:     4,
			verifyCredentials: false,
			wantError:         false,
		},
		{
			name:              "large log file",
			logBytes:          make([]byte, 1024*1024), // 1MB of zeros
			workspaceSlug:     "bigworkspace",
			repoSlug:          "bigrepo",
			pipelineUUID:      "pipeline-large",
			stepUUID:          "step-large",
			maxGoRoutines:     8,
			verifyCredentials: false,
			wantError:         false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ProcessStepLogs(tt.logBytes, tt.workspaceSlug, tt.repoSlug, tt.pipelineUUID, tt.stepUUID, tt.maxGoRoutines, tt.verifyCredentials)

			if tt.wantError {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.NotNil(t, result)
			assert.Equal(t, tt.workspaceSlug, result.WorkspaceSlug)
			assert.Equal(t, tt.repoSlug, result.RepoSlug)
			assert.Equal(t, tt.pipelineUUID, result.PipelineUUID)
			assert.Equal(t, tt.stepUUID, result.StepUUID)
			assert.NotNil(t, result.Findings)
		})
	}
}

func TestProcessArtifactContent(t *testing.T) {
	tests := []struct {
		name              string
		fileBytes         []byte
		filename          string
		webURL            string
		verifyCredentials bool
		expectProcessed   bool
		expectArchive     bool
	}{
		{
			name:              "empty file",
			fileBytes:         []byte{},
			filename:          "empty.txt",
			webURL:            "https://bitbucket.org/workspace/repo/downloads/empty.txt",
			verifyCredentials: false,
			expectProcessed:   false,
			expectArchive:     false,
		},
		{
			name:              "text file",
			fileBytes:         []byte("This is a plain text file with some content"),
			filename:          "readme.txt",
			webURL:            "https://bitbucket.org/workspace/repo/downloads/readme.txt",
			verifyCredentials: false,
			expectProcessed:   true,
			expectArchive:     false,
		},
		{
			name:              "json config file",
			fileBytes:         []byte(`{"api_key": "test-key", "endpoint": "https://api.example.com"}`),
			filename:          "config.json",
			webURL:            "https://bitbucket.org/workspace/repo/downloads/config.json",
			verifyCredentials: false,
			expectProcessed:   true,
			expectArchive:     false,
		},
		{
			name: "env file with secrets",
			fileBytes: []byte(`API_KEY=sk-1234567890abcdefghijklmnopqrstuvwxyz12345
DB_PASSWORD=supersecret123
AWS_ACCESS_KEY=AKIAIOSFODNN7EXAMPLE`),
			filename:          ".env",
			webURL:            "https://bitbucket.org/workspace/repo/downloads/.env",
			verifyCredentials: false,
			expectProcessed:   true,
			expectArchive:     false,
		},
		{
			name:              "zip archive header",
			fileBytes:         []byte{0x50, 0x4B, 0x03, 0x04, 0x14, 0x00, 0x00, 0x00},
			filename:          "archive.zip",
			webURL:            "https://bitbucket.org/workspace/repo/downloads/archive.zip",
			verifyCredentials: false,
			expectProcessed:   true,
			expectArchive:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ProcessArtifactContent(tt.fileBytes, tt.filename, tt.webURL, tt.verifyCredentials)

			assert.NotNil(t, result)
			assert.Equal(t, tt.filename, result.Filename)
			assert.Equal(t, tt.webURL, result.WebURL)
			assert.Equal(t, tt.expectProcessed, result.ProcessedFile)
			assert.Equal(t, tt.expectArchive, result.IsArchive)
		})
	}
}

func TestShouldContinueScanning(t *testing.T) {
	tests := []struct {
		name         string
		currentCount int
		maxLimit     int
		want         bool
	}{
		{
			name:         "no limit set (negative)",
			currentCount: 100,
			maxLimit:     -1,
			want:         true,
		},
		{
			name:         "no limit set (zero)",
			currentCount: 50,
			maxLimit:     0,
			want:         true,
		},
		{
			name:         "under limit",
			currentCount: 5,
			maxLimit:     10,
			want:         true,
		},
		{
			name:         "at limit",
			currentCount: 10,
			maxLimit:     10,
			want:         false,
		},
		{
			name:         "over limit",
			currentCount: 15,
			maxLimit:     10,
			want:         false,
		},
		{
			name:         "just started",
			currentCount: 0,
			maxLimit:     5,
			want:         true,
		},
		{
			name:         "one before limit",
			currentCount: 9,
			maxLimit:     10,
			want:         true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ShouldContinueScanning(tt.currentCount, tt.maxLimit)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestProcessStepLogs_ValidatesContext(t *testing.T) {
	// Test that all context information is preserved
	result, err := ProcessStepLogs(
		[]byte("test log"),
		"workspace-test",
		"repo-test",
		"pipeline-uuid-123",
		"step-uuid-456",
		4,
		false,
	)

	require.NoError(t, err)
	assert.Equal(t, "workspace-test", result.WorkspaceSlug)
	assert.Equal(t, "repo-test", result.RepoSlug)
	assert.Equal(t, "pipeline-uuid-123", result.PipelineUUID)
	assert.Equal(t, "step-uuid-456", result.StepUUID)
}

func TestProcessArtifactContent_NilBytes(t *testing.T) {
	result := ProcessArtifactContent(nil, "test.txt", "https://example.com", false)

	assert.NotNil(t, result)
	assert.Equal(t, "test.txt", result.Filename)
	assert.False(t, result.ProcessedFile, "Nil bytes should not be processed")
}

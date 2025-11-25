package url

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBuildPipelineStepArtifactURL(t *testing.T) {
	tests := []struct {
		name          string
		workspaceSlug string
		repoSlug      string
		buildNumber   int
		stepUUID      string
		expectedURL   string
	}{
		{
			name:          "builds valid artifact URL",
			workspaceSlug: "myworkspace",
			repoSlug:      "myrepo",
			buildNumber:   123,
			stepUUID:      "uuid-1234",
			expectedURL:   "https://bitbucket.org/repositories/myworkspace/myrepo/pipelines/results/123/steps/uuid-1234/artifacts",
		},
		{
			name:          "handles special characters in slug",
			workspaceSlug: "my-workspace",
			repoSlug:      "my-repo",
			buildNumber:   456,
			stepUUID:      "uuid-5678",
			expectedURL:   "https://bitbucket.org/repositories/my-workspace/my-repo/pipelines/results/456/steps/uuid-5678/artifacts",
		},
		{
			name:          "handles build number zero",
			workspaceSlug: "workspace",
			repoSlug:      "repo",
			buildNumber:   0,
			stepUUID:      "uuid",
			expectedURL:   "https://bitbucket.org/repositories/workspace/repo/pipelines/results/0/steps/uuid/artifacts",
		},
		{
			name:          "handles large build number",
			workspaceSlug: "workspace",
			repoSlug:      "repo",
			buildNumber:   999999,
			stepUUID:      "uuid",
			expectedURL:   "https://bitbucket.org/repositories/workspace/repo/pipelines/results/999999/steps/uuid/artifacts",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := BuildPipelineStepArtifactURL("https://bitbucket.org", tt.workspaceSlug, tt.repoSlug, tt.buildNumber, tt.stepUUID)
			assert.Equal(t, tt.expectedURL, result)
		})
	}
}

func TestGetWebBaseURL(t *testing.T) {
	tests := []struct {
		name     string
		apiURL   string
		expected string
	}{
		{name: "standard api url", apiURL: "https://api.bitbucket.org/2.0", expected: "https://bitbucket.org"},
		{name: "without version suffix", apiURL: "https://api.bitbucket.org", expected: "https://bitbucket.org"},
		{name: "already web url", apiURL: "https://bitbucket.org", expected: "https://bitbucket.org"},
		{name: "custom host with api prefix", apiURL: "https://api.example.com/2.0", expected: "https://example.com"},
		{name: "custom host without api prefix", apiURL: "https://example.com/2.0", expected: "https://example.com"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := GetWebBaseURL(tt.apiURL)
			assert.Equal(t, tt.expected, got)
		})
	}
}

func TestBuildDownloadArtifactWebURL(t *testing.T) {
	urlStr, err := BuildDownloadArtifactWebURL("https://bitbucket.org", "workspace", "repo", "artifact.zip")
	assert.NoError(t, err)
	assert.True(t, strings.HasSuffix(urlStr, "/workspace/repo/downloads/artifact.zip"))
}

func TestBuildDownloadArtifactWebURL_InvalidBase(t *testing.T) {
	_, err := BuildDownloadArtifactWebURL("://bad-url", "w", "r", "a")
	assert.Error(t, err)
}

func TestBuildPipelineStepURL(t *testing.T) {
	// Curly braces should be preserved; trailing slash in base should be trimmed; default base applied when empty
	got := BuildPipelineStepURL("https://bitbucket.org/", "ws", "repo", "{pipeline-uuid}", "{step-uuid}")
	assert.Equal(t, "https://bitbucket.org/ws/repo/pipelines/results/{pipeline-uuid}/steps/{step-uuid}", got)

	gotDefault := BuildPipelineStepURL("", "ws", "repo", "p", "s")
	assert.Equal(t, "https://bitbucket.org/ws/repo/pipelines/results/p/steps/s", gotDefault)
}

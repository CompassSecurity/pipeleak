package url

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuildDownloadArtifactWebURL(t *testing.T) {
	tests := []struct {
		name          string
		workspaceSlug string
		repoSlug      string
		artifactName  string
		want          string
		wantError     bool
	}{
		{
			name:          "simple artifact",
			workspaceSlug: "myworkspace",
			repoSlug:      "myrepo",
			artifactName:  "artifact.zip",
			want:          "https://bitbucket.org/myworkspace/myrepo/downloads/artifact.zip",
			wantError:     false,
		},
		{
			name:          "artifact with version",
			workspaceSlug: "company",
			repoSlug:      "project",
			artifactName:  "release-v1.2.3.tar.gz",
			want:          "https://bitbucket.org/company/project/downloads/release-v1.2.3.tar.gz",
			wantError:     false,
		},
		{
			name:          "workspace with hyphen",
			workspaceSlug: "my-workspace",
			repoSlug:      "my-repo",
			artifactName:  "build.zip",
			want:          "https://bitbucket.org/my-workspace/my-repo/downloads/build.zip",
			wantError:     false,
		},
		{
			name:          "artifact with spaces",
			workspaceSlug: "workspace",
			repoSlug:      "repo",
			artifactName:  "my artifact.zip",
			want:          "https://bitbucket.org/workspace/repo/downloads/my%20artifact.zip",
			wantError:     false,
		},
		{
			name:          "empty workspace",
			workspaceSlug: "",
			repoSlug:      "repo",
			artifactName:  "artifact.zip",
			want:          "https://bitbucket.org/repo/downloads/artifact.zip",
			wantError:     false,
		},
		{
			name:          "empty artifact name",
			workspaceSlug: "workspace",
			repoSlug:      "repo",
			artifactName:  "",
			want:          "https://bitbucket.org/workspace/repo/downloads",
			wantError:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := BuildDownloadArtifactWebURL(tt.workspaceSlug, tt.repoSlug, tt.artifactName)

			if tt.wantError {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestBuildPipelineStepURL(t *testing.T) {
	tests := []struct {
		name         string
		workspaceSlug string
		repoSlug     string
		pipelineUUID string
		stepUUID     string
		want         string
	}{
		{
			name:         "typical pipeline step",
			workspaceSlug: "myworkspace",
			repoSlug:     "myrepo",
			pipelineUUID: "{abc123}",
			stepUUID:     "{def456}",
			want:         "https://bitbucket.org/myworkspace/myrepo/pipelines/results/{abc123}/steps/{def456}",
		},
		{
			name:         "pipeline with numbers",
			workspaceSlug: "company",
			repoSlug:     "project",
			pipelineUUID: "12345",
			stepUUID:     "67890",
			want:         "https://bitbucket.org/company/project/pipelines/results/12345/steps/67890",
		},
		{
			name:         "workspace with special characters",
			workspaceSlug: "my-workspace_2024",
			repoSlug:     "my.repo",
			pipelineUUID: "pipeline-uuid-123",
			stepUUID:     "step-uuid-456",
			want:         "https://bitbucket.org/my-workspace_2024/my.repo/pipelines/results/pipeline-uuid-123/steps/step-uuid-456",
		},
		{
			name:         "empty strings",
			workspaceSlug: "",
			repoSlug:     "",
			pipelineUUID: "",
			stepUUID:     "",
			want:         "https://bitbucket.org///pipelines/results//steps/",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := BuildPipelineStepURL(tt.workspaceSlug, tt.repoSlug, tt.pipelineUUID, tt.stepUUID)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestBuildDownloadArtifactWebURL_URLEncoding(t *testing.T) {
	// Test that special characters are properly URL encoded
	got, err := BuildDownloadArtifactWebURL("workspace", "repo", "file with spaces & special chars.zip")
	require.NoError(t, err)
	
	// The URL should have encoded spaces
	assert.Contains(t, got, "%20")
	assert.Contains(t, got, "workspace/repo/downloads")
}

func TestBuildPipelineStepURL_Consistency(t *testing.T) {
	// Test that the same inputs always produce the same output
	url1 := BuildPipelineStepURL("ws", "repo", "p123", "s456")
	url2 := BuildPipelineStepURL("ws", "repo", "p123", "s456")
	
	assert.Equal(t, url1, url2, "Same inputs should produce same output")
}

package url

import (
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

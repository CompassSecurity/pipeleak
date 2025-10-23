package bitbucket

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestPaginatedResponse(t *testing.T) {
	t.Run("creates paginated response with workspaces", func(t *testing.T) {
		resp := PaginatedResponse[Workspace]{
			Pagelen: 10,
			Page:    1,
			Size:    5,
			Next:    "https://api.bitbucket.org/next",
			Values: []Workspace{
				{UUID: "uuid1", Slug: "workspace1", Name: "Workspace 1"},
			},
		}

		assert.Equal(t, 10, resp.Pagelen)
		assert.Equal(t, 1, resp.Page)
		assert.Equal(t, 5, resp.Size)
		assert.Equal(t, "https://api.bitbucket.org/next", resp.Next)
		assert.Len(t, resp.Values, 1)
	})

	t.Run("creates paginated response with repositories", func(t *testing.T) {
		resp := PaginatedResponse[Repository]{
			Pagelen: 25,
			Page:    2,
			Size:    10,
			Values: []Repository{
				{UUID: "repo-uuid", FullName: "workspace/repo", Name: "repo"},
			},
		}

		assert.Equal(t, 25, resp.Pagelen)
		assert.Equal(t, 2, resp.Page)
		assert.Len(t, resp.Values, 1)
	})

	t.Run("handles empty paginated response", func(t *testing.T) {
		resp := PaginatedResponse[Workspace]{
			Pagelen: 10,
			Page:    1,
			Size:    0,
			Values:  []Workspace{},
		}

		assert.Equal(t, 0, resp.Size)
		assert.Len(t, resp.Values, 0)
	})
}

func TestWorkspace(t *testing.T) {
	t.Run("creates workspace with required fields", func(t *testing.T) {
		now := time.Now()
		ws := Workspace{
			UUID:      "uuid-123",
			Slug:      "my-workspace",
			Name:      "My Workspace",
			IsPrivate: false,
			Type:      "workspace",
			CreatedOn: now,
		}

		assert.Equal(t, "uuid-123", ws.UUID)
		assert.Equal(t, "my-workspace", ws.Slug)
		assert.Equal(t, "My Workspace", ws.Name)
		assert.False(t, ws.IsPrivate)
		assert.Equal(t, "workspace", ws.Type)
		assert.Equal(t, now, ws.CreatedOn)
	})

	t.Run("creates private workspace", func(t *testing.T) {
		ws := Workspace{
			UUID:      "private-uuid",
			Slug:      "private-workspace",
			IsPrivate: true,
		}

		assert.True(t, ws.IsPrivate)
	})
}

func TestRepository(t *testing.T) {
	t.Run("creates repository with required fields", func(t *testing.T) {
		repo := Repository{
			UUID:      "repo-uuid",
			FullName:  "workspace/repository",
			Name:      "repository",
			IsPrivate: false,
			Scm:       "git",
			Type:      "repository",
		}

		assert.Equal(t, "repo-uuid", repo.UUID)
		assert.Equal(t, "workspace/repository", repo.FullName)
		assert.Equal(t, "repository", repo.Name)
		assert.False(t, repo.IsPrivate)
		assert.Equal(t, "git", repo.Scm)
		assert.Equal(t, "repository", repo.Type)
	})

	t.Run("creates private repository", func(t *testing.T) {
		repo := Repository{
			UUID:      "private-repo",
			FullName:  "workspace/private",
			IsPrivate: true,
			Scm:       "hg",
		}

		assert.True(t, repo.IsPrivate)
		assert.Equal(t, "hg", repo.Scm)
	})
}

func TestPipeline(t *testing.T) {
	t.Run("creates pipeline with required fields", func(t *testing.T) {
		pipeline := Pipeline{
			UUID:        "pipeline-uuid",
			BuildNumber: 42,
			CreatedOn:   "2024-01-01T00:00:00Z",
			CompletedOn: "2024-01-01T00:05:00Z",
		}

		assert.Equal(t, "pipeline-uuid", pipeline.UUID)
		assert.Equal(t, 42, pipeline.BuildNumber)
		assert.Equal(t, "2024-01-01T00:00:00Z", pipeline.CreatedOn)
		assert.Equal(t, "2024-01-01T00:05:00Z", pipeline.CompletedOn)
	})
}

func TestPublicRepository(t *testing.T) {
	t.Run("creates public repository", func(t *testing.T) {
		repo := PublicRepository{
			UUID:      "public-repo-uuid",
			FullName:  "workspace/public-repo",
			Name:      "public-repo",
			IsPrivate: false,
		}

		assert.Equal(t, "public-repo-uuid", repo.UUID)
		assert.Equal(t, "workspace/public-repo", repo.FullName)
		assert.False(t, repo.IsPrivate)
	})
}

func TestPipelineStep(t *testing.T) {
	t.Run("creates pipeline step", func(t *testing.T) {
		step := PipelineStep{
			UUID:      "step-uuid",
			StartedOn: "2024-01-01T00:00:00Z",
		}

		assert.Equal(t, "step-uuid", step.UUID)
		assert.Equal(t, "2024-01-01T00:00:00Z", step.StartedOn)
	})
}

func TestArtifact(t *testing.T) {
	t.Run("creates artifact", func(t *testing.T) {
		artifact := Artifact{
			UUID: "artifact-uuid",
			Name: "build.zip",
			Path: "/path/to/artifact",
		}

		assert.Equal(t, "artifact-uuid", artifact.UUID)
		assert.Equal(t, "build.zip", artifact.Name)
		assert.Equal(t, "/path/to/artifact", artifact.Path)
	})
}

func TestDownloadArtifact(t *testing.T) {
	t.Run("creates download artifact", func(t *testing.T) {
		artifact := DownloadArtifact{
			Name:      "artifact.zip",
			Size:      1024,
			Downloads: 5,
			Type:      "download",
		}

		assert.Equal(t, "artifact.zip", artifact.Name)
		assert.Equal(t, 1024, artifact.Size)
		assert.Equal(t, 5, artifact.Downloads)
	})
}

func TestBitBucketScanOptions(t *testing.T) {
	t.Run("creates scan options with defaults", func(t *testing.T) {
		opts := BitBucketScanOptions{
			Username:               "testuser",
			AccessToken:            "token123",
			Verbose:                false,
			MaxScanGoRoutines:      4,
			TruffleHogVerification: true,
			MaxPipelines:           -1,
			Artifacts:              false,
			Owned:                  false,
			Public:                 false,
		}

		assert.Equal(t, "testuser", opts.Username)
		assert.Equal(t, "token123", opts.AccessToken)
		assert.False(t, opts.Verbose)
		assert.Equal(t, 4, opts.MaxScanGoRoutines)
		assert.True(t, opts.TruffleHogVerification)
		assert.Equal(t, -1, opts.MaxPipelines)
		assert.False(t, opts.Artifacts)
		assert.False(t, opts.Owned)
		assert.False(t, opts.Public)
	})

	t.Run("creates scan options with custom values", func(t *testing.T) {
		opts := BitBucketScanOptions{
			Username:               "admin",
			AccessToken:            "secret",
			Verbose:                true,
			ConfidenceFilter:       []string{"high", "medium"},
			MaxScanGoRoutines:      8,
			TruffleHogVerification: false,
			MaxPipelines:           10,
			Workspace:              "myworkspace",
			Owned:                  true,
			Artifacts:              true,
			After:                  "2024-01-01T00:00:00Z",
		}

		assert.Equal(t, "admin", opts.Username)
		assert.True(t, opts.Verbose)
		assert.Len(t, opts.ConfidenceFilter, 2)
		assert.Equal(t, 8, opts.MaxScanGoRoutines)
		assert.False(t, opts.TruffleHogVerification)
		assert.Equal(t, 10, opts.MaxPipelines)
		assert.Equal(t, "myworkspace", opts.Workspace)
		assert.True(t, opts.Owned)
		assert.True(t, opts.Artifacts)
		assert.Equal(t, "2024-01-01T00:00:00Z", opts.After)
	})
}

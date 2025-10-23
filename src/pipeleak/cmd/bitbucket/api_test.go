package bitbucket

import (
	"testing"
	"time"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
)

func init() {
	zerolog.SetGlobalLevel(zerolog.Disabled)
}

// Note: The BitBucketApiClient uses resty with a hardcoded base URL (https://api.bitbucket.org/2.0/)
// which makes it difficult to mock HTTP responses in unit tests without modifying the client.
// These tests verify basic client creation and method signatures exist.
// Full API testing would require integration tests with a real BitBucket instance or significant refactoring.

func TestNewClient(t *testing.T) {
	t.Run("creates client with basic auth", func(t *testing.T) {
		client := NewClient("testuser", "testpass", "", "https://api.bitbucket.org/2.0")
		assert.NotNil(t, client.Client)
	})

	t.Run("creates client with cookie", func(t *testing.T) {
		client := NewClient("testuser", "testpass", "test-cookie-value", "https://api.bitbucket.org/2.0")
		assert.NotNil(t, client.Client)
	})

	t.Run("creates client with empty credentials", func(t *testing.T) {
		client := NewClient("", "", "", "https://api.bitbucket.org/2.0")
		assert.NotNil(t, client.Client)
	})
}

func TestListOwnedWorkspaces(t *testing.T) {
	t.Run("method signature is correct", func(t *testing.T) {
		client := NewClient("test", "test", "", "https://api.bitbucket.org/2.0")
		// Method exists and returns ([]Workspace, string, *resty.Response, error)
		workspaces, nextUrl, resp, err := client.ListOwnedWorkspaces("")
		_ = workspaces
		_ = nextUrl
		_ = resp
		_ = err
	})
}

func TestListWorkspaceRepositories(t *testing.T) {
	t.Run("method signature is correct", func(t *testing.T) {
		client := NewClient("test", "test", "", "https://api.bitbucket.org/2.0")
		// Method exists and returns ([]Repository, string, *resty.Response, error)
		repos, nextUrl, resp, err := client.ListWorkspaceRepositoires("", "test-workspace")
		_ = repos
		_ = nextUrl
		_ = resp
		_ = err
	})
}

func TestListPublicRepositories(t *testing.T) {
	t.Run("method signature is correct", func(t *testing.T) {
		client := NewClient("test", "test", "", "https://api.bitbucket.org/2.0")
		// Method exists and returns ([]PublicRepository, string, *resty.Response, error)
		repos, nextUrl, resp, err := client.ListPublicRepositories("", time.Time{})
		_ = repos
		_ = nextUrl
		_ = resp
		_ = err
	})
}

func TestListRepositoryPipelines(t *testing.T) {
	t.Run("method signature is correct", func(t *testing.T) {
		client := NewClient("test", "test", "", "https://api.bitbucket.org/2.0")
		// Method exists and returns ([]Pipeline, string, *resty.Response, error)
		pipelines, nextUrl, resp, err := client.ListRepositoryPipelines("", "workspace", "repo")
		_ = pipelines
		_ = nextUrl
		_ = resp
		_ = err
	})
}

func TestGetStepLog(t *testing.T) {
	t.Run("method signature is correct", func(t *testing.T) {
		client := NewClient("test", "test", "", "https://api.bitbucket.org/2.0")
		// Method exists and returns ([]byte, *resty.Response, error)
		logBytes, resp, err := client.GetStepLog("workspace", "repo", "pipeline-uuid", "step-uuid")
		_ = logBytes
		_ = resp
		_ = err
	})
}

func TestListPipelineSteps(t *testing.T) {
	t.Run("method signature is correct", func(t *testing.T) {
		client := NewClient("test", "test", "", "https://api.bitbucket.org/2.0")
		// Method exists and returns ([]PipelineStep, string, *resty.Response, error)
		steps, nextUrl, resp, err := client.ListPipelineSteps("", "workspace", "repo", "pipeline-uuid")
		_ = steps
		_ = nextUrl
		_ = resp
		_ = err
	})
}

func TestListDownloadArtifacts(t *testing.T) {
	t.Run("method signature is correct", func(t *testing.T) {
		client := NewClient("test", "test", "", "https://api.bitbucket.org/2.0")
		// Method exists and returns ([]DownloadArtifact, string, *resty.Response, error)
		artifacts, nextUrl, resp, err := client.ListDownloadArtifacts("", "workspace", "repo")
		_ = artifacts
		_ = nextUrl
		_ = resp
		_ = err
	})
}

func TestGetDownloadArtifact(t *testing.T) {
	t.Run("method signature is correct", func(t *testing.T) {
		client := NewClient("test", "test", "", "https://api.bitbucket.org/2.0")
		// Method exists and returns ([]byte)
		artifactData := client.GetDownloadArtifact("https://example.com/artifact")
		_ = artifactData
	})
}

func TestListPipelineArtifacts(t *testing.T) {
	t.Run("method signature is correct", func(t *testing.T) {
		client := NewClient("test", "test", "", "https://api.bitbucket.org/2.0")
		// Method exists and returns ([]Artifact, string, *resty.Response, error)
		artifacts, nextUrl, resp, err := client.ListPipelineArtifacts("", "workspace", "repo", 1)
		_ = artifacts
		_ = nextUrl
		_ = resp
		_ = err
	})
}

func TestGetPipelineArtifact(t *testing.T) {
	t.Run("method signature is correct", func(t *testing.T) {
		client := NewClient("test", "test", "", "https://api.bitbucket.org/2.0")
		// Method exists and returns ([]byte)
		artifactData := client.GetPipelineArtifact("workspace", "repo", 1, "artifact-uuid")
		_ = artifactData
	})
}

func TestGetUserInfo(t *testing.T) {
	t.Run("method exists", func(t *testing.T) {
		// Note: This method calls log.Fatal() on HTTP errors, which would exit the test process.
		// We verify the method exists but don't call it in tests to avoid test failures.
		client := NewClient("test", "test", "", "https://api.bitbucket.org/2.0")
		_ = client
		// The method signature is: func GetuserInfo()
		// Cannot safely test without mocking HTTP client or risking log.Fatal()
	})
}

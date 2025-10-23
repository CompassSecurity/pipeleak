package devops

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// Note: The AzureDevOpsApiClient uses resty with various Azure DevOps API endpoints.
// These tests verify basic client creation and method signatures exist.
// Full API testing would require integration tests with a real Azure DevOps instance.

func TestNewClient(t *testing.T) {
	t.Run("creates client with basic auth", func(t *testing.T) {
		client := NewClient("testuser", "testpass")
		assert.NotNil(t, client.Client)
	})

	t.Run("creates client with empty credentials", func(t *testing.T) {
		client := NewClient("", "")
		assert.NotNil(t, client.Client)
	})
}

func TestGetAuthenticatedUser(t *testing.T) {
	t.Run("method signature is correct", func(t *testing.T) {
		client := NewClient("test", "test")
		// Method exists and returns (*AuthenticatedUser, *resty.Response, error)
		user, resp, err := client.GetAuthenticatedUser()
		_ = user
		_ = resp
		_ = err
	})
}

func TestListAccounts(t *testing.T) {
	t.Run("method signature is correct", func(t *testing.T) {
		client := NewClient("test", "test")
		// Method exists and returns ([]Account, *resty.Response, error)
		accounts, resp, err := client.ListAccounts("owner-id")
		_ = accounts
		_ = resp
		_ = err
	})
}

func TestListProjects(t *testing.T) {
	t.Run("method signature is correct", func(t *testing.T) {
		client := NewClient("test", "test")
		// Method exists and returns ([]Project, *resty.Response, string, error)
		projects, resp, continuationToken, err := client.ListProjects("", "myorg")
		_ = projects
		_ = resp
		_ = continuationToken
		_ = err
	})
}

func TestListBuilds(t *testing.T) {
	t.Run("method signature is correct", func(t *testing.T) {
		client := NewClient("test", "test")
		// Method exists and returns ([]Build, *resty.Response, string, error)
		builds, resp, continuationToken, err := client.ListBuilds("", "myorg", "myproject")
		_ = builds
		_ = resp
		_ = continuationToken
		_ = err
	})
}

func TestListBuildLogs(t *testing.T) {
	t.Run("method signature is correct", func(t *testing.T) {
		client := NewClient("test", "test")
		// Method exists and returns ([]BuildLog, *resty.Response, error)
		logs, resp, err := client.ListBuildLogs("myorg", "myproject", 123)
		_ = logs
		_ = resp
		_ = err
	})
}

func TestGetLog(t *testing.T) {
	t.Run("method signature is correct", func(t *testing.T) {
		client := NewClient("test", "test")
		// Method exists and returns ([]byte, *resty.Response, error)
		logData, resp, err := client.GetLog("myorg", "myproject", 123, 10)
		_ = logData
		_ = resp
		_ = err
	})
}

func TestDownloadArtifactZip(t *testing.T) {
	t.Run("method signature is correct", func(t *testing.T) {
		client := NewClient("test", "test")
		// Method exists and returns ([]byte, *resty.Response, error)
		zipData, resp, err := client.DownloadArtifactZip("https://example.com/artifact.zip")
		_ = zipData
		_ = resp
		_ = err
	})
}

func TestListBuildArtifacts(t *testing.T) {
	t.Run("method signature is correct", func(t *testing.T) {
		client := NewClient("test", "test")
		// Method exists and returns ([]Artifact, *resty.Response, string, error)
		artifacts, resp, continuationToken, err := client.ListBuildArtifacts("", "myorg", "myproject", 123)
		_ = artifacts
		_ = resp
		_ = continuationToken
		_ = err
	})
}

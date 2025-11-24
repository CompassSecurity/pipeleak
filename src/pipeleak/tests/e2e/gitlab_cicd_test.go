//go:build e2e
// +build e2e

package e2e

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func setupMockGitLabCicdAPI(t *testing.T) string {
	mux := http.NewServeMux()

	// Project endpoint with CI/CD configuration
	mux.HandleFunc("/api/v4/projects/", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"id":123,"name":"test-project","web_url":"https://gitlab.com/test-project"}`))
	})

	// CI lint endpoint
	mux.HandleFunc("/api/v4/projects/123/ci/lint", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{
			"valid": true,
			"merged_yaml": "stages:\n  - test\n\ntest-job:\n  stage: test\n  script:\n    - echo 'Testing'",
			"warnings": [],
			"errors": []
		}`))
	})

	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)
	return server.URL
}

func TestGLCicdYaml(t *testing.T) {
	apiURL := setupMockGitLabCicdAPI(t)
	stdout, stderr, exitErr := runCLI(t, []string{
		"gl", "cicd", "yaml",
		"--gitlab", apiURL,
		"--token", "mock-token",
		"--project", "test-project",
	}, nil, 10*time.Second)

	assert.Nil(t, exitErr, "CI/CD yaml command should succeed")
	assert.Contains(t, stdout, "test-job", "Should contain job name from YAML")
	assert.Contains(t, stdout, "Done, Bye Bye", "Should show completion message")
	assert.NotContains(t, stderr, "fatal")
}

func TestGLCicdYaml_MissingProject(t *testing.T) {
	apiURL := setupMockGitLabCicdAPI(t)
	_, stderr, exitErr := runCLI(t, []string{
		"gl", "cicd", "yaml",
		"--gitlab", apiURL,
		"--token", "mock-token",
	}, nil, 5*time.Second)

	assert.NotNil(t, exitErr, "Should fail without project flag")
	assert.Contains(t, stderr, "required flag(s)", "Should mention missing required flag")
}

func TestGLCicdYaml_InvalidProject(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v4/projects/", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte(`{"message":"404 Project Not Found"}`))
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	stdout, stderr, _ := runCLI(t, []string{
		"gl", "cicd", "yaml",
		"--gitlab", server.URL,
		"--token", "mock-token",
		"--project", "nonexistent/project",
	}, nil, 10*time.Second)

	// Command completes but may log errors
	assert.Contains(t, stdout+stderr, "Done, Bye Bye", "Should complete execution")
}

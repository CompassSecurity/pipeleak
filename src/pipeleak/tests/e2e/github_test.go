package e2e

import (
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// TestGitHubScan_HappyPath tests successful GitHub Actions scanning
func TestGitHubScan_HappyPath(t *testing.T) {

	server, getRequests, cleanup := startMockServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		
		t.Logf("GitHub Mock: %s %s", r.Method, r.URL.Path)

		switch r.URL.Path {
		case "/api/v3/user/repos":
			w.WriteHeader(http.StatusOK)
			// GitHub API returns an array directly, not wrapped in an object
			_ = json.NewEncoder(w).Encode([]map[string]interface{}{
				{
					"id":        1,
					"name":      "test-repo",
					"full_name": "user/test-repo",
					"html_url":  "https://github.com/user/test-repo",
					"owner":     map[string]interface{}{"login": "user"},
				},
			})

		case "/api/v3/repos/user/test-repo/actions/runs":
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"workflow_runs": []map[string]interface{}{
					{
						"id":            100,
						"status":        "completed",
						"display_title": "Test Workflow Run",
						"html_url":      "https://github.com/user/test-repo/actions/runs/100",
					},
				},
				"total_count": 1,
			})

		case "/api/v3/repos/user/test-repo/actions/runs/100/jobs":
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"jobs": []map[string]interface{}{
					{"id": 1000, "name": "test-job"},
				},
				"total_count": 1,
			})

		default:
			t.Logf("Unmocked path: %s", r.URL.Path)
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(map[string]interface{}{})
		}
	})
	defer cleanup()

	stdout, stderr, exitErr := runCLI(t, []string{
		"gh", "scan",
		"--github", server.URL,
		"--token", "ghp_test_token",
		"--owned",
	}, nil, 10*time.Second)

	assert.Nil(t, exitErr, "GitHub scan should succeed")

	requests := getRequests()
	assert.True(t, len(requests) >= 1, "Should make API requests")

	// Verify Authorization header
	for _, req := range requests {
		authHeader := req.Headers.Get("Authorization")
		if authHeader != "" {
			assert.Contains(t, authHeader, "token", "Should use token authentication")
		}
	}

	t.Logf("STDOUT:\n%s", stdout)
	t.Logf("STDERR:\n%s", stderr)
}

// TestGitHubScan_MissingToken tests missing required flags
func TestGitHubScan_MissingToken(t *testing.T) {

	stdout, stderr, exitErr := runCLI(t, []string{
		"gh", "scan",
		"--github", "https://api.github.com",
	}, nil, 5*time.Second)

	assert.NotNil(t, exitErr, "Should fail without token")

	output := stdout + stderr
	t.Logf("Output:\n%s", output)
}

// TestGitHubScan_InvalidToken tests authentication error
func TestGitHubScan_InvalidToken(t *testing.T) {

	server, _, cleanup := startMockServer(t, withError(http.StatusUnauthorized, "Bad credentials"))
	defer cleanup()

	stdout, stderr, exitErr := runCLI(t, []string{
		"gh", "scan",
		"--github", server.URL,
		"--token", "invalid-token",
	}, nil, 10*time.Second)

	t.Logf("Exit error: %v", exitErr)
	t.Logf("STDOUT:\n%s", stdout)
	t.Logf("STDERR:\n%s", stderr)
}

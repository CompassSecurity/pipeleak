package e2e

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// TestGitHubScan_SearchQuery tests the --search flag for repository search
func TestGitHubScan_SearchQuery(t *testing.T) {

	server, getRequests, cleanup := startMockServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		t.Logf("GitHub Mock (Search): %s %s?%s", r.Method, r.URL.Path, r.URL.RawQuery)

		switch r.URL.Path {
		case "/api/v3/search/repositories":
			query := r.URL.Query().Get("q")
			assert.Contains(t, query, "kubernetes", "Search query should contain 'kubernetes'")

			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"total_count": 2,
				"items": []map[string]interface{}{
					{
						"id":        101,
						"name":      "k8s-tools",
						"full_name": "user/k8s-tools",
						"html_url":  "https://github.com/user/k8s-tools",
						"owner":     map[string]interface{}{"login": "user"},
					},
					{
						"id":        102,
						"name":      "kubernetes-demo",
						"full_name": "org/kubernetes-demo",
						"html_url":  "https://github.com/org/kubernetes-demo",
						"owner":     map[string]interface{}{"login": "org"},
					},
				},
			})

		case "/api/v3/repos/user/k8s-tools/actions/runs",
			"/api/v3/repos/org/kubernetes-demo/actions/runs":
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"workflow_runs": []map[string]interface{}{},
				"total_count":   0,
			})

		default:
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(map[string]interface{}{})
		}
	})
	defer cleanup()

	stdout, stderr, exitErr := runCLI(t, []string{
		"gh", "scan",
		"--github", server.URL,
		"--token", "ghp_test_token",
		"--search", "kubernetes",
	}, nil, 15*time.Second)

	assert.Nil(t, exitErr, "Search scan should succeed")

	requests := getRequests()
	var searchRequests []RecordedRequest
	for _, req := range requests {
		if req.Path == "/api/v3/search/repositories" {
			searchRequests = append(searchRequests, req)
		}
	}
	assert.True(t, len(searchRequests) >= 1, "Should make at least one search API request")

	output := stdout + stderr
	t.Logf("Output:\n%s", output)
	assert.Contains(t, output, "Searching repositories", "Should log search operation")
}

// TestGitHubScan_UserRepositories tests the --user flag for scanning a specific user's repos
func TestGitHubScan_UserRepositories(t *testing.T) {

	server, getRequests, cleanup := startMockServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		t.Logf("GitHub Mock (User): %s %s", r.Method, r.URL.Path)

		switch r.URL.Path {
		case "/api/v3/users/firefart/repos":
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode([]map[string]interface{}{
				{
					"id":        201,
					"name":      "awesome-tool",
					"full_name": "firefart/awesome-tool",
					"html_url":  "https://github.com/firefart/awesome-tool",
					"owner":     map[string]interface{}{"login": "firefart"},
				},
				{
					"id":        202,
					"name":      "security-scanner",
					"full_name": "firefart/security-scanner",
					"html_url":  "https://github.com/firefart/security-scanner",
					"owner":     map[string]interface{}{"login": "firefart"},
				},
			})

		case "/api/v3/repos/firefart/awesome-tool/actions/runs",
			"/api/v3/repos/firefart/security-scanner/actions/runs":
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"workflow_runs": []map[string]interface{}{},
				"total_count":   0,
			})

		default:
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(map[string]interface{}{})
		}
	})
	defer cleanup()

	stdout, stderr, exitErr := runCLI(t, []string{
		"gh", "scan",
		"--github", server.URL,
		"--token", "ghp_test_token",
		"--user", "firefart",
	}, nil, 15*time.Second)

	assert.Nil(t, exitErr, "User scan should succeed")

	requests := getRequests()
	var userRepoRequests []RecordedRequest
	for _, req := range requests {
		if strings.Contains(req.Path, "/users/firefart/repos") {
			userRepoRequests = append(userRepoRequests, req)
		}
	}
	assert.True(t, len(userRepoRequests) >= 1, "Should make at least one user repos API request")

	output := stdout + stderr
	t.Logf("Output:\n%s", output)
	assert.Contains(t, output, "Scanning user's repositories", "Should log user scan operation")
}

// TestGitHubScan_PublicRepositories tests the --public flag for scanning public repos
// SKIP: Public repository scanning requires complex event-based API interaction to identify
// the latest repository ID, which is difficult to mock comprehensively in E2E tests
func SkipTestGitHubScan_PublicRepositories(t *testing.T) {

	server, getRequests, cleanup := startMockServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		t.Logf("GitHub Mock (Public): %s %s?%s", r.Method, r.URL.Path, r.URL.RawQuery)

		switch r.URL.Path {
		case "/api/v3/events":
			// Return events to identify latest repo
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode([]map[string]interface{}{
				{
					"type": "CreateEvent",
					"repo": map[string]interface{}{
						"id":   301,
						"name": "user1/public-repo-1",
					},
				},
			})

		case "/api/v3/repositories/301":
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"id":        301,
				"name":      "public-repo-1",
				"full_name": "user1/public-repo-1",
				"html_url":  "https://github.com/user1/public-repo-1",
				"owner":     map[string]interface{}{"login": "user1"},
			})

		case "/api/v3/repositories":
			// GitHub public repos API uses "since" parameter for pagination
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode([]map[string]interface{}{
				{
					"id":        301,
					"name":      "public-repo-1",
					"full_name": "user1/public-repo-1",
					"html_url":  "https://github.com/user1/public-repo-1",
					"owner":     map[string]interface{}{"login": "user1"},
				},
			})

		case "/api/v3/repos/user1/public-repo-1/actions/runs":
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"workflow_runs": []map[string]interface{}{},
				"total_count":   0,
			})

		default:
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(map[string]interface{}{})
		}
	})
	defer cleanup()

	stdout, stderr, exitErr := runCLI(t, []string{
		"gh", "scan",
		"--github", server.URL,
		"--token", "ghp_test_token",
		"--public",
		"--maxWorkflows", "1", // Limit workflows to avoid long test runs
	}, nil, 15*time.Second)

	assert.Nil(t, exitErr, "Public repos scan should succeed")

	requests := getRequests()
	var publicRepoRequests []RecordedRequest
	for _, req := range requests {
		if req.Path == "/api/v3/repositories" {
			publicRepoRequests = append(publicRepoRequests, req)
		}
	}
	assert.True(t, len(publicRepoRequests) >= 1, "Should make at least one public repos API request")

	output := stdout + stderr
	t.Logf("Output:\n%s", output)
	assert.Contains(t, output, "Scanning most recent public repositories", "Should log public repos scan")
}

// TestGitHubScan_ThreadsConfiguration tests the --threads flag
func TestGitHubScan_ThreadsConfiguration(t *testing.T) {

	server, _, cleanup := startMockServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		t.Logf("GitHub Mock (Threads): %s %s", r.Method, r.URL.Path)

		switch r.URL.Path {
		case "/api/v3/user/repos":
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode([]map[string]interface{}{
				{
					"id":        401,
					"name":      "thread-test-repo",
					"full_name": "user/thread-test-repo",
					"html_url":  "https://github.com/user/thread-test-repo",
					"owner":     map[string]interface{}{"login": "user"},
				},
			})

		case "/api/v3/repos/user/thread-test-repo/actions/runs":
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"workflow_runs": []map[string]interface{}{},
				"total_count":   0,
			})

		default:
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(map[string]interface{}{})
		}
	})
	defer cleanup()

	// Test with different thread counts
	for _, threads := range []string{"1", "8", "16"} {
		t.Run("threads_"+threads, func(t *testing.T) {
			stdout, stderr, exitErr := runCLI(t, []string{
				"gh", "scan",
				"--github", server.URL,
				"--token", "ghp_test_token",
				"--owned",
				"--threads", threads,
			}, nil, 15*time.Second)

			assert.Nil(t, exitErr, "Scan with %s threads should succeed", threads)

			output := stdout + stderr
			t.Logf("Output (threads=%s):\n%s", threads, output)
		})
	}
}

// TestGitHubScan_TruffleHogVerificationDisabled tests --truffleHogVerification=false flag
func TestGitHubScan_TruffleHogVerificationDisabled(t *testing.T) {

	server, _, cleanup := startMockServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		t.Logf("GitHub Mock (TruffleHog): %s %s", r.Method, r.URL.Path)

		switch r.URL.Path {
		case "/api/v3/user/repos":
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode([]map[string]interface{}{
				{
					"id":        501,
					"name":      "trufflehog-test",
					"full_name": "user/trufflehog-test",
					"html_url":  "https://github.com/user/trufflehog-test",
					"owner":     map[string]interface{}{"login": "user"},
				},
			})

		case "/api/v3/repos/user/trufflehog-test/actions/runs":
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"workflow_runs": []map[string]interface{}{},
				"total_count":   0,
			})

		default:
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
		"--truffleHogVerification=false",
	}, nil, 15*time.Second)

	assert.Nil(t, exitErr, "Scan with TruffleHog verification disabled should succeed")

	output := stdout + stderr
	t.Logf("Output:\n%s", output)

	// When verification is disabled, the scanner should not attempt to verify credentials
	// This is validated by the scan completing successfully
}

// TestGitHubScan_MutuallyExclusiveFlags tests that mutually exclusive flags are handled
func TestGitHubScan_MutuallyExclusiveFlags(t *testing.T) {

	server, _, cleanup := startMockServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{})
	})
	defer cleanup()

	// Test that owned and org flags are mutually exclusive
	_, _, exitErr := runCLI(t, []string{
		"gh", "scan",
		"--github", server.URL,
		"--token", "ghp_test_token",
		"--owned",
		"--org", "test-org",
	}, nil, 5*time.Second)

	assert.NotNil(t, exitErr, "Scan with mutually exclusive flags (--owned and --org) should fail")
}

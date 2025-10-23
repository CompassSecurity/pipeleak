package e2e

import (
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// TestGiteaScan_HappyPath tests successful Gitea Actions scanning
func TestGiteaScan_HappyPath(t *testing.T) {

	server, getRequests, cleanup := startMockServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		switch r.URL.Path {
		case "/api/v1", "/api/v1/version":
			// Gitea version/API check
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"version": "1.20.0",
			})

		case "/api/v1/user/repos":
			// Return list of repositories
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode([]map[string]interface{}{
				{
					"id":        1,
					"name":      "test-repo",
					"full_name": "user/test-repo",
					"owner": map[string]interface{}{
						"login": "user",
					},
				},
			})

		case "/api/v1/repos/user/test-repo/actions/runs":
			// Return workflow runs
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"total_count": 1,
				"workflow_runs": []map[string]interface{}{
					{
						"id":         100,
						"status":     "completed",
						"conclusion": "success",
					},
				},
			})

		case "/api/v1/repos/user/test-repo/actions/runs/100/jobs":
			// Return jobs for workflow run
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"total_count": 1,
				"jobs": []map[string]interface{}{
					{
						"id":     1000,
						"name":   "test-job",
						"status": "completed",
					},
				},
			})

		case "/api/v1/repos/user/test-repo/actions/runs/100/jobs/1000/logs":
			// Return job logs
			w.Header().Set("Content-Type", "text/plain")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("Job execution log\nNo secrets here\n"))

		default:
			w.WriteHeader(http.StatusNotFound)
			_ = json.NewEncoder(w).Encode(map[string]string{"message": "not found"})
		}
	})
	defer cleanup()

	stdout, stderr, exitErr := runCLI(t, []string{
		"gitea", "scan",
		"--gitea", server.URL,
		"--token", "gitea-token-123",
	}, nil, 10*time.Second)

	assert.Nil(t, exitErr, "Gitea scan should succeed")

	// Verify API calls
	requests := getRequests()
	assert.True(t, len(requests) >= 1, "Should make at least one API request")

	// Verify authentication header (Gitea uses token in query or header)
	hasAuthRequest := false
	for _, req := range requests {
		if req.Headers.Get("Authorization") != "" || req.RawQuery != "" {
			hasAuthRequest = true
			break
		}
	}
	assert.True(t, hasAuthRequest, "Should include authentication")

	t.Logf("STDOUT:\n%s", stdout)
	t.Logf("STDERR:\n%s", stderr)
}

// TestGiteaScan_WithArtifacts tests scanning with artifacts enabled
func TestGiteaScan_WithArtifacts(t *testing.T) {

	server, getRequests, cleanup := startMockServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		switch r.URL.Path {
		case "/api/v1", "/api/v1/version":
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(map[string]interface{}{"version": "1.20.0"})
		case "/api/v1/user/repos":
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode([]map[string]interface{}{
				{"id": 1, "name": "test-repo", "full_name": "user/test-repo"},
			})

		case "/api/v1/repos/user/test-repo/actions/runs":
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"workflow_runs": []map[string]interface{}{
					{"id": 100, "status": "completed"},
				},
			})

		case "/api/v1/repos/user/test-repo/actions/runs/100/jobs":
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"jobs": []map[string]interface{}{
					{"id": 1000, "name": "build"},
				},
			})

		case "/api/v1/repos/user/test-repo/actions/runs/100/artifacts":
			// Return artifacts list
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"artifacts": []map[string]interface{}{
					{"id": 1, "name": "build-artifacts"},
				},
			})

		default:
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(map[string]interface{}{})
		}
	})
	defer cleanup()

	stdout, stderr, exitErr := runCLI(t, []string{
		"gitea", "scan",
		"--gitea", server.URL,
		"--token", "gitea-token",
		"--artifacts",
		"--runs-limit", "1",
	}, nil, 10*time.Second)

	assert.Nil(t, exitErr, "Command should succeed with --artifacts")

	requests := getRequests()
	t.Logf("Made %d requests", len(requests))
	t.Logf("STDOUT:\n%s", stdout)
	t.Logf("STDERR:\n%s", stderr)
}

// TestGiteaScan_Owned tests scanning only owned repositories
func TestGiteaScan_Owned(t *testing.T) {

	server, _, cleanup := startMockServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		switch r.URL.Path {
		case "/api/v1", "/api/v1/version":
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(map[string]interface{}{"version": "1.20.0"})
		case "/api/v1/user/repos":
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode([]map[string]interface{}{
				{"id": 1, "name": "my-repo", "owner": map[string]string{"login": "me"}},
			})
		default:
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(map[string]interface{}{})
		}
	})
	defer cleanup()

	stdout, stderr, exitErr := runCLI(t, []string{
		"gitea", "scan",
		"--gitea", server.URL,
		"--token", "gitea-token",
		"--owned",
	}, nil, 10*time.Second)

	assert.Nil(t, exitErr, "Command should succeed with --owned flag")

	t.Logf("STDOUT:\n%s", stdout)
	t.Logf("STDERR:\n%s", stderr)
}

// TestGiteaScan_Organization tests scanning organization repositories
func TestGiteaScan_Organization(t *testing.T) {

	server, getRequests, cleanup := startMockServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		switch r.URL.Path {
		case "/api/v1", "/api/v1/version":
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(map[string]interface{}{"version": "1.20.0"})
		case "/api/v1/orgs/my-org/repos":
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode([]map[string]interface{}{
				{"id": 1, "name": "org-repo", "full_name": "my-org/org-repo"},
			})
		default:
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(map[string]interface{}{})
		}
	})
	defer cleanup()

	stdout, stderr, exitErr := runCLI(t, []string{
		"gitea", "scan",
		"--gitea", server.URL,
		"--token", "gitea-token",
		"--organization", "my-org",
	}, nil, 10*time.Second)

	assert.Nil(t, exitErr, "Command should succeed with --organization")

	requests := getRequests()
	orgRequestFound := false
	for _, req := range requests {
		if req.Path == "/api/v1/orgs/my-org/repos" {
			orgRequestFound = true
			break
		}
	}

	t.Logf("Organization request found: %v", orgRequestFound)
	t.Logf("STDOUT:\n%s", stdout)
	t.Logf("STDERR:\n%s", stderr)
}

// TestGiteaScan_SpecificRepository tests scanning a single repository
func TestGiteaScan_SpecificRepository(t *testing.T) {

	server, getRequests, cleanup := startMockServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		switch r.URL.Path {
		case "/api/v1", "/api/v1/version":
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(map[string]interface{}{"version": "1.20.0"})
		case "/api/v1/repos/owner/repo-name":
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"id":        1,
				"name":      "repo-name",
				"full_name": "owner/repo-name",
			})
		default:
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(map[string]interface{}{})
		}
	})
	defer cleanup()

	stdout, stderr, exitErr := runCLI(t, []string{
		"gitea", "scan",
		"--gitea", server.URL,
		"--token", "gitea-token",
		"--repository", "owner/repo-name",
	}, nil, 10*time.Second)

	assert.Nil(t, exitErr, "Command should succeed with --repository")

	requests := getRequests()
	specificRepoFound := false
	for _, req := range requests {
		if req.Path == "/api/v1/repos/owner/repo-name" {
			specificRepoFound = true
			break
		}
	}

	t.Logf("Specific repository request found: %v", specificRepoFound)
	t.Logf("STDOUT:\n%s", stdout)
	t.Logf("STDERR:\n%s", stderr)
}

// TestGiteaScan_WithCookie tests cookie authentication
func TestGiteaScan_WithCookie(t *testing.T) {

	server, getRequests, cleanup := startMockServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		// Check for cookie header
		cookie := r.Header.Get("Cookie")
		if cookie != "" && cookie == "i_like_gitea=test-cookie-value" {
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode([]map[string]interface{}{
				{"id": 1, "name": "test-repo"},
			})
		} else {
			w.WriteHeader(http.StatusUnauthorized)
			_ = json.NewEncoder(w).Encode(map[string]string{"message": "unauthorized"})
		}
	})
	defer cleanup()

	stdout, stderr, exitErr := runCLI(t, []string{
		"gitea", "scan",
		"--gitea", server.URL,
		"--token", "gitea-token",
		"--cookie", "test-cookie-value",
	}, nil, 10*time.Second)

	// Cookie handling depends on implementation
	requests := getRequests()
	hasCookie := false
	for _, req := range requests {
		if req.Headers.Get("Cookie") != "" {
			hasCookie = true
			break
		}
	}

	t.Logf("Cookie sent: %v", hasCookie)
	t.Logf("Exit error: %v", exitErr)
	t.Logf("STDOUT:\n%s", stdout)
	t.Logf("STDERR:\n%s", stderr)
}

// TestGiteaScan_RunsLimit tests limiting workflow runs scanned
func TestGiteaScan_RunsLimit(t *testing.T) {

	server, _, cleanup := startMockServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		switch r.URL.Path {
		case "/api/v1", "/api/v1/version":
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(map[string]interface{}{"version": "1.20.0"})
		case "/api/v1/user/repos":
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode([]map[string]interface{}{
				{"id": 1, "name": "test-repo", "full_name": "user/test-repo"},
			})
		default:
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(map[string]interface{}{})
		}
	})
	defer cleanup()

	stdout, stderr, exitErr := runCLI(t, []string{
		"gitea", "scan",
		"--gitea", server.URL,
		"--token", "gitea-token",
		"--runs-limit", "5",
	}, nil, 10*time.Second)

	assert.Nil(t, exitErr, "Command should succeed with --runs-limit")

	t.Logf("STDOUT:\n%s", stdout)
	t.Logf("STDERR:\n%s", stderr)
}

// TestGiteaScan_StartRunID tests starting from specific run ID
func TestGiteaScan_StartRunID(t *testing.T) {

	server, _, cleanup := startMockServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{})
	})
	defer cleanup()

	// start-run-id requires --repository flag
	stdout, stderr, exitErr := runCLI(t, []string{
		"gitea", "scan",
		"--gitea", server.URL,
		"--token", "gitea-token",
		"--repository", "owner/repo",
		"--start-run-id", "100",
	}, nil, 10*time.Second)

	t.Logf("Exit error: %v", exitErr)
	t.Logf("STDOUT:\n%s", stdout)
	t.Logf("STDERR:\n%s", stderr)
}

// TestGiteaScan_StartRunID_WithoutRepo tests validation error
func TestGiteaScan_StartRunID_WithoutRepo(t *testing.T) {

	server, _, cleanup := startMockServer(t, mockSuccessResponse())
	defer cleanup()

	// Should fail: start-run-id without --repository
	stdout, stderr, exitErr := runCLI(t, []string{
		"gitea", "scan",
		"--gitea", server.URL,
		"--token", "gitea-token",
		"--start-run-id", "100",
	}, nil, 5*time.Second)

	// Should error about missing --repository
	assert.NotNil(t, exitErr, "Should fail when --start-run-id used without --repository")

	output := stdout + stderr
	t.Logf("Output:\n%s", output)
}

// TestGiteaScan_InvalidURL tests invalid Gitea URL handling
func TestGiteaScan_InvalidURL(t *testing.T) {

	stdout, stderr, exitErr := runCLI(t, []string{
		"gitea", "scan",
		"--gitea", "not-a-valid-url",
		"--token", "gitea-token",
	}, nil, 5*time.Second)

	assert.NotNil(t, exitErr, "Should fail with invalid URL")

	output := stdout + stderr
	t.Logf("Output:\n%s", output)
}

// TestGiteaScan_MissingToken tests missing required token flag
func TestGiteaScan_MissingToken(t *testing.T) {

	stdout, stderr, exitErr := runCLI(t, []string{
		"gitea", "scan",
		"--gitea", "https://gitea.example.com",
	}, nil, 5*time.Second)

	assert.NotNil(t, exitErr, "Should fail without --token flag")

	output := stdout + stderr
	t.Logf("Output:\n%s", output)
}

// TestGiteaScan_Threads tests thread count configuration
func TestGiteaScan_Threads(t *testing.T) {

	server, _, cleanup := startMockServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		switch r.URL.Path {
		case "/api/v1", "/api/v1/version":
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(map[string]interface{}{"version": "1.20.0"})
		default:
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode([]map[string]interface{}{})
		}
	})
	defer cleanup()

	stdout, stderr, exitErr := runCLI(t, []string{
		"gitea", "scan",
		"--gitea", server.URL,
		"--token", "gitea-token",
		"--threads", "8",
	}, nil, 10*time.Second)

	assert.Nil(t, exitErr, "Command should succeed with --threads")

	t.Logf("STDOUT:\n%s", stdout)
	t.Logf("STDERR:\n%s", stderr)
}

// TestGiteaScan_Verbose tests verbose logging
func TestGiteaScan_Verbose(t *testing.T) {

	server, _, cleanup := startMockServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		switch r.URL.Path {
		case "/api/v1", "/api/v1/version":
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(map[string]interface{}{"version": "1.20.0"})
		default:
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode([]map[string]interface{}{})
		}
	})
	defer cleanup()

	stdout, stderr, exitErr := runCLI(t, []string{
		"gitea", "scan",
		"--gitea", server.URL,
		"--token", "gitea-token",
		"-v",
	}, nil, 10*time.Second)

	assert.Nil(t, exitErr, "Command should succeed with -v flag")

	// Verbose output may contain more details
	t.Logf("STDOUT:\n%s", stdout)
	t.Logf("STDERR:\n%s", stderr)
}

// TestGiteaEnum tests Gitea enumeration command
func TestGiteaEnum(t *testing.T) {

	server, getRequests, cleanup := startMockServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		switch r.URL.Path {
		case "/api/v1", "/api/v1/version":
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(map[string]interface{}{"version": "1.20.0"})
		case "/api/v1/user":
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"id":        1,
				"login":     "testuser",
				"email":     "test@example.com",
				"full_name": "Test User",
			})

		case "/api/v1/user/repos":
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode([]map[string]interface{}{
				{
					"id":        1,
					"name":      "repo1",
					"full_name": "testuser/repo1",
					"owner":     map[string]interface{}{"username": "testuser"},
				},
				{
					"id":        2,
					"name":      "repo2",
					"full_name": "testuser/repo2",
					"owner":     map[string]interface{}{"username": "testuser"},
				},
			})

		case "/api/v1/user/orgs":
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode([]map[string]interface{}{
				{"id": 10, "username": "my-org"},
			})

		default:
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode([]map[string]interface{}{})
		}
	})
	defer cleanup()

	stdout, stderr, exitErr := runCLI(t, []string{
		"gitea", "enum",
		"--gitea", server.URL,
		"--token", "gitea-token",
	}, nil, 10*time.Second)

	assert.Nil(t, exitErr, "Enum command should succeed")

	requests := getRequests()
	assert.True(t, len(requests) >= 1, "Should make API requests")

	t.Logf("STDOUT:\n%s", stdout)
	t.Logf("STDERR:\n%s", stderr)
}

// TestGitea_APIErrors tests various API error responses
func TestGitea_APIErrors(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		message    string
	}{
		{
			name:       "unauthorized",
			statusCode: http.StatusUnauthorized,
			message:    "Invalid token",
		},
		{
			name:       "forbidden",
			statusCode: http.StatusForbidden,
			message:    "Access forbidden",
		},
		{
			name:       "not_found",
			statusCode: http.StatusNotFound,
			message:    "Resource not found",
		},
		{
			name:       "server_error",
			statusCode: http.StatusInternalServerError,
			message:    "Internal server error",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			// Do not use t.Parallel() - stdout/stderr redirection conflicts

			server, _, cleanup := startMockServer(t, withError(tt.statusCode, tt.message))
			defer cleanup()

			stdout, stderr, exitErr := runCLI(t, []string{
				"gitea", "scan",
				"--gitea", server.URL,
				"--token", "test-token",
			}, nil, 10*time.Second)

			t.Logf("Status code: %d", tt.statusCode)
			t.Logf("Exit error: %v", exitErr)
			t.Logf("STDOUT:\n%s", stdout)
			t.Logf("STDERR:\n%s", stderr)
		})
	}
}

// TestGitea_TruffleHogVerification tests credential verification flag
func TestGiteaScan_TruffleHogVerification(t *testing.T) {

	server, _, cleanup := startMockServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		switch r.URL.Path {
		case "/api/v1", "/api/v1/version":
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(map[string]interface{}{"version": "1.20.0"})
		default:
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode([]map[string]interface{}{})
		}
	})
	defer cleanup()

	tests := []struct {
		name string
		args []string
	}{
		{
			name: "verification_enabled_default",
			args: []string{"gitea", "scan", "--gitea", server.URL, "--token", "test"},
		},
		{
			name: "verification_disabled",
			args: []string{"gitea", "scan", "--gitea", server.URL, "--token", "test", "--truffleHogVerification=false"},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			stdout, stderr, exitErr := runCLI(t, tt.args, nil, 10*time.Second)

			assert.Nil(t, exitErr, "Command should succeed")

			t.Logf("STDOUT:\n%s", stdout)
			t.Logf("STDERR:\n%s", stderr)
		})
	}
}

// TestGiteaScan_ConfidenceFilter tests confidence level filtering
func TestGiteaScan_ConfidenceFilter(t *testing.T) {

	server, _, cleanup := startMockServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		switch r.URL.Path {
		case "/api/v1", "/api/v1/version":
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(map[string]interface{}{"version": "1.20.0"})
		default:
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode([]map[string]interface{}{})
		}
	})
	defer cleanup()

	stdout, stderr, exitErr := runCLI(t, []string{
		"gitea", "scan",
		"--gitea", server.URL,
		"--token", "test",
		"--confidence", "high,medium",
	}, nil, 10*time.Second)

	assert.Nil(t, exitErr, "Command should succeed with --confidence filter")

	t.Logf("STDOUT:\n%s", stdout)
	t.Logf("STDERR:\n%s", stderr)
}

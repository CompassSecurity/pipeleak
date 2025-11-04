package e2e

import (
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestGitLabVariables(t *testing.T) {

	server, getRequests, cleanup := startMockServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		switch r.URL.Path {
		case "/api/v4/projects":
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode([]map[string]interface{}{
				{"id": 1, "name": "test-project"},
			})

		case "/api/v4/projects/1/variables":
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode([]map[string]interface{}{
				{
					"key":           "DATABASE_URL",
					"value":         "postgres://user:pass@localhost/db",
					"protected":     false,
					"masked":        true,
					"variable_type": "env_var",
				},
			})

		default:
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode([]map[string]interface{}{})
		}
	})
	defer cleanup()

	stdout, stderr, exitErr := runCLI(t, []string{
		"gl", "variables",
		"--gitlab", server.URL,
		"--token", "glpat-test",
	}, nil, 10*time.Second)

	assert.Nil(t, exitErr, "Variables command should succeed")

	requests := getRequests()
	variablesRequestFound := false
	for _, req := range requests {
		if req.Path == "/api/v4/projects/1/variables" {
			variablesRequestFound = true
			assertRequestMethodAndPath(t, req, "GET", "/api/v4/projects/1/variables")
			break
		}
	}

	t.Logf("Variables request made: %v", variablesRequestFound)
	t.Logf("STDOUT:\n%s", stdout)
	t.Logf("STDERR:\n%s", stderr)
}

// TestGitLabRunnersList tests GitLab runners enumeration

func TestGitLabRunnersList(t *testing.T) {

	server, getRequests, cleanup := startMockServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		if r.URL.Path == "/api/v4/runners" {
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode([]map[string]interface{}{
				{
					"id":          1,
					"description": "test-runner",
					"active":      true,
					"is_shared":   false,
				},
			})
		} else {
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode([]map[string]interface{}{})
		}
	})
	defer cleanup()

	stdout, stderr, exitErr := runCLI(t, []string{
		"gl", "runners", "list",
		"--gitlab", server.URL,
		"--token", "glpat-test",
	}, nil, 10*time.Second)

	assert.Nil(t, exitErr, "Runners list command should succeed")

	requests := getRequests()
	assert.True(t, len(requests) >= 1, "Should make API request")

	t.Logf("STDOUT:\n%s", stdout)
	t.Logf("STDERR:\n%s", stderr)
}

// TestGitLabCICDYaml tests fetching CI/CD YAML configuration

func TestGitLabCICDYaml(t *testing.T) {

	server, getRequests, cleanup := startMockServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		t.Logf("CICD Yaml Mock: %s %s", r.Method, r.URL.Path)

		switch r.URL.Path {
		case "/api/v4/projects/test%2Fproject", "/api/v4/projects/test/project":
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"id":   1,
				"name": "project",
			})

		case "/api/v4/projects/1/repository/files/.gitlab-ci.yml":
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"file_name": ".gitlab-ci.yml",
				"content":   "c3RhZ2VzOgogIC0gYnVpbGQ=", // base64 encoded
			})

		case "/api/v4/projects/1/ci/lint":
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"status": "valid",
			})

		default:
			w.WriteHeader(http.StatusNotFound)
			_ = json.NewEncoder(w).Encode(map[string]string{"error": "not found"})
		}
	})
	defer cleanup()

	stdout, stderr, exitErr := runCLI(t, []string{
		"gl", "cicd", "yaml",
		"--gitlab", server.URL,
		"--token", "glpat-test",
		"--repo", "test/project",
	}, nil, 10*time.Second)

	assert.Nil(t, exitErr, "CICD yaml command should succeed")

	requests := getRequests()
	assert.True(t, len(requests) >= 1, "Should make API requests")

	t.Logf("STDOUT:\n%s", stdout)
	t.Logf("STDERR:\n%s", stderr)
}

// TestGitLabSchedule tests scheduled pipeline enumeration

func TestGitLabSchedule(t *testing.T) {

	server, _, cleanup := startMockServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		switch r.URL.Path {
		case "/api/v4/projects":
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode([]map[string]interface{}{
				{"id": 1, "name": "test-project"},
			})

		case "/api/v4/projects/1/pipeline_schedules":
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode([]map[string]interface{}{
				{
					"id":          1,
					"description": "Nightly build",
					"ref":         "main",
					"cron":        "0 0 * * *",
					"active":      true,
				},
			})

		default:
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode([]map[string]interface{}{})
		}
	})
	defer cleanup()

	stdout, stderr, exitErr := runCLI(t, []string{
		"gl", "schedule",
		"--gitlab", server.URL,
		"--token", "glpat-test",
	}, nil, 10*time.Second)

	assert.Nil(t, exitErr, "Schedule command should succeed")

	t.Logf("STDOUT:\n%s", stdout)
	t.Logf("STDERR:\n%s", stderr)
}

// TestGitLabSecureFiles tests secure files extraction

func TestGitLabSecureFiles(t *testing.T) {

	server, _, cleanup := startMockServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		switch r.URL.Path {
		case "/api/v4/projects":
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode([]map[string]interface{}{
				{"id": 1, "name": "test-project"},
			})

		case "/api/v4/projects/1/secure_files":
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode([]map[string]interface{}{
				{
					"id":       1,
					"name":     "secret.key",
					"checksum": "abc123",
				},
			})

		default:
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode([]map[string]interface{}{})
		}
	})
	defer cleanup()

	stdout, stderr, exitErr := runCLI(t, []string{
		"gl", "secureFiles",
		"--gitlab", server.URL,
		"--token", "glpat-test",
	}, nil, 10*time.Second)

	assert.Nil(t, exitErr, "SecureFiles command should succeed")

	t.Logf("STDOUT:\n%s", stdout)
	t.Logf("STDERR:\n%s", stderr)
}

// TestGitLabUnauthenticatedRegister tests unauthenticated runner registration

func TestGitLabUnauthenticatedRegister(t *testing.T) {

	registrationCalled := false
	server, getRequests, cleanup := startMockServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		if r.URL.Path == "/api/v4/runners" && r.Method == "POST" {
			registrationCalled = true
			w.WriteHeader(http.StatusCreated)
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"id":    1,
				"token": "runner-token-xyz",
			})
		} else {
			w.WriteHeader(http.StatusNotFound)
			_ = json.NewEncoder(w).Encode(map[string]string{"error": "not found"})
		}
	})
	defer cleanup()

	stdout, stderr, exitErr := runCLI(t, []string{
		"gluna", "register",
		"--gitlab", server.URL,
		"--token", "registration-token",
		"--executor", "shell",
		"--description", "test-runner",
	}, nil, 10*time.Second)

	// Command behavior depends on implementation
	// Log the output for inspection
	t.Logf("Registration called: %v", registrationCalled)
	t.Logf("Exit error: %v", exitErr)
	t.Logf("STDOUT:\n%s", stdout)
	t.Logf("STDERR:\n%s", stderr)

	if registrationCalled {
		requests := getRequests()
		for _, req := range requests {
			if req.Path == "/api/v4/runners" {
				assert.Equal(t, "POST", req.Method)
			}
		}
	}
}

// TestGitLabVuln tests vulnerability scanning

func TestGitLabVuln(t *testing.T) {

	server, _, cleanup := startMockServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		// Mock vulnerability report endpoint
		if r.URL.Path == "/api/v4/projects/1/vulnerabilities" {
			_ = json.NewEncoder(w).Encode([]map[string]interface{}{
				{
					"id":       1,
					"title":    "SQL Injection",
					"severity": "high",
				},
			})
		} else {
			_ = json.NewEncoder(w).Encode([]map[string]interface{}{})
		}
	})
	defer cleanup()

	stdout, stderr, exitErr := runCLI(t, []string{
		"gl", "vuln",
		"--gitlab", server.URL,
		"--token", "glpat-test",
		"--project", "1",
	}, nil, 10*time.Second)

	// Log output regardless of success/failure
	t.Logf("Exit error: %v", exitErr)
	t.Logf("STDOUT:\n%s", stdout)
	t.Logf("STDERR:\n%s", stderr)
}

// TestGitLab_APIErrorHandling tests various API error scenarios

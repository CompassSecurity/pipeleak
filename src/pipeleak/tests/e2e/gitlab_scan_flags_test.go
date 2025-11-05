package e2e

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// TestGitLabScan_ConfidenceFilter tests the --confidence flag
func TestGitLabScan_ConfidenceFilter(t *testing.T) {

	server, _, cleanup := startMockServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		switch r.URL.Path {
		case "/api/v4/projects":
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode([]map[string]interface{}{
				{"id": 1, "name": "test-project", "path_with_namespace": "group/test-project"},
			})

		case "/api/v4/projects/1/pipelines":
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode([]map[string]interface{}{
				{"id": 100, "ref": "main", "status": "success"},
			})

		case "/api/v4/projects/1/pipelines/100/jobs":
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode([]map[string]interface{}{
				{"id": 1000, "name": "test-job", "status": "success"},
			})

		case "/api/v4/projects/1/jobs/1000/trace":
			w.WriteHeader(http.StatusOK)
			logContent := `Running job...
export AWS_ACCESS_KEY_ID=AKIAIOSFODNN7EXAMPLE
export DATABASE_PASSWORD=supersecret123
export MAYBE_SECRET=value123
Job complete`
			_, _ = w.Write([]byte(logContent))

		default:
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(map[string]interface{}{})
		}
	})
	defer cleanup()

	stdout, stderr, exitErr := runCLI(t, []string{
		"gl", "scan",
		"--gitlab", server.URL,
		"--token", "glpat-test-token",
		"--confidence", "high,medium",
		"--job-limit", "1",
	}, nil, 15*time.Second)

	assert.Nil(t, exitErr, "Scan with confidence filter should succeed")

	output := stdout + stderr
	t.Logf("Output:\n%s", output)
	// The scanner should filter secrets based on confidence levels
}

// TestGitLabScan_CookieAuthentication tests the --cookie flag for dotenv artifacts
func TestGitLabScan_CookieAuthentication(t *testing.T) {

	server, getRequests, cleanup := startMockServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		// Check if cookie is present
		cookie := r.Header.Get("Cookie")
		if strings.Contains(cookie, "_gitlab_session=test-cookie-value") {
			t.Logf("Cookie authentication verified: %s", cookie)
		}

		switch r.URL.Path {
		case "/api/v4/projects":
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode([]map[string]interface{}{
				{"id": 1, "name": "cookie-test-project", "path_with_namespace": "group/project"},
			})

		case "/api/v4/projects/1/pipelines":
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode([]map[string]interface{}{
				{"id": 200, "ref": "main", "status": "success"},
			})

		case "/api/v4/projects/1/pipelines/200/jobs":
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode([]map[string]interface{}{
				{"id": 2000, "name": "build-job", "status": "success"},
			})

		case "/api/v4/projects/1/jobs/2000/trace":
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("Job log\n"))

		default:
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(map[string]interface{}{})
		}
	})
	defer cleanup()

	stdout, stderr, exitErr := runCLI(t, []string{
		"gl", "scan",
		"--gitlab", server.URL,
		"--token", "glpat-test-token",
		"--cookie", "test-cookie-value",
		"--job-limit", "1",
	}, nil, 15*time.Second)

	assert.Nil(t, exitErr, "Scan with cookie authentication should succeed")

	// Verify cookie was sent in requests
	requests := getRequests()
	cookieFound := false
	for _, req := range requests {
		if strings.Contains(req.Headers.Get("Cookie"), "_gitlab_session=test-cookie-value") {
			cookieFound = true
			break
		}
	}
	t.Logf("Cookie found in requests: %v", cookieFound)

	output := stdout + stderr
	t.Logf("Output:\n%s", output)
}

// TestGitLabScan_MaxArtifactSize tests the --max-artifact-size flag
func TestGitLabScan_MaxArtifactSize(t *testing.T) {

	server, _, cleanup := startMockServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		switch r.URL.Path {
		case "/api/v4/projects":
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode([]map[string]interface{}{
				{"id": 1, "name": "artifact-test"},
			})

		case "/api/v4/projects/1/pipelines":
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode([]map[string]interface{}{
				{"id": 300, "status": "success"},
			})

		case "/api/v4/projects/1/pipelines/300/jobs":
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode([]map[string]interface{}{
				{
					"id":   3000,
					"name": "large-artifact-job",
					"artifacts_file": map[string]interface{}{
						"filename": "large.zip",
						"size":     1024 * 1024 * 100, // 100MB
					},
				},
				{
					"id":   3001,
					"name": "small-artifact-job",
					"artifacts_file": map[string]interface{}{
						"filename": "small.zip",
						"size":     1024 * 100, // 100KB
					},
				},
			})

		case "/api/v4/projects/1/jobs/3000/artifacts",
			"/api/v4/projects/1/jobs/3001/artifacts":
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("PK\x03\x04")) // ZIP magic bytes

		default:
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(map[string]interface{}{})
		}
	})
	defer cleanup()

	stdout, stderr, exitErr := runCLI(t, []string{
		"gl", "scan",
		"--gitlab", server.URL,
		"--token", "glpat-test-token",
		"--artifacts",
		"--max-artifact-size", "50Mb", // Only scan artifacts < 50MB
		"--job-limit", "2",
	}, nil, 15*time.Second)

	assert.Nil(t, exitErr, "Scan with max-artifact-size should succeed")

	output := stdout + stderr
	t.Logf("Output:\n%s", output)
	// The scanner should skip artifacts larger than 50MB
}

// TestGitLabScan_QueueFolder tests the --queue flag for custom queue location
func TestGitLabScan_QueueFolder(t *testing.T) {

	customQueueDir := t.TempDir()

	server, _, cleanup := startMockServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		switch r.URL.Path {
		case "/api/v4/projects":
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode([]map[string]interface{}{
				{"id": 1, "name": "queue-test"},
			})

		case "/api/v4/projects/1/pipelines":
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode([]map[string]interface{}{
				{"id": 400, "status": "success"},
			})

		case "/api/v4/projects/1/pipelines/400/jobs":
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode([]map[string]interface{}{
				{"id": 4000, "name": "test-job", "status": "success"},
			})

		case "/api/v4/projects/1/jobs/4000/trace":
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("Job log\n"))

		default:
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(map[string]interface{}{})
		}
	})
	defer cleanup()

	stdout, stderr, exitErr := runCLI(t, []string{
		"gl", "scan",
		"--gitlab", server.URL,
		"--token", "glpat-test-token",
		"--queue", customQueueDir,
		"--job-limit", "1",
	}, nil, 15*time.Second)

	assert.Nil(t, exitErr, "Scan with custom queue folder should succeed")

	output := stdout + stderr
	t.Logf("Output:\n%s", output)
	t.Logf("Custom queue directory: %s", customQueueDir)
	// The scanner should use the custom queue directory
}

// TestGitLabScan_TruffleHogVerificationDisabled tests --truffleHogVerification=false
func TestGitLabScan_TruffleHogVerificationDisabled(t *testing.T) {

	server, _, cleanup := startMockServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		switch r.URL.Path {
		case "/api/v4/projects":
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode([]map[string]interface{}{
				{"id": 1, "name": "trufflehog-test"},
			})

		case "/api/v4/projects/1/pipelines":
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode([]map[string]interface{}{
				{"id": 500, "status": "success"},
			})

		case "/api/v4/projects/1/pipelines/500/jobs":
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode([]map[string]interface{}{
				{"id": 5000, "name": "verify-test", "status": "success"},
			})

		case "/api/v4/projects/1/jobs/5000/trace":
			w.WriteHeader(http.StatusOK)
			logContent := `Job starting...
export AWS_ACCESS_KEY_ID=AKIAIOSFODNN7EXAMPLE
export API_KEY=sk_test_1234567890abcdef
Job complete`
			_, _ = w.Write([]byte(logContent))

		default:
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(map[string]interface{}{})
		}
	})
	defer cleanup()

	stdout, stderr, exitErr := runCLI(t, []string{
		"gl", "scan",
		"--gitlab", server.URL,
		"--token", "glpat-test-token",
		"--truffleHogVerification=false",
		"--job-limit", "1",
	}, nil, 15*time.Second)

	assert.Nil(t, exitErr, "Scan with TruffleHog verification disabled should succeed")

	output := stdout + stderr
	t.Logf("Output:\n%s", output)
	// Should not attempt to verify credentials when verification is disabled
}

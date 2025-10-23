package e2e

import (
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// TestBitBucketScan_HappyPath tests BitBucket pipeline scanning with credential detection
func TestBitBucketScan_HappyPath(t *testing.T) {

	server, getRequests, cleanup := startMockServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		t.Logf("BitBucket Mock: %s %s", r.Method, r.URL.Path)

		switch r.URL.Path {
		case "/repositories/test-workspace":
			// Return list of repositories in workspace
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"values": []map[string]interface{}{
					{
						"uuid":       "{repo-uuid-1}",
						"name":       "test-repo",
						"slug":       "test-repo",
						"created_on": "2023-01-01T00:00:00.000000+00:00",
						"updated_on": "2023-01-02T00:00:00.000000+00:00",
						"links": map[string]interface{}{
							"html": map[string]interface{}{
								"href": "https://bitbucket.org/test-workspace/test-repo",
							},
						},
					},
				},
			})

		case "/repositories/test-workspace/test-repo/pipelines":
			// Return list of pipelines
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"values": []map[string]interface{}{
					{
						"uuid":         "{pipeline-uuid-1}",
						"build_number": 1,
						"state": map[string]interface{}{
							"name": "COMPLETED",
						},
					},
				},
			})

		case "/repositories/test-workspace/test-repo/pipelines/{pipeline-uuid-1}/steps":
			// Return pipeline steps
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"values": []map[string]interface{}{
					{
						"uuid": "{step-uuid-1}",
						"name": "Build and Test",
						"state": map[string]interface{}{
							"name": "COMPLETED",
						},
					},
				},
			})

		case "/repositories/test-workspace/test-repo/pipelines/{pipeline-uuid-1}/steps/{step-uuid-1}/log":
			// Return step logs containing credentials
			w.Header().Set("Content-Type", "text/plain")
			w.WriteHeader(http.StatusOK)
			logContent := `+ echo "Starting build process"
Starting build process
+ export DATABASE_URL="postgres://admin:SuperSecret123!@db.example.com:5432/mydb"
+ export AWS_SECRET_ACCESS_KEY="wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY"
+ echo "Running tests..."
Running tests...
+ curl -H "Authorization: Bearer ghp_1234567890abcdefghijklmnopqrstuvwxyz" https://api.github.com/user
{"login": "testuser"}
+ echo "Build completed successfully"
Build completed successfully`
			_, _ = w.Write([]byte(logContent))

		default:
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(map[string]interface{}{"values": []interface{}{}})
		}
	})
	defer cleanup()

	stdout, stderr, exitErr := runCLI(t, []string{
		"bb", "scan",
		"--bitbucket", server.URL,
		"--username", "testuser",
		"--token", "testpass",
		"--workspace", "test-workspace",
	}, nil, 15*time.Second)

	assert.Nil(t, exitErr, "BitBucket scan should succeed")

	requests := getRequests()
	assert.True(t, len(requests) >= 1, "Should make API requests")

	// Verify Basic Auth
	hasAuthRequest := false
	for _, req := range requests {
		authHeader := req.Headers.Get("Authorization")
		if authHeader != "" {
			assert.Contains(t, authHeader, "Basic", "Should use Basic authentication")
			hasAuthRequest = true
		}
	}
	assert.True(t, hasAuthRequest, "Should have authenticated requests")

	// Verify credentials were detected in logs
	output := stdout + stderr
	
	// Check that the scanner detected the secrets
	assert.Contains(t, output, "postgres://", "Should detect PostgreSQL connection string")
	assert.Contains(t, output, "AWS_SECRET_ACCESS_KEY", "Should detect AWS secret key")
	assert.Contains(t, output, "Github", "Should detect GitHub token")
	
	// Verify the scanner logged findings with HIT marker
	assert.Contains(t, output, "HIT", "Should log HIT for secret detection")
	assert.Contains(t, output, "ruleName", "Should log rule name for detected secrets")
	
	// Verify multiple secrets were found
	assert.Contains(t, output, "Password in URL", "Should detect password in database URL")
	assert.Contains(t, output, "Github Personal Access Token", "Should detect GitHub PAT")

	t.Logf("STDOUT:\n%s", stdout)
	t.Logf("STDERR:\n%s", stderr)
}

// TestBitBucketScan_MissingCredentials tests missing credentials
func TestBitBucketScan_MissingCredentials(t *testing.T) {

	stdout, stderr, _ := runCLI(t, []string{
		"bb", "scan",
		"--bitbucket", "https://api.bitbucket.org",
		"--owned", // Need a scan mode
	}, nil, 5*time.Second)

	// The command completes but logs authentication errors
	output := stdout + stderr
	assert.Contains(t, output, "401", "Should show 401 authentication error when credentials missing")
	t.Logf("Output:\n%s", output)
}

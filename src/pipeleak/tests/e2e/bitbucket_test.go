package e2e

import (
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// TestBitBucketScan_HappyPath tests BitBucket pipeline scanning
func TestBitBucketScan_HappyPath(t *testing.T) {

	server, getRequests, cleanup := startMockServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		switch { //nolint:staticcheck
		case r.URL.Path == "/2.0/repositories":
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"values": []map[string]interface{}{
					{"uuid": "repo-1", "name": "test-repo"},
				},
			})

		case r.URL.Path == "/2.0/repositories/test-repo/pipelines/":
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"values": []map[string]interface{}{
					{"uuid": "pipeline-1", "build_number": 1},
				},
			})

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
	}, nil, 10*time.Second)

	assert.Nil(t, exitErr, "BitBucket scan should succeed")

	requests := getRequests()
	assert.True(t, len(requests) >= 1, "Should make API requests")

	// Verify Basic Auth
	for _, req := range requests {
		authHeader := req.Headers.Get("Authorization")
		if authHeader != "" {
			assert.Contains(t, authHeader, "Basic", "Should use Basic authentication")
		}
	}

	t.Logf("STDOUT:\n%s", stdout)
	t.Logf("STDERR:\n%s", stderr)
}

// TestBitBucketScan_MissingCredentials tests missing credentials
func TestBitBucketScan_MissingCredentials(t *testing.T) {

	stdout, stderr, exitErr := runCLI(t, []string{
		"bb", "scan",
		"--bitbucket", "https://api.bitbucket.org",
	}, nil, 5*time.Second)

	assert.NotNil(t, exitErr, "Should fail without credentials")

	output := stdout + stderr
	t.Logf("Output:\n%s", output)
}

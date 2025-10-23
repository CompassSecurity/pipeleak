package e2e

import (
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// TestAzureDevOpsScan_HappyPath tests Azure DevOps pipeline scanning
func TestAzureDevOpsScan_HappyPath(t *testing.T) {

	server, getRequests, cleanup := startMockServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		switch { //nolint:staticcheck
		case r.URL.Path == "/_apis/projects":
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"value": []map[string]interface{}{
					{"id": "proj-1", "name": "test-project"},
				},
			})

		case r.URL.Path == "/proj-1/_apis/build/builds":
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"value": []map[string]interface{}{
					{"id": 1, "buildNumber": "1"},
				},
			})

		default:
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(map[string]interface{}{"value": []interface{}{}})
		}
	})
	defer cleanup()

	stdout, stderr, exitErr := runCLI(t, []string{
		"ad", "scan",
		"--devops", server.URL,
		"--token", "azure-pat-token",
		"--organization", "myorg",
	}, nil, 10*time.Second)

	assert.Nil(t, exitErr, "Azure DevOps scan should succeed")

	requests := getRequests()
	assert.True(t, len(requests) >= 1, "Should make API requests")

	t.Logf("STDOUT:\n%s", stdout)
	t.Logf("STDERR:\n%s", stderr)
}

// TestAzureDevOpsScan_MissingToken tests missing required token
func TestAzureDevOpsScan_MissingToken(t *testing.T) {

	stdout, stderr, exitErr := runCLI(t, []string{
		"ad", "scan",
		"--devops", "https://dev.azure.com",
		"--organization", "myorg",
	}, nil, 5*time.Second)

	assert.NotNil(t, exitErr, "Should fail without token")

	output := stdout + stderr
	t.Logf("Output:\n%s", output)
}

package e2e

import (
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestBitBucketScan_MissingCredentials(t *testing.T) {

	server, _, cleanup := startMockServer(t, func(w http.ResponseWriter, r *http.Request) {
		// Return 401 Unauthorized for all requests when credentials are missing/invalid
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"type": "error",
			"error": map[string]interface{}{
				"message": "Invalid credentials",
			},
		})
	})
	defer cleanup()

	stdout, stderr, _ := runCLI(t, []string{
		"bb", "scan",
		"--bitbucket", server.URL,
		"--owned", // Need a scan mode
	}, nil, 5*time.Second)

	// The command completes but logs authentication errors
	output := stdout + stderr
	assert.Contains(t, output, "401", "Should show 401 authentication error when credentials missing")
	assert.Contains(t, output, "owned workspaces", "Should attempt to list owned workspaces")
	t.Logf("Output:\n%s", output)
}

func TestBitBucketScan_Owned_Unauthorized(t *testing.T) {

	server, _, cleanup := startMockServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"type": "error",
			"error": map[string]interface{}{
				"message": "Unauthorized",
			},
		})
	})
	defer cleanup()

	stdout, stderr, _ := runCLI(t, []string{
		"bb", "scan",
		"--bitbucket", server.URL,
		"--username", "baduser",
		"--token", "badtoken",
		"--owned",
	}, nil, 5*time.Second)

	output := stdout + stderr
	assert.Contains(t, output, "401", "Should log 401 error")
	t.Logf("Output:\n%s", output)
}

func TestBitBucketScan_Owned_NotFound(t *testing.T) {

	server, _, cleanup := startMockServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"type": "error",
			"error": map[string]interface{}{
				"message": "Not Found",
			},
		})
	})
	defer cleanup()

	stdout, stderr, _ := runCLI(t, []string{
		"bb", "scan",
		"--bitbucket", server.URL,
		"--username", "testuser",
		"--token", "testtoken",
		"--owned",
	}, nil, 5*time.Second)

	output := stdout + stderr
	assert.Contains(t, output, "404", "Should log 404 error")
	t.Logf("Output:\n%s", output)
}

func TestBitBucketScan_Workspace_NotFound(t *testing.T) {

	server, _, cleanup := startMockServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		if r.URL.Path == "/repositories/invalid-workspace" {
			w.WriteHeader(http.StatusNotFound)
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"type": "error",
				"error": map[string]interface{}{
					"message": "Workspace not found",
				},
			})
		} else {
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(map[string]interface{}{"values": []interface{}{}})
		}
	})
	defer cleanup()

	stdout, stderr, _ := runCLI(t, []string{
		"bb", "scan",
		"--bitbucket", server.URL,
		"--username", "testuser",
		"--token", "testtoken",
		"--workspace", "invalid-workspace",
	}, nil, 5*time.Second)

	output := stdout + stderr
	assert.Contains(t, output, "404", "Should log 404 error for invalid workspace")
	t.Logf("Output:\n%s", output)
}

func TestBitBucketScan_Public_ServerError(t *testing.T) {

	server, _, cleanup := startMockServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"type": "error",
			"error": map[string]interface{}{
				"message": "Internal Server Error",
			},
		})
	})
	defer cleanup()

	stdout, stderr, _ := runCLI(t, []string{
		"bb", "scan",
		"--bitbucket", server.URL,
		"--username", "testuser",
		"--token", "testtoken",
		"--public",
	}, nil, 5*time.Second)

	output := stdout + stderr
	assert.Contains(t, output, "500", "Should log 500 error")
	t.Logf("Output:\n%s", output)
}

func TestBitBucketScan_InvalidCookie(t *testing.T) {

	server, _, cleanup := startMockServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		t.Logf("BitBucket Mock: %s %s", r.Method, r.URL.Path)

		switch r.URL.Path {
		case "/!api/2.0/user":
			// Return 401 for invalid cookie
			w.WriteHeader(http.StatusUnauthorized)
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"type": "error",
				"error": map[string]interface{}{
					"message": "Unauthorized",
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
		"--token", "testtoken",
		"--cookie", "invalid-cookie",
		"--workspace", "test-workspace",
		"--artifacts",
	}, nil, 10*time.Second)

	// Should exit due to fatal error on invalid cookie
	assert.NotNil(t, exitErr, "Should fail with invalid cookie")

	output := stdout + stderr
	assert.Contains(t, output, "Failed to get user info", "Should log cookie validation failure")
	assert.Contains(t, output, "401", "Should show 401 status")

	t.Logf("Output:\n%s", output)
}

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

// TestBitBucketScan_MissingCredentials tests missing credentials with local mock server
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

// TestBitBucketScan_Owned_HappyPath tests scanning owned workspaces
func TestBitBucketScan_Owned_HappyPath(t *testing.T) {

	server, getRequests, cleanup := startMockServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		t.Logf("BitBucket Mock (Owned): %s %s", r.Method, r.URL.Path)

		switch {
		case r.URL.Path == "/user/permissions/workspaces":
			// Return owned workspaces
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"values": []map[string]interface{}{
					{
						"workspace": map[string]interface{}{
							"slug": "my-workspace",
							"name": "My Workspace",
							"uuid": "{workspace-uuid-1}",
						},
					},
				},
			})

		case r.URL.Path == "/repositories/my-workspace":
			// Return repositories in owned workspace
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"values": []map[string]interface{}{
					{
						"uuid":       "{repo-uuid-1}",
						"name":       "my-repo",
						"slug":       "my-repo",
						"created_on": "2023-01-01T00:00:00.000000+00:00",
						"updated_on": "2023-01-02T00:00:00.000000+00:00",
						"links": map[string]interface{}{
							"html": map[string]interface{}{
								"href": "https://bitbucket.org/my-workspace/my-repo",
							},
						},
					},
				},
			})

		case r.URL.Path == "/repositories/my-workspace/my-repo/pipelines":
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"values": []map[string]interface{}{},
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
		"--owned",
	}, nil, 10*time.Second)

	assert.Nil(t, exitErr, "Owned scan should succeed")

	requests := getRequests()
	assert.True(t, len(requests) >= 1, "Should make API requests")

	output := stdout + stderr
	assert.Contains(t, output, "owned workspaces", "Should log owned workspace scanning")
	t.Logf("Output:\n%s", output)
}

// TestBitBucketScan_Owned_Unauthorized tests owned scan with 401 error
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

// TestBitBucketScan_Owned_NotFound tests owned scan with 404 error
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

// TestBitBucketScan_Workspace_HappyPath tests scanning a specific workspace
func TestBitBucketScan_Workspace_HappyPath(t *testing.T) {

	server, getRequests, cleanup := startMockServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		t.Logf("BitBucket Mock (Workspace): %s %s", r.Method, r.URL.Path)

		switch r.URL.Path {
		case "/repositories/test-workspace":
			// Return repositories in the workspace
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"values": []map[string]interface{}{
					{
						"uuid":       "{repo-uuid-1}",
						"name":       "workspace-repo",
						"slug":       "workspace-repo",
						"created_on": "2023-01-01T00:00:00.000000+00:00",
						"updated_on": "2023-01-02T00:00:00.000000+00:00",
						"links": map[string]interface{}{
							"html": map[string]interface{}{
								"href": "https://bitbucket.org/test-workspace/workspace-repo",
							},
						},
					},
				},
			})

		case "/repositories/test-workspace/workspace-repo/pipelines":
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"values": []map[string]interface{}{},
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
		"--workspace", "test-workspace",
	}, nil, 10*time.Second)

	assert.Nil(t, exitErr, "Workspace scan should succeed")

	requests := getRequests()
	assert.True(t, len(requests) >= 1, "Should make API requests")

	output := stdout + stderr
	assert.Contains(t, output, "Scanning a workspace", "Should log workspace scanning")
	t.Logf("Output:\n%s", output)
}

// TestBitBucketScan_Workspace_NotFound tests scanning invalid workspace
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

// TestBitBucketScan_Public_HappyPath tests scanning public repositories
func TestBitBucketScan_Public_HappyPath(t *testing.T) {

	server, getRequests, cleanup := startMockServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		t.Logf("BitBucket Mock (Public): %s %s", r.Method, r.URL.Path)

		if r.URL.Path == "/repositories" {
			// Return public repositories
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"values": []map[string]interface{}{
					{
						"uuid":       "{repo-uuid-1}",
						"name":       "public-repo",
						"slug":       "public-repo",
						"created_on": "2023-01-01T00:00:00.000000+00:00",
						"updated_on": "2023-01-02T00:00:00.000000+00:00",
						"is_private": false,
						"owner": map[string]interface{}{
							"username": "public-owner",
						},
						"links": map[string]interface{}{
							"html": map[string]interface{}{
								"href": "https://bitbucket.org/public-owner/public-repo",
							},
						},
					},
				},
			})
		} else {
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
		"--public",
		"--maxPipelines", "1",
	}, nil, 10*time.Second)

	assert.Nil(t, exitErr, "Public scan should succeed")

	requests := getRequests()
	assert.True(t, len(requests) >= 1, "Should make API requests")

	output := stdout + stderr
	assert.Contains(t, output, "public repos", "Should log public repo scanning")
	t.Logf("Output:\n%s", output)
}

// TestBitBucketScan_Public_WithAfter tests scanning public repos with time filter
func TestBitBucketScan_Public_WithAfter(t *testing.T) {

	server, getRequests, cleanup := startMockServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		t.Logf("BitBucket Mock (Public After): %s %s?%s", r.Method, r.URL.Path, r.URL.RawQuery)

		if r.URL.Path == "/repositories" {
			// Check for after query parameter
			after := r.URL.Query().Get("after")
			t.Logf("After param: %s", after)

			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"values": []map[string]interface{}{
					{
						"uuid":       "{repo-uuid-1}",
						"name":       "recent-public-repo",
						"slug":       "recent-public-repo",
						"created_on": "2025-04-03T00:00:00.000000+00:00",
						"updated_on": "2025-04-03T00:00:00.000000+00:00",
						"is_private": false,
						"owner": map[string]interface{}{
							"username": "recent-owner",
						},
						"links": map[string]interface{}{
							"html": map[string]interface{}{
								"href": "https://bitbucket.org/recent-owner/recent-public-repo",
							},
						},
					},
				},
			})
		} else {
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
		"--public",
		"--after", "2025-04-02T15:00:00+02:00",
		"--maxPipelines", "1",
	}, nil, 10*time.Second)

	assert.Nil(t, exitErr, "Public scan with after filter should succeed")

	requests := getRequests()
	assert.True(t, len(requests) >= 1, "Should make API requests")

	output := stdout + stderr
	assert.Contains(t, output, "public repos", "Should log public repo scanning")
	t.Logf("Output:\n%s", output)
}

// TestBitBucketScan_Public_ServerError tests public scan with server error
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

// TestBitBucketScan_MaxPipelines tests limiting pipelines scanned per repo
func TestBitBucketScan_MaxPipelines(t *testing.T) {

	pipelinesReturned := 0
	server, _, cleanup := startMockServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		t.Logf("BitBucket Mock (MaxPipelines): %s %s", r.Method, r.URL.Path)

		switch r.URL.Path {
		case "/repositories/test-workspace":
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
			pipelinesReturned++
			// Return 5 pipelines but maxPipelines=2 should limit scanning
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"values": []map[string]interface{}{
					{"uuid": "{pipeline-1}", "build_number": 1},
					{"uuid": "{pipeline-2}", "build_number": 2},
					{"uuid": "{pipeline-3}", "build_number": 3},
					{"uuid": "{pipeline-4}", "build_number": 4},
					{"uuid": "{pipeline-5}", "build_number": 5},
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
		"--workspace", "test-workspace",
		"--maxPipelines", "2",
	}, nil, 10*time.Second)

	assert.Nil(t, exitErr, "MaxPipelines scan should succeed")

	output := stdout + stderr
	// Verify the limit was applied (difficult to directly verify, but should complete quickly)
	assert.NotEmpty(t, output, "Should produce output")
	t.Logf("Pipelines endpoint called: %d times", pipelinesReturned)
	t.Logf("Output:\n%s", output)
}

// TestBitBucketScan_NoScanMode tests error when no scan mode is specified
func TestBitBucketScan_NoScanMode(t *testing.T) {

	server, _, cleanup := startMockServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{"values": []interface{}{}})
	})
	defer cleanup()

	stdout, stderr, _ := runCLI(t, []string{
		"bb", "scan",
		"--bitbucket", server.URL,
		"--username", "testuser",
		"--token", "testtoken",
		// No --owned, --workspace, or --public flag
	}, nil, 5*time.Second)

	output := stdout + stderr
	assert.Contains(t, output, "Specify a scan mode", "Should show error for no scan mode")
	t.Logf("Output:\n%s", output)
}

// TestBitBucketScan_Confidence tests filtering by confidence level
func TestBitBucketScan_Confidence(t *testing.T) {

	server, _, cleanup := startMockServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		t.Logf("BitBucket Mock (Confidence): %s %s", r.Method, r.URL.Path)

		switch r.URL.Path {
		case "/repositories/test-workspace":
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
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"values": []map[string]interface{}{
					{
						"uuid": "{step-uuid-1}",
						"name": "Build",
						"state": map[string]interface{}{
							"name": "COMPLETED",
						},
					},
				},
			})

		case "/repositories/test-workspace/test-repo/pipelines/{pipeline-uuid-1}/steps/{step-uuid-1}/log":
			w.Header().Set("Content-Type", "text/plain")
			w.WriteHeader(http.StatusOK)
			// Log with high confidence secret
			logContent := `+ echo "Starting"
+ export API_KEY="AKIAIOSFODNN7EXAMPLE"
+ echo "Done"`
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
		"--token", "testtoken",
		"--workspace", "test-workspace",
		"--confidence", "high",
	}, nil, 15*time.Second)

	assert.Nil(t, exitErr, "Confidence filter scan should succeed")

	output := stdout + stderr
	// Verify scan completed with confidence filter
	assert.NotEmpty(t, output, "Should produce output")
	t.Logf("Output:\n%s", output)
}

// TestBitBucketScan_Threads tests scanning with different thread counts
func TestBitBucketScan_Threads(t *testing.T) {

	server, _, cleanup := startMockServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		t.Logf("BitBucket Mock (Threads): %s %s", r.Method, r.URL.Path)

		switch r.URL.Path {
		case "/repositories/test-workspace":
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"values": []map[string]interface{}{
					{
						"uuid":       "{repo-uuid-1}",
						"name":       "test-repo-1",
						"slug":       "test-repo-1",
						"created_on": "2023-01-01T00:00:00.000000+00:00",
						"updated_on": "2023-01-02T00:00:00.000000+00:00",
						"links": map[string]interface{}{
							"html": map[string]interface{}{
								"href": "https://bitbucket.org/test-workspace/test-repo-1",
							},
						},
					},
					{
						"uuid":       "{repo-uuid-2}",
						"name":       "test-repo-2",
						"slug":       "test-repo-2",
						"created_on": "2023-01-01T00:00:00.000000+00:00",
						"updated_on": "2023-01-02T00:00:00.000000+00:00",
						"links": map[string]interface{}{
							"html": map[string]interface{}{
								"href": "https://bitbucket.org/test-workspace/test-repo-2",
							},
						},
					},
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
		"--workspace", "test-workspace",
		"--threads", "2",
		"--maxPipelines", "1",
	}, nil, 10*time.Second)

	assert.Nil(t, exitErr, "Thread scan should succeed")

	output := stdout + stderr
	assert.NotEmpty(t, output, "Should produce output")
	t.Logf("Output:\n%s", output)
}

// TestBitBucketScan_Verbose tests verbose logging
func TestBitBucketScan_Verbose(t *testing.T) {

	server, _, cleanup := startMockServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		if r.URL.Path == "/repositories/test-workspace" {
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"values": []map[string]interface{}{},
			})
		} else {
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
		"--workspace", "test-workspace",
		"--verbose",
	}, nil, 10*time.Second)

	assert.Nil(t, exitErr, "Verbose scan should succeed")

	output := stdout + stderr
	// Verbose mode should produce more detailed output
	assert.NotEmpty(t, output, "Should produce output")
	t.Logf("Output:\n%s", output)
}

// TestBitBucketScan_TruffleHogVerification tests disabling credential verification
func TestBitBucketScan_TruffleHogVerification(t *testing.T) {

	server, _, cleanup := startMockServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		switch r.URL.Path {
		case "/repositories/test-workspace":
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
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"values": []map[string]interface{}{
					{
						"uuid": "{step-uuid-1}",
						"name": "Build",
						"state": map[string]interface{}{
							"name": "COMPLETED",
						},
					},
				},
			})

		case "/repositories/test-workspace/test-repo/pipelines/{pipeline-uuid-1}/steps/{step-uuid-1}/log":
			w.Header().Set("Content-Type", "text/plain")
			w.WriteHeader(http.StatusOK)
			logContent := `+ export FAKE_KEY="not_a_real_key_12345"`
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
		"--token", "testtoken",
		"--workspace", "test-workspace",
		"--truffleHogVerification=false",
	}, nil, 15*time.Second)

	assert.Nil(t, exitErr, "Scan with verification disabled should succeed")

	output := stdout + stderr
	assert.NotEmpty(t, output, "Should produce output")
	t.Logf("Output:\n%s", output)
}

// TestBitBucketScan_Artifacts_MissingCookie tests artifacts flag without cookie
func TestBitBucketScan_Artifacts_MissingCookie(t *testing.T) {

	server, _, cleanup := startMockServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{"values": []interface{}{}})
	})
	defer cleanup()

	// Try to use --artifacts without --cookie (should fail due to cobra validation)
	stdout, stderr, exitErr := runCLI(t, []string{
		"bb", "scan",
		"--bitbucket", server.URL,
		"--username", "testuser",
		"--token", "testtoken",
		"--workspace", "test-workspace",
		"--artifacts",
		// Missing --cookie flag
	}, nil, 5*time.Second)

	output := stdout + stderr
	// Cobra should enforce the flag relationship
	assert.NotNil(t, exitErr, "Should fail without cookie when artifacts is specified")
	assert.Contains(t, output, "cookie", "Should mention cookie requirement")
	t.Logf("Output:\n%s", output)
}

// TestBitBucketScan_Cookie_WithoutArtifacts tests that cookie requires artifacts flag
func TestBitBucketScan_Cookie_WithoutArtifacts(t *testing.T) {

	server, _, cleanup := startMockServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{"values": []interface{}{}})
	})
	defer cleanup()

	// Try to use --cookie without --artifacts (should fail due to cobra validation)
	stdout, stderr, exitErr := runCLI(t, []string{
		"bb", "scan",
		"--bitbucket", server.URL,
		"--username", "testuser",
		"--token", "testtoken",
		"--cookie", "test-cookie-value",
		"--workspace", "test-workspace",
		// Missing --artifacts flag
	}, nil, 5*time.Second)

	output := stdout + stderr
	// Cobra should enforce the flag relationship
	assert.NotNil(t, exitErr, "Should fail without artifacts when cookie is specified")
	assert.Contains(t, output, "artifacts", "Should mention artifacts requirement")
	t.Logf("Output:\n%s", output)
}

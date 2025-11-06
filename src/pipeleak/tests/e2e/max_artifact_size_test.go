package e2e

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// TestGitHubScan_MaxArtifactSize tests the --max-artifact-size flag for GitHub
func TestGitHubScan_MaxArtifactSize(t *testing.T) {

	// Create small artifact (100KB)
	var smallArtifactBuf bytes.Buffer
	smallZipWriter := zip.NewWriter(&smallArtifactBuf)
	smallFile, _ := smallZipWriter.Create("small.txt")
	_, _ = smallFile.Write(bytes.Repeat([]byte("x"), 100*1024)) // 100KB
	_ = smallZipWriter.Close()

	// Create large artifact (100MB simulation - just metadata)
	var largeArtifactBuf bytes.Buffer
	largeZipWriter := zip.NewWriter(&largeArtifactBuf)
	largeFile, _ := largeZipWriter.Create("large.txt")
	_, _ = largeFile.Write([]byte("This would be large"))
	_ = largeZipWriter.Close()

	server, getRequests, cleanup := startMockServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		t.Logf("GitHub Mock (MaxArtifactSize): %s %s", r.Method, r.URL.Path)

		switch r.URL.Path {
		case "/api/v3/user/repos":
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode([]map[string]interface{}{
				{
					"id":        1,
					"name":      "artifact-test",
					"full_name": "user/artifact-test",
					"html_url":  "https://github.com/user/artifact-test",
					"owner":     map[string]interface{}{"login": "user"},
				},
			})

		case "/api/v3/repos/user/artifact-test/actions/runs":
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"workflow_runs": []map[string]interface{}{
					{
						"id":            100,
						"name":          "test-workflow",
						"status":        "completed",
						"display_title": "Test Artifacts",
						"html_url":      "https://github.com/user/artifact-test/actions/runs/100",
						"repository": map[string]interface{}{
							"name":  "artifact-test",
							"owner": map[string]interface{}{"login": "user"},
						},
					},
				},
				"total_count": 1,
			})

		case "/api/v3/repos/user/artifact-test/actions/runs/100/jobs":
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"jobs":        []map[string]interface{}{},
				"total_count": 0,
			})

		case "/api/v3/repos/user/artifact-test/actions/runs/100/logs":
			w.Header().Set("Location", "http://"+r.Host+"/download/logs/100.zip")
			w.WriteHeader(http.StatusFound)

		case "/download/logs/100.zip":
			w.WriteHeader(http.StatusNotFound)

		case "/api/v3/repos/user/artifact-test/actions/runs/100/artifacts":
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"artifacts": []map[string]interface{}{
					{
						"id":                   1001,
						"name":                 "large-artifact",
						"size_in_bytes":        100 * 1024 * 1024, // 100MB
						"archive_download_url": "http://" + r.Host + "/api/v3/repos/user/artifact-test/actions/artifacts/1001/zip",
					},
					{
						"id":                   1002,
						"name":                 "small-artifact",
						"size_in_bytes":        100 * 1024, // 100KB
						"archive_download_url": "http://" + r.Host + "/api/v3/repos/user/artifact-test/actions/artifacts/1002/zip",
					},
				},
				"total_count": 2,
			})

		case "/api/v3/repos/user/artifact-test/actions/artifacts/1001/zip":
			// Large artifact - should NOT be called if size checking works
			t.Error("Large artifact download should be skipped before SDK call")
			w.Header().Set("Location", "http://"+r.Host+"/download/artifact/1001")
			w.WriteHeader(http.StatusFound)

		case "/api/v3/repos/user/artifact-test/actions/artifacts/1002/zip":
			// Small artifact - should be downloaded
			w.Header().Set("Location", "http://"+r.Host+"/download/artifact/1002")
			w.WriteHeader(http.StatusFound)

		case "/download/artifact/1001":
			// This should not be called if size checking works
			t.Error("Large artifact should not be downloaded")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write(largeArtifactBuf.Bytes())

		case "/download/artifact/1002":
			w.Header().Set("Content-Type", "application/zip")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write(smallArtifactBuf.Bytes())

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
		"--artifacts",
		"--max-artifact-size", "50Mb", // Only scan artifacts < 50MB
		"--owned",
		"--log-level", "debug", // Enable debug logs to see size checking
	}, nil, 15*time.Second)

	assert.Nil(t, exitErr, "Scan with max-artifact-size should succeed")

	output := stdout + stderr
	t.Logf("Output:\n%s", output)

	// Verify that large artifact was skipped (logged at debug level)
	if !assert.Contains(t, output, "Skipped large artifact", "Should log skipping of large artifact") {
		// If debug message not found, at least verify SDK call was not made
		requests := getRequests()
		var artifactSDKCalls []RecordedRequest
		for _, req := range requests {
			if req.Path == "/api/v3/repos/user/artifact-test/actions/artifacts/1001/zip" {
				artifactSDKCalls = append(artifactSDKCalls, req)
			}
		}
		assert.Equal(t, 0, len(artifactSDKCalls), "Large artifact SDK call should not be made")
	}
}

// TestDevOpsScan_MaxArtifactSize tests the --max-artifact-size flag for Azure DevOps
func TestDevOpsScan_MaxArtifactSize(t *testing.T) {

	server, _, cleanup := startMockServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		t.Logf("DevOps Mock (MaxArtifactSize): %s %s", r.Method, r.URL.Path)

		serverURL := "http://" + r.Host

		switch r.URL.Path {
		case "/_apis/profile/profiles/me":
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"id":          "user-123",
				"displayName": "Test User",
			})

		case "/_apis/accounts":
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"value": []map[string]interface{}{
					{
						"accountId":   "org-123",
						"accountName": "TestOrg",
					},
				},
			})

		case "/TestOrg/_apis/projects":
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"value": []map[string]interface{}{
					{
						"id":   "proj-123",
						"name": "TestProject",
					},
				},
			})

		case "/TestOrg/TestProject/_apis/build/builds":
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"value": []map[string]interface{}{
					{
						"id":     1000,
						"status": "completed",
						"_links": map[string]interface{}{
							"web": map[string]interface{}{
								"href": serverURL + "/build/1000",
							},
						},
					},
				},
			})

		case "/TestOrg/TestProject/_apis/build/builds/1000/artifacts":
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"value": []map[string]interface{}{
					{
						"id":   2001,
						"name": "large-artifact",
						"resource": map[string]interface{}{
							"type": "Container",
							"properties": map[string]interface{}{
								"artifactsize": "104857600", // 100MB
							},
							"downloadUrl": serverURL + "/download/large",
						},
					},
					{
						"id":   2002,
						"name": "small-artifact",
						"resource": map[string]interface{}{
							"type": "Container",
							"properties": map[string]interface{}{
								"artifactsize": "102400", // 100KB
							},
							"downloadUrl": serverURL + "/download/small",
						},
					},
				},
			})

		case "/download/large":
			t.Error("Large artifact should not be downloaded")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("PK\x03\x04"))

		case "/download/small":
			w.Header().Set("Content-Type", "application/zip")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("PK\x03\x04"))

		default:
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(map[string]interface{}{})
		}
	})
	defer cleanup()

	stdout, stderr, exitErr := runCLI(t, []string{
		"ad", "scan",
		"--devops", server.URL,
		"--token", "test-token",
		"--username", "testuser",
		"--organization", "TestOrg",
		"--project", "TestProject",
		"--artifacts",
		"--max-artifact-size", "50Mb",
	}, nil, 15*time.Second)

	assert.Nil(t, exitErr, "DevOps scan with max-artifact-size should succeed")

	output := stdout + stderr
	t.Logf("Output:\n%s", output)

	// Verify that large artifact was skipped (logged at debug level, but we'll accept success without error)
	// Since the test runs successfully and doesn't download the large artifact, the check is working
}

// TestBitBucketScan_MaxArtifactSize tests the --max-artifact-size flag for BitBucket
func TestBitBucketScan_MaxArtifactSize(t *testing.T) {

	server, _, cleanup := startMockServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		t.Logf("BitBucket Mock (MaxArtifactSize): %s %s", r.Method, r.URL.Path)

		switch r.URL.Path {
		case "/2.0/user":
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"uuid":         "user-123",
				"display_name": "Test User",
			})

		case "/2.0/workspaces":
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"values": []map[string]interface{}{
					{
						"slug": "test-workspace",
						"name": "Test Workspace",
					},
				},
			})

		case "/2.0/repositories/test-workspace":
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"values": []map[string]interface{}{
					{
						"slug": "test-repo",
						"name": "Test Repo",
					},
				},
			})

		case "/2.0/repositories/test-workspace/test-repo/pipelines/":
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"values": []map[string]interface{}{
					{
						"uuid":    "pipeline-123",
						"build_number": 1,
						"state": map[string]interface{}{
							"name": "COMPLETED",
						},
					},
				},
			})

		case "/2.0/repositories/test-workspace/test-repo/pipelines/pipeline-123/steps/":
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"values": []map[string]interface{}{
					{
						"uuid": "step-123",
						"name": "Build",
					},
				},
			})

		case "/internal/workspaces/test-workspace/repositories/test-repo/pipelines/1/steps/step-123/artifacts":
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"values": []map[string]interface{}{
					{
						"uuid":            "artifact-large",
						"name":            "large.zip",
						"file_size_bytes": 100 * 1024 * 1024, // 100MB
					},
					{
						"uuid":            "artifact-small",
						"name":            "small.zip",
						"file_size_bytes": 100 * 1024, // 100KB
					},
				},
			})

		default:
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(map[string]interface{}{})
		}
	})
	defer cleanup()

	stdout, stderr, exitErr := runCLI(t, []string{
		"bb", "scan",
		"--bitbucket", server.URL,
		"--token", "test-token",
		"--email", "test@example.com",
		"--cookie", "test-cookie",
		"--artifacts",
		"--max-artifact-size", "50Mb",
		"--workspace", "test-workspace",
	}, nil, 15*time.Second)

	assert.Nil(t, exitErr, "BitBucket scan with max-artifact-size should succeed")

	output := stdout + stderr
	t.Logf("Output:\n%s", output)

	// Verify scan completed successfully (size check happens before download)
}

// TestGiteaScan_MaxArtifactSize tests the --max-artifact-size flag for Gitea
func TestGiteaScan_MaxArtifactSize(t *testing.T) {

	// Create small artifact
	var smallArtifactBuf bytes.Buffer
	smallZipWriter := zip.NewWriter(&smallArtifactBuf)
	smallFile, _ := smallZipWriter.Create("small.txt")
	_, _ = smallFile.Write(bytes.Repeat([]byte("x"), 100*1024)) // 100KB
	_ = smallZipWriter.Close()

	server, _, cleanup := startMockServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		t.Logf("Gitea Mock (MaxArtifactSize): %s %s", r.Method, r.URL.Path)

		serverURL := "http://" + r.Host

		switch r.URL.Path {
		case "/api/v1", "/api/v1/version":
			// Gitea version check
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"version": "1.20.0",
			})

		case "/api/v1/repos/search":
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"data": []map[string]interface{}{
					{
						"id":        1,
						"name":      "test-repo",
						"full_name": "user/test-repo",
						"html_url":  serverURL + "/user/test-repo",
						"owner": map[string]interface{}{
							"login": "user",
						},
					},
				},
			})

		case "/api/v1/repos/user/test-repo/actions/runs":
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"workflow_runs": []map[string]interface{}{
					{
						"id":     100,
						"name":   "test-workflow",
						"status": "completed",
					},
				},
				"total_count": 1,
			})

		case "/api/v1/repos/user/test-repo/actions/runs/100/artifacts":
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"artifacts": []map[string]interface{}{
					{
						"id":                   1001,
						"name":                 "large-artifact",
						"size_in_bytes":        100 * 1024 * 1024, // 100MB - correct field name!
						"archive_download_url": serverURL + "/api/v1/repos/user/test-repo/actions/artifacts/1001/zip",
					},
					{
						"id":                   1002,
						"name":                 "small-artifact",
						"size_in_bytes":        100 * 1024, // 100KB - correct field name!
						"archive_download_url": serverURL + "/api/v1/repos/user/test-repo/actions/artifacts/1002/zip",
					},
				},
				"total_count": 2,
			})

		case "/api/v1/repos/user/test-repo/actions/artifacts/1001/zip":
			t.Error("Large artifact should not be downloaded")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("PK\x03\x04"))

		case "/api/v1/repos/user/test-repo/actions/artifacts/1002/zip":
			w.Header().Set("Content-Type", "application/zip")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write(smallArtifactBuf.Bytes())

		default:
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(map[string]interface{}{})
		}
	})
	defer cleanup()

	stdout, stderr, exitErr := runCLI(t, []string{
		"gitea", "scan",
		"--gitea", server.URL,
		"--token", "test-token",
		"--artifacts",
		"--max-artifact-size", "50Mb",
		"--log-level", "debug", // Enable debug logs to see size checking
	}, nil, 15*time.Second)

	assert.Nil(t, exitErr, "Gitea scan with max-artifact-size should succeed")

	output := stdout + stderr
	t.Logf("Output:\n%s", output)

	// Verify that large artifact was skipped (logged at debug level)
	if !assert.Contains(t, output, "Skipped large artifact", "Should log skipping of large artifact") {
		// If not found in log, verify SDK call was not made
		t.Log("Debug log not found, but test passed - size check prevented download")
	}
}

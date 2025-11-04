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


func TestBitBucketScan_Artifacts_WithDotEnv(t *testing.T) {

	server, getRequests, cleanup := startMockServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		t.Logf("BitBucket Mock: %s %s", r.Method, r.URL.Path)

		switch r.URL.Path {
		case "/!api/2.0/user":
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"username":     "testuser",
				"display_name": "Test User",
				"uuid":         "{user-uuid-1}",
			})

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
			_, _ = w.Write([]byte("+ echo 'Build completed'"))

		case "/!api/internal/repositories/test-workspace/test-repo/pipelines/1/artifacts":
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"values": []map[string]interface{}{
					{
						"uuid":            "{artifact-uuid-1}",
						"step_uuid":       "{step-uuid-1}",
						"name":            "environment.zip",
						"path":            "artifacts/environment.zip",
						"artifactType":    "file",
						"file_size_bytes": 2048,
						"created_on":      "2023-01-01T00:00:00.000000+00:00",
					},
				},
			})

		case "/!api/internal/repositories/test-workspace/test-repo/pipelines/1/artifacts/{artifact-uuid-1}/content":
			w.Header().Set("Content-Type", "application/zip")
			w.WriteHeader(http.StatusOK)

			// Create a zip with a .env file containing secrets
			var buf bytes.Buffer
			zw := zip.NewWriter(&buf)
			envFile, _ := zw.Create(".env")
			_, _ = envFile.Write([]byte(`# Environment configuration
DATABASE_URL=postgresql://admin:MySecretP@ssw0rd123@db.example.com:5432/proddb
AWS_ACCESS_KEY_ID=AKIAIOSFODNN7EXAMPLE
AWS_SECRET_ACCESS_KEY=wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY
STRIPE_SECRET_KEY=sk_live_51Hxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx
GITHUB_TOKEN=ghp_AbCdEfGhIjKlMnOpQrStUvWxYz1234567890
`))
			_ = zw.Close()
			_, _ = w.Write(buf.Bytes())

		case "/repositories/test-workspace/test-repo/downloads":
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(map[string]interface{}{"values": []interface{}{}})

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
		"--cookie", "test-cookie-value",
		"--workspace", "test-workspace",
		"--artifacts",
	}, nil, 20*time.Second)

	assert.Nil(t, exitErr, "Artifact scan with .env should succeed")

	requests := getRequests()
	assert.True(t, len(requests) >= 1, "Should make API requests")

	output := stdout + stderr

	// Verify .env file was scanned
	assert.Contains(t, output, ".env", "Should detect .env file in artifact")
	assert.Contains(t, output, "HIT Artifact", "Should detect secrets in .env file")

	// Verify various secret types were detected
	assert.Contains(t, output, "Password in URL", "Should detect password in database URL")
	assert.Contains(t, output, "Github", "Should detect GitHub token")

	t.Logf("STDOUT:\n%s", stdout)
	t.Logf("STDERR:\n%s", stderr)
}


func TestBitBucketScan_Artifacts_NestedArchive(t *testing.T) {

	server, getRequests, cleanup := startMockServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		t.Logf("BitBucket Mock: %s %s", r.Method, r.URL.Path)

		switch r.URL.Path {
		case "/!api/2.0/user":
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"username":     "testuser",
				"display_name": "Test User",
				"uuid":         "{user-uuid-1}",
			})

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
			_, _ = w.Write([]byte("+ echo 'Build completed'"))

		case "/!api/internal/repositories/test-workspace/test-repo/pipelines/1/artifacts":
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"values": []map[string]interface{}{
					{
						"uuid":            "{artifact-uuid-1}",
						"step_uuid":       "{step-uuid-1}",
						"name":            "nested.zip",
						"path":            "artifacts/nested.zip",
						"artifactType":    "file",
						"file_size_bytes": 3072,
						"created_on":      "2023-01-01T00:00:00.000000+00:00",
					},
				},
			})

		case "/!api/internal/repositories/test-workspace/test-repo/pipelines/1/artifacts/{artifact-uuid-1}/content":
			w.Header().Set("Content-Type", "application/zip")
			w.WriteHeader(http.StatusOK)

			// Create inner zip with secret
			var innerBuf bytes.Buffer
			innerZw := zip.NewWriter(&innerBuf)
			secretFile, _ := innerZw.Create("secret.txt")
			_, _ = secretFile.Write([]byte(`API_TOKEN=sk_test_1234567890abcdefghijklmnopqrstuvwxyzABCDEF
ADMIN_PASSWORD=SuperSecretAdminPass123!
`))
			_ = innerZw.Close()

			// Create outer zip containing inner zip
			var outerBuf bytes.Buffer
			outerZw := zip.NewWriter(&outerBuf)
			nestedZipFile, _ := outerZw.Create("inner.zip")
			_, _ = nestedZipFile.Write(innerBuf.Bytes())
			_ = outerZw.Close()

			_, _ = w.Write(outerBuf.Bytes())

		case "/repositories/test-workspace/test-repo/downloads":
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(map[string]interface{}{"values": []interface{}{}})

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
		"--cookie", "test-cookie-value",
		"--workspace", "test-workspace",
		"--artifacts",
	}, nil, 20*time.Second)

	assert.Nil(t, exitErr, "Nested archive scan should succeed")

	requests := getRequests()
	assert.True(t, len(requests) >= 1, "Should make API requests")

	output := stdout + stderr

	// Verify nested archive was processed
	assert.Contains(t, output, "HIT Artifact", "Should detect secrets in nested archive")
	// The scanner should find secrets in the inner archive
	assert.Contains(t, output, "secret.txt", "Should scan files in nested archive")

	t.Logf("STDOUT:\n%s", stdout)
	t.Logf("STDERR:\n%s", stderr)
}


func TestBitBucketScan_Artifacts_MultipleFiles(t *testing.T) {

	server, getRequests, cleanup := startMockServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		t.Logf("BitBucket Mock: %s %s", r.Method, r.URL.Path)

		switch r.URL.Path {
		case "/!api/2.0/user":
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"username":     "testuser",
				"display_name": "Test User",
				"uuid":         "{user-uuid-1}",
			})

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
			_, _ = w.Write([]byte("+ echo 'Build completed'"))

		case "/!api/internal/repositories/test-workspace/test-repo/pipelines/1/artifacts":
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"values": []map[string]interface{}{
					{
						"uuid":            "{artifact-uuid-1}",
						"step_uuid":       "{step-uuid-1}",
						"name":            "app-bundle.zip",
						"path":            "artifacts/app-bundle.zip",
						"artifactType":    "file",
						"file_size_bytes": 4096,
						"created_on":      "2023-01-01T00:00:00.000000+00:00",
					},
				},
			})

		case "/!api/internal/repositories/test-workspace/test-repo/pipelines/1/artifacts/{artifact-uuid-1}/content":
			w.Header().Set("Content-Type", "application/zip")
			w.WriteHeader(http.StatusOK)

			// Create zip with multiple files containing different secrets
			var buf bytes.Buffer
			zw := zip.NewWriter(&buf)

			// File 1: Database config
			dbConfig, _ := zw.Create("config/database.yml")
			_, _ = dbConfig.Write([]byte(`production:
  adapter: postgresql
  host: db.example.com
  database: myapp_prod
  username: dbadmin
  password: MyDBP@ssw0rd123!
  port: 5432
`))

			// File 2: API keys
			apiKeys, _ := zw.Create("config/api_keys.json")
			_, _ = apiKeys.Write([]byte(`{
  "stripe": "sk_live_51ABCDEFxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx",
  "sendgrid": "SG.1234567890abcdefghijklmnopqrstuvwxyz",
  "aws_access_key": "AKIAIOSFODNN7EXAMPLE"
}
`))

			// File 3: Environment variables
			envFile, _ := zw.Create(".env.production")
			_, _ = envFile.Write([]byte(`JWT_SECRET=supersecretjwtkey123456789
ENCRYPTION_KEY=abc123def456ghi789jkl012mno345pqr
`))

			_ = zw.Close()
			_, _ = w.Write(buf.Bytes())

		case "/repositories/test-workspace/test-repo/downloads":
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(map[string]interface{}{"values": []interface{}{}})

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
		"--cookie", "test-cookie-value",
		"--workspace", "test-workspace",
		"--artifacts",
	}, nil, 20*time.Second)

	assert.Nil(t, exitErr, "Multi-file artifact scan should succeed")

	requests := getRequests()
	assert.True(t, len(requests) >= 1, "Should make API requests")

	output := stdout + stderr

	// Verify multiple files were scanned
	assert.Contains(t, output, "HIT Artifact", "Should detect secrets across multiple files")

	// Check for secrets from different files that were actually scanned
	assert.Contains(t, output, "api_keys.json", "Should scan API keys file")
	assert.Contains(t, output, ".env.production", "Should scan env file")

	// Verify different secret types detected
	assert.Contains(t, output, "Stripe", "Should detect Stripe key")

	t.Logf("STDOUT:\n%s", stdout)
	t.Logf("STDERR:\n%s", stderr)
}


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


func TestBitBucketScan_DownloadArtifacts(t *testing.T) {

	server, getRequests, cleanup := startMockServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		t.Logf("BitBucket Mock: %s %s", r.Method, r.URL.Path)

		switch r.URL.Path {
		case "/!api/2.0/user":
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"username":     "testuser",
				"display_name": "Test User",
				"uuid":         "{user-uuid-1}",
			})

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
			_, _ = w.Write([]byte("+ echo 'Build completed'"))

		case "/!api/internal/repositories/test-workspace/test-repo/pipelines/1/artifacts":
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(map[string]interface{}{"values": []interface{}{}})

		case "/repositories/test-workspace/test-repo/downloads":
			// Return download artifacts pointing to a full URL with scheme
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"values": []map[string]interface{}{
					{
						"name": "release-v1.0.zip",
						"user": map[string]interface{}{
							"display_name": "Release Manager",
							"username":     "releaser",
						},
						"links": map[string]interface{}{
							"self": map[string]interface{}{
								"href": "http://" + r.Host + "/download-artifact-content",
							},
						},
						"created_on": "2023-01-01T00:00:00.000000+00:00",
					},
				},
			})

		case "/download-artifact-content":
			w.Header().Set("Content-Type", "application/zip")
			w.WriteHeader(http.StatusOK)

			// Create a zip with secrets
			var buf bytes.Buffer
			zw := zip.NewWriter(&buf)
			releaseConfig, _ := zw.Create("release-config.json")
			_, _ = releaseConfig.Write([]byte(`{
  "version": "1.0.0",
  "api_key": "sk_live_ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnop1234",
  "webhook_secret": "whsec_1234567890abcdefghijklmnopqrstuvwxyz"
}
`))
			_ = zw.Close()
			_, _ = w.Write(buf.Bytes())

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
		"--cookie", "test-cookie-value",
		"--workspace", "test-workspace",
		"--artifacts",
	}, nil, 20*time.Second)

	assert.Nil(t, exitErr, "Download artifacts scan should succeed")

	requests := getRequests()
	assert.True(t, len(requests) >= 1, "Should make API requests")

	output := stdout + stderr

	// Verify download artifact was processed
	assert.Contains(t, output, "HIT Artifact", "Should detect secrets in download artifact")
	assert.Contains(t, output, "release-config.json", "Should scan downloaded artifact file")

	t.Logf("STDOUT:\n%s", stdout)
	t.Logf("STDERR:\n%s", stderr)
}



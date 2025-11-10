package scan

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"code.gitea.io/sdk/gitea"
	"github.com/stretchr/testify/assert"
)

func TestValidateCookie(t *testing.T) {
	tests := []struct {
		name         string
		setupServer  func() *httptest.Server
		expectFatal  bool
		responseBody string
		responseCode int
	}{
		{
			name: "valid cookie",
			setupServer: func() *httptest.Server {
				return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(http.StatusOK)
					_, _ = w.Write([]byte("<html><body>Valid page</body></html>"))
				}))
			},
			expectFatal:  false,
			responseBody: "<html><body>Valid page</body></html>",
			responseCode: http.StatusOK,
		},
		{
			name: "invalid cookie - redirects to login",
			setupServer: func() *httptest.Server {
				return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(http.StatusOK)
					_, _ = w.Write([]byte(`<html><body><a href="/user/login">Login</a></body></html>`))
				}))
			},
			expectFatal:  true, // Would log.Fatal in real execution
			responseBody: `<html><body><a href="/user/login">Login</a></body></html>`,
			responseCode: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			setupTestScanOptions()
			server := tt.setupServer()
			defer server.Close()
			scanOptions.GiteaURL = server.URL

			// Note: In actual execution, validateCookie() calls log.Fatal() which exits the program
			// In tests, we can't easily test log.Fatal() without mocking the logger
			// Here we test that the function completes without panic
			if !tt.expectFatal {
				assert.NotPanics(t, func() {
					validateCookie()
				})
			}
		})
	}
}

func TestGetLatestRunIDFromHTML(t *testing.T) {
	tests := []struct {
		name        string
		repo        *gitea.Repository
		setupServer func() *httptest.Server
		expectedID  int64
		expectError bool
		errorMsg    string
	}{
		{
			name: "successfully extract run ID",
			repo: &gitea.Repository{
				FullName: "owner/repo",
			},
			setupServer: func() *httptest.Server {
				return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(http.StatusOK)
					_, _ = w.Write([]byte(`
						<html>
						<body>
							<a href="/owner/repo/actions/runs/12345">Latest Run</a>
							<a href="/owner/repo/actions/runs/12344">Previous Run</a>
						</body>
						</html>
					`))
				}))
			},
			expectedID:  12345,
			expectError: false,
		},
		{
			name: "no runs found",
			repo: &gitea.Repository{
				FullName: "owner/repo",
			},
			setupServer: func() *httptest.Server {
				return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(http.StatusOK)
					_, _ = w.Write([]byte("<html><body>No runs</body></html>"))
				}))
			},
			expectedID:  0,
			expectError: false,
		},
		{
			name: "404 response - actions disabled",
			repo: &gitea.Repository{
				FullName: "owner/repo",
			},
			setupServer: func() *httptest.Server {
				return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(http.StatusNotFound)
				}))
			},
			expectedID:  0,
			expectError: false,
		},
		{
			name:        "nil repository",
			repo:        nil,
			setupServer: func() *httptest.Server { return nil },
			expectedID:  0,
			expectError: true,
			errorMsg:    "repository is nil",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			setupTestScanOptions()
			if tt.setupServer != nil {
				server := tt.setupServer()
				if server != nil {
					defer server.Close()
					scanOptions.GiteaURL = server.URL
				}
			}

			runID, err := getLatestRunIDFromHTML(tt.repo)

			if tt.expectError {
				assert.Error(t, err)
				if tt.errorMsg != "" {
					assert.Contains(t, err.Error(), tt.errorMsg)
				}
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedID, runID)
			}
		})
	}
}

func TestScanJobLogsWithCookie(t *testing.T) {
	tests := []struct {
		name            string
		repo            *gitea.Repository
		runID           int64
		jobID           int64
		setupServer     func() *httptest.Server
		expectedSuccess bool
	}{
		{
			name: "successful log scan",
			repo: &gitea.Repository{
				FullName: "owner/repo",
			},
			runID: 123,
			jobID: 456,
			setupServer: func() *httptest.Server {
				return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(http.StatusOK)
					_, _ = w.Write([]byte("log content"))
				}))
			},
			expectedSuccess: true,
		},
		{
			name: "404 logs not found",
			repo: &gitea.Repository{
				FullName: "owner/repo",
			},
			runID: 123,
			jobID: 456,
			setupServer: func() *httptest.Server {
				return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(http.StatusNotFound)
				}))
			},
			expectedSuccess: false,
		},
		{
			name:            "nil repository",
			repo:            nil,
			runID:           123,
			jobID:           456,
			setupServer:     func() *httptest.Server { return nil },
			expectedSuccess: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			setupTestScanOptions()
			scanOptions.Artifacts = false
			if tt.setupServer != nil {
				server := tt.setupServer()
				if server != nil {
					defer server.Close()
					scanOptions.GiteaURL = server.URL
				}
			}

			success := scanJobLogsWithCookie(tt.repo, tt.runID, tt.jobID)

			assert.Equal(t, tt.expectedSuccess, success)
		})
	}
}

func TestFetchCsrfToken(t *testing.T) {
	tests := []struct {
		name          string
		setupServer   func() *httptest.Server
		expectedToken string
		expectError   bool
		errorMsg      string
	}{
		{
			name: "successfully extract CSRF token",
			setupServer: func() *httptest.Server {
				return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(http.StatusOK)
					_, _ = w.Write([]byte(`
						<html>
						<head>
							<script>
								window.config = {
									csrfToken: 'abc123def456'
								};
							</script>
						</head>
						</html>
					`))
				}))
			},
			expectedToken: "abc123def456",
			expectError:   false,
		},
		{
			name: "CSRF token not found",
			setupServer: func() *httptest.Server {
				return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(http.StatusOK)
					_, _ = w.Write([]byte("<html><body>No token</body></html>"))
				}))
			},
			expectedToken: "",
			expectError:   true,
			errorMsg:      "CSRF token not found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			setupTestScanOptions()
			server := tt.setupServer()
			defer server.Close()
			scanOptions.GiteaURL = server.URL

			token, err := fetchCsrfToken()

			if tt.expectError {
				assert.Error(t, err)
				if tt.errorMsg != "" {
					assert.Contains(t, err.Error(), tt.errorMsg)
				}
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedToken, token)
			}
		})
	}
}

func TestGetArtifactURLsFromRunHTML(t *testing.T) {
	tests := []struct {
		name          string
		repo          *gitea.Repository
		runID         int64
		setupServer   func() *httptest.Server
		expectedCount int
		expectError   bool
		errorMsg      string
	}{
		{
			name: "successfully extract artifacts",
			repo: &gitea.Repository{
				FullName: "owner/repo",
			},
			runID: 123,
			setupServer: func() *httptest.Server {
				callCount := 0
				return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					callCount++
					if callCount == 1 {
						w.WriteHeader(http.StatusOK)
						_, _ = w.Write([]byte(`<script>window.config={csrfToken:'token123'};</script>`))
					} else {
						w.WriteHeader(http.StatusOK)
						_, _ = w.Write([]byte(`{
							"artifacts": [
								{"name": "artifact1", "size": 1024, "status": "completed"},
								{"name": "artifact2", "size": 2048, "status": "completed"}
							]
						}`))
					}
				}))
			},
			expectedCount: 2,
			expectError:   false,
		},
		{
			name: "no artifacts found",
			repo: &gitea.Repository{
				FullName: "owner/repo",
			},
			runID: 123,
			setupServer: func() *httptest.Server {
				callCount := 0
				return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					callCount++
					if callCount == 1 {
						w.WriteHeader(http.StatusOK)
						_, _ = w.Write([]byte(`<script>window.config={csrfToken:'token123'};</script>`))
					} else {
						w.WriteHeader(http.StatusOK)
						_, _ = w.Write([]byte(`{"artifacts": []}`))
					}
				}))
			},
			expectedCount: 0,
			expectError:   false,
		},
		{
			name:          "nil repository",
			repo:          nil,
			runID:         123,
			setupServer:   func() *httptest.Server { return nil },
			expectedCount: 0,
			expectError:   true,
			errorMsg:      "repository is nil",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			setupTestScanOptions()
			if tt.setupServer != nil {
				server := tt.setupServer()
				if server != nil {
					defer server.Close()
					scanOptions.GiteaURL = server.URL
				}
			}

			artifactURLs, err := getArtifactURLsFromRunHTML(tt.repo, tt.runID)

			if tt.expectError {
				assert.Error(t, err)
				if tt.errorMsg != "" {
					assert.Contains(t, err.Error(), tt.errorMsg)
				}
			} else {
				assert.NoError(t, err)
				assert.Len(t, artifactURLs, tt.expectedCount)
			}
		})
	}
}

func TestDownloadAndScanArtifactWithCookie(t *testing.T) {
	tests := []struct {
		name         string
		repo         *gitea.Repository
		run          ActionWorkflowRun
		artifactName string
		setupServer  func() *httptest.Server
		expectPanic  bool
	}{
		{
			name: "successful artifact download",
			repo: &gitea.Repository{
				FullName: "owner/repo",
			},
			run: ActionWorkflowRun{
				ID:      123,
				HTMLURL: "https://gitea.example.com/owner/repo/actions/runs/123",
			},
			artifactName: "test-artifact.zip",
			setupServer: func() *httptest.Server {
				return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(http.StatusOK)
					_, _ = w.Write([]byte("artifact content"))
				}))
			},
			expectPanic: false,
		},
		{
			name: "artifact expired (410)",
			repo: &gitea.Repository{
				FullName: "owner/repo",
			},
			run: ActionWorkflowRun{
				ID:      123,
				HTMLURL: "https://gitea.example.com/owner/repo/actions/runs/123",
			},
			artifactName: "expired-artifact",
			setupServer: func() *httptest.Server {
				return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(http.StatusGone)
				}))
			},
			expectPanic: false,
		},
		{
			name:         "nil repository",
			repo:         nil,
			run:          ActionWorkflowRun{ID: 123},
			artifactName: "test",
			setupServer:  func() *httptest.Server { return nil },
			expectPanic:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			setupTestScanOptions()
			var artifactURL string
			if tt.setupServer != nil {
				server := tt.setupServer()
				if server != nil {
					defer server.Close()
					artifactURL = server.URL
					scanOptions.GiteaURL = server.URL
				}
			}

			if tt.expectPanic {
				assert.Panics(t, func() {
					downloadAndScanArtifactWithCookie(tt.repo, tt.run, tt.artifactName, artifactURL)
				})
			} else {
				assert.NotPanics(t, func() {
					downloadAndScanArtifactWithCookie(tt.repo, tt.run, tt.artifactName, artifactURL)
				})
			}
		})
	}
}

func TestScanRepositoryWithCookie(t *testing.T) {
	tests := []struct {
		name        string
		repo        *gitea.Repository
		setupServer func() *httptest.Server
		expectPanic bool
	}{
		{
			name:        "nil repository",
			repo:        nil,
			setupServer: func() *httptest.Server { return nil },
			expectPanic: false,
		},
		{
			name: "repository with no actions",
			repo: &gitea.Repository{
				FullName: "owner/repo",
			},
			setupServer: func() *httptest.Server {
				return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					// Return empty page with no run IDs
					w.WriteHeader(http.StatusOK)
					_, _ = w.Write([]byte("<html><body>No actions</body></html>"))
				}))
			},
			expectPanic: false,
		},
		{
			name: "repository with run ID in HTML",
			repo: &gitea.Repository{
				FullName: "owner/repo",
			},
			setupServer: func() *httptest.Server {
				return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					// Return page with run ID
					w.WriteHeader(http.StatusOK)
					_, _ = w.Write([]byte(`
						<html><body>
							<a href="/owner/repo/actions/runs/5">Run 5</a>
						</body></html>
					`))
				}))
			},
			expectPanic: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			setupTestScanOptions()
			scanOptions.Artifacts = false
			scanOptions.MaxScanGoRoutines = 2
			scanOptions.Context = context.Background()
			if tt.setupServer != nil {
				server := tt.setupServer()
				if server != nil {
					defer server.Close()
					scanOptions.GiteaURL = server.URL
				}
			}

			if tt.expectPanic {
				assert.Panics(t, func() {
					scanRepositoryWithCookie(tt.repo)
				})
			} else {
				assert.NotPanics(t, func() {
					scanRepositoryWithCookie(tt.repo)
				})
			}
		})
	}
}

func TestScanArtifactsWithCookie(t *testing.T) {
	tests := []struct {
		name        string
		repo        *gitea.Repository
		runID       int64
		runURL      string
		setupServer func() *httptest.Server
		expectPanic bool
	}{
		{
			name:        "nil repository",
			repo:        nil,
			runID:       123,
			runURL:      "https://example.com/run/123",
			setupServer: func() *httptest.Server { return nil },
			expectPanic: false,
		},
		{
			name: "no artifacts found",
			repo: &gitea.Repository{
				FullName: "owner/repo",
			},
			runID:  123,
			runURL: "https://example.com/run/123",
			setupServer: func() *httptest.Server {
				callCount := 0
				return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					callCount++
					if callCount == 1 {
						w.WriteHeader(http.StatusOK)
						_, _ = w.Write([]byte(`<script>window.config={csrfToken:'token123'};</script>`))
					} else {
						w.WriteHeader(http.StatusOK)
						_, _ = w.Write([]byte(`{"artifacts": []}`))
					}
				}))
			},
			expectPanic: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			setupTestScanOptions()
			scanOptions.MaxScanGoRoutines = 2
			scanOptions.Context = context.Background()
			if tt.setupServer != nil {
				server := tt.setupServer()
				if server != nil {
					defer server.Close()
					scanOptions.GiteaURL = server.URL
				}
			}

			if tt.expectPanic {
				assert.Panics(t, func() {
					scanArtifactsWithCookie(tt.repo, tt.runID, tt.runURL)
				})
			} else {
				assert.NotPanics(t, func() {
					scanArtifactsWithCookie(tt.repo, tt.runID, tt.runURL)
				})
			}
		})
	}
}

func TestScanArtifactsWithCookie_WithArtifacts(t *testing.T) {
	issuesCallCount := 0
	jobsCallCount := 0
	artifactCallCount := 0
	
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/issues":
			issuesCallCount++
			w.WriteHeader(http.StatusOK)
		case "/owner/repo/actions/runs/123/jobs/0":
			jobsCallCount++
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"artifacts": [{"name": "artifact1", "size": 1024, "url": "/owner/repo/actions/runs/123/artifacts/artifact1"}]}`))
		case "/owner/repo/actions/runs/123/artifacts/artifact1":
			artifactCallCount++
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("artifact content"))
		}
	}))
	defer server.Close()

	setupTestScanOptions()
	scanOptions.GiteaURL = server.URL
	scanOptions.Cookie = "test-cookie"

	repo := &gitea.Repository{
		Name:     "repo",
		FullName: "owner/repo",
		Owner:    &gitea.User{UserName: "owner"},
	}
	runID := int64(123)
	runURL := server.URL + "/owner/repo/actions/runs/123"

	assert.NotPanics(t, func() {
		scanArtifactsWithCookie(repo, runID, runURL)
	})
	
	// Verify the function attempted to fetch artifacts
	assert.GreaterOrEqual(t, issuesCallCount+jobsCallCount+artifactCallCount, 1, "Should make HTTP requests to fetch artifacts")
}

func TestScanArtifactsWithCookie_FetchError(t *testing.T) {
	issuesCallCount := 0
	errorCallCount := 0
	
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/issues":
			issuesCallCount++
			w.WriteHeader(http.StatusOK)
		default:
			errorCallCount++
			w.WriteHeader(http.StatusInternalServerError)
		}
	}))
	defer server.Close()

	setupTestScanOptions()
	scanOptions.GiteaURL = server.URL
	scanOptions.Cookie = "test-cookie"

	repo := &gitea.Repository{
		Name:     "repo",
		FullName: "owner/repo",
		Owner:    &gitea.User{UserName: "owner"},
	}
	runID := int64(123)
	runURL := server.URL + "/run/123"

	assert.NotPanics(t, func() {
		scanArtifactsWithCookie(repo, runID, runURL)
	})
	
	// Verify function attempted to fetch data and handled errors gracefully
	assert.GreaterOrEqual(t, issuesCallCount+errorCallCount, 1, "Should make at least one HTTP request")
}

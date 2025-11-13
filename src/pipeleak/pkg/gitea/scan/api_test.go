package gitea

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"code.gitea.io/sdk/gitea"
	"github.com/stretchr/testify/assert"
)

func TestListWorkflowRuns(t *testing.T) {
	tests := []struct {
		name           string
		repo           *gitea.Repository
		setupServer    func() *httptest.Server
		expectedRuns   int
		expectError    bool
		errorMsg       string
		setupRunsLimit int
	}{
		{
			name: "successful fetch with runs response",
			repo: &gitea.Repository{
				Name:     "test-repo",
				FullName: "owner/test-repo",
				Owner: &gitea.User{
					UserName: "owner",
				},
			},
			setupServer: func() *httptest.Server {
				return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					resp := ActionWorkflowRunsResponse{
						TotalCount: 2,
						WorkflowRuns: []ActionWorkflowRun{
							{ID: 1, Name: "Run 1", Status: "completed"},
							{ID: 2, Name: "Run 2", Status: "success"},
						},
					}
					w.WriteHeader(http.StatusOK)
					_ = json.NewEncoder(w).Encode(resp)
				}))
			},
			expectedRuns:   2,
			expectError:    false,
			setupRunsLimit: 0,
		},
		{
			name: "successful fetch with runs array (no wrapper)",
			repo: &gitea.Repository{
				Name:     "test-repo",
				FullName: "owner/test-repo",
				Owner: &gitea.User{
					UserName: "owner",
				},
			},
			setupServer: func() *httptest.Server {
				return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					runs := []ActionWorkflowRun{
						{ID: 1, Name: "Run 1"},
						{ID: 2, Name: "Run 2"},
						{ID: 3, Name: "Run 3"},
					}
					w.WriteHeader(http.StatusOK)
					_ = json.NewEncoder(w).Encode(runs)
				}))
			},
			expectedRuns:   3,
			expectError:    false,
			setupRunsLimit: 0,
		},
		{
			name: "404 response - no runs",
			repo: &gitea.Repository{
				Name:     "test-repo",
				FullName: "owner/test-repo",
				Owner: &gitea.User{
					UserName: "owner",
				},
			},
			setupServer: func() *httptest.Server {
				return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(http.StatusNotFound)
				}))
			},
			expectedRuns: 0,
			expectError:  false,
		},
		{
			name: "403 forbidden",
			repo: &gitea.Repository{
				Name:     "test-repo",
				FullName: "owner/test-repo",
				Owner: &gitea.User{
					UserName: "owner",
				},
			},
			setupServer: func() *httptest.Server {
				return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(http.StatusForbidden)
				}))
			},
			expectedRuns: 0,
			expectError:  true,
			errorMsg:     "403",
		},
		{
			name:        "nil repository",
			repo:        nil,
			setupServer: func() *httptest.Server { return nil },
			expectError: true,
			errorMsg:    "repository is nil",
		},
		{
			name: "nil repository owner",
			repo: &gitea.Repository{
				Name:  "test-repo",
				Owner: nil,
			},
			setupServer: func() *httptest.Server { return nil },
			expectError: true,
			errorMsg:    "repository owner is nil",
		},
		{
			name: "runs limit enforced",
			repo: &gitea.Repository{
				Name:     "test-repo",
				FullName: "owner/test-repo",
				Owner: &gitea.User{
					UserName: "owner",
				},
			},
			setupServer: func() *httptest.Server {
				return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					resp := ActionWorkflowRunsResponse{
						TotalCount: 10,
						WorkflowRuns: []ActionWorkflowRun{
							{ID: 1, Name: "Run 1"},
							{ID: 2, Name: "Run 2"},
							{ID: 3, Name: "Run 3"},
							{ID: 4, Name: "Run 4"},
							{ID: 5, Name: "Run 5"},
						},
					}
					w.WriteHeader(http.StatusOK)
					_ = json.NewEncoder(w).Encode(resp)
				}))
			},
			expectedRuns:   2,
			expectError:    false,
			setupRunsLimit: 2,
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
			scanOptions.RunsLimit = tt.setupRunsLimit

			runs, err := ListWorkflowRuns(nil, tt.repo)

			if tt.expectError {
				assert.Error(t, err)
				if tt.errorMsg != "" {
					assert.Contains(t, err.Error(), tt.errorMsg)
				}
			} else {
				assert.NoError(t, err)
				assert.Len(t, runs, tt.expectedRuns)
			}
		})
	}
}

func TestListWorkflowJobs(t *testing.T) {
	tests := []struct {
		name         string
		repo         *gitea.Repository
		run          ActionWorkflowRun
		setupServer  func() *httptest.Server
		expectedJobs int
		expectError  bool
		errorMsg     string
	}{
		{
			name: "successful fetch with jobs response",
			repo: &gitea.Repository{
				Name:     "test-repo",
				FullName: "owner/test-repo",
				Owner: &gitea.User{
					UserName: "owner",
				},
			},
			run: ActionWorkflowRun{ID: 123, Name: "Test Run"},
			setupServer: func() *httptest.Server {
				return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					resp := ActionJobsResponse{
						TotalCount: 3,
						Jobs: []ActionJob{
							{ID: 1, Name: "Job 1", Status: "completed"},
							{ID: 2, Name: "Job 2", Status: "success"},
							{ID: 3, Name: "Job 3", Status: "running"},
						},
					}
					w.WriteHeader(http.StatusOK)
					_ = json.NewEncoder(w).Encode(resp)
				}))
			},
			expectedJobs: 3,
			expectError:  false,
		},
		{
			name: "successful fetch with jobs array",
			repo: &gitea.Repository{
				Name:     "test-repo",
				FullName: "owner/test-repo",
				Owner: &gitea.User{
					UserName: "owner",
				},
			},
			run: ActionWorkflowRun{ID: 456, Name: "Another Run"},
			setupServer: func() *httptest.Server {
				return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					jobs := []ActionJob{
						{ID: 10, Name: "Job A"},
						{ID: 20, Name: "Job B"},
					}
					w.WriteHeader(http.StatusOK)
					_ = json.NewEncoder(w).Encode(jobs)
				}))
			},
			expectedJobs: 2,
			expectError:  false,
		},
		{
			name: "404 response - no jobs",
			repo: &gitea.Repository{
				Name:     "test-repo",
				FullName: "owner/test-repo",
				Owner: &gitea.User{
					UserName: "owner",
				},
			},
			run: ActionWorkflowRun{ID: 789, Name: "Empty Run"},
			setupServer: func() *httptest.Server {
				return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(http.StatusNotFound)
				}))
			},
			expectedJobs: 0,
			expectError:  false,
		},
		{
			name:        "nil repository",
			repo:        nil,
			run:         ActionWorkflowRun{ID: 123},
			setupServer: func() *httptest.Server { return nil },
			expectError: true,
			errorMsg:    "repository is nil",
		},
		{
			name: "nil repository owner",
			repo: &gitea.Repository{
				Name:  "test-repo",
				Owner: nil,
			},
			run:         ActionWorkflowRun{ID: 123},
			setupServer: func() *httptest.Server { return nil },
			expectError: true,
			errorMsg:    "repository owner is nil",
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

			jobs, err := listWorkflowJobs(nil, tt.repo, tt.run)

			if tt.expectError {
				assert.Error(t, err)
				if tt.errorMsg != "" {
					assert.Contains(t, err.Error(), tt.errorMsg)
				}
			} else {
				assert.NoError(t, err)
				assert.Len(t, jobs, tt.expectedJobs)
			}
		})
	}
}

func TestListArtifacts(t *testing.T) {
	tests := []struct {
		name              string
		repo              *gitea.Repository
		run               ActionWorkflowRun
		setupServer       func() *httptest.Server
		expectedArtifacts int
		expectError       bool
		errorMsg          string
	}{
		{
			name: "successful fetch with artifacts response",
			repo: &gitea.Repository{
				Name:     "test-repo",
				FullName: "owner/test-repo",
				Owner: &gitea.User{
					UserName: "owner",
				},
			},
			run: ActionWorkflowRun{ID: 123},
			setupServer: func() *httptest.Server {
				return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					resp := ActionArtifactsResponse{
						TotalCount: 2,
						Artifacts: []ActionArtifact{
							{ID: 1, Name: "artifact1", Size: 1024},
							{ID: 2, Name: "artifact2", Size: 2048},
						},
					}
					w.WriteHeader(http.StatusOK)
					_ = json.NewEncoder(w).Encode(resp)
				}))
			},
			expectedArtifacts: 2,
			expectError:       false,
		},
		{
			name: "successful fetch with artifacts array",
			repo: &gitea.Repository{
				Name:     "test-repo",
				FullName: "owner/test-repo",
				Owner: &gitea.User{
					UserName: "owner",
				},
			},
			run: ActionWorkflowRun{ID: 456},
			setupServer: func() *httptest.Server {
				return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					artifacts := []ActionArtifact{
						{ID: 10, Name: "log.zip"},
					}
					w.WriteHeader(http.StatusOK)
					_ = json.NewEncoder(w).Encode(artifacts)
				}))
			},
			expectedArtifacts: 1,
			expectError:       false,
		},
		{
			name: "404 response - no artifacts",
			repo: &gitea.Repository{
				Name:     "test-repo",
				FullName: "owner/test-repo",
				Owner: &gitea.User{
					UserName: "owner",
				},
			},
			run: ActionWorkflowRun{ID: 789},
			setupServer: func() *httptest.Server {
				return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(http.StatusNotFound)
				}))
			},
			expectedArtifacts: 0,
			expectError:       false,
		},
		{
			name:        "nil repository",
			repo:        nil,
			run:         ActionWorkflowRun{ID: 123},
			setupServer: func() *httptest.Server { return nil },
			expectError: true,
			errorMsg:    "repository is nil",
		},
		{
			name: "nil repository owner",
			repo: &gitea.Repository{
				Name:  "test-repo",
				Owner: nil,
			},
			run:         ActionWorkflowRun{ID: 123},
			setupServer: func() *httptest.Server { return nil },
			expectError: true,
			errorMsg:    "repository owner is nil",
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

			artifacts, err := listArtifacts(tt.repo, tt.run)

			if tt.expectError {
				assert.Error(t, err)
				if tt.errorMsg != "" {
					assert.Contains(t, err.Error(), tt.errorMsg)
				}
			} else {
				assert.NoError(t, err)
				assert.Len(t, artifacts, tt.expectedArtifacts)
			}
		})
	}
}

func TestScanJobLogs(t *testing.T) {
	tests := []struct {
		name        string
		repo        *gitea.Repository
		run         ActionWorkflowRun
		job         ActionJob
		setupServer func() *httptest.Server
		expectPanic bool
	}{
		{
			name: "successful log scan",
			repo: &gitea.Repository{
				Name:     "test-repo",
				FullName: "owner/test-repo",
				Owner: &gitea.User{
					UserName: "owner",
				},
			},
			run: ActionWorkflowRun{ID: 123, HTMLURL: "https://gitea.example.com/owner/test-repo/actions/runs/123"},
			job: ActionJob{ID: 456, Name: "test-job"},
			setupServer: func() *httptest.Server {
				return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(http.StatusOK)
					_, _ = w.Write([]byte("test log content"))
				}))
			},
			expectPanic: false,
		},
		{
			name:        "nil repository",
			repo:        nil,
			run:         ActionWorkflowRun{ID: 123},
			job:         ActionJob{ID: 456},
			setupServer: func() *httptest.Server { return nil },
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

			if tt.expectPanic {
				assert.Panics(t, func() {
					scanJobLogs(nil, tt.repo, tt.run, tt.job)
				})
			} else {
				assert.NotPanics(t, func() {
					scanJobLogs(nil, tt.repo, tt.run, tt.job)
				})
			}
		})
	}
}

func TestScanWorkflowRunLogs(t *testing.T) {
	tests := []struct {
		name        string
		repo        *gitea.Repository
		run         ActionWorkflowRun
		setupServer func() *httptest.Server
		expectPanic bool
	}{
		{
			name:        "nil repository",
			repo:        nil,
			run:         ActionWorkflowRun{ID: 123},
			setupServer: func() *httptest.Server { return nil },
			expectPanic: false,
		},
		{
			name: "successful workflow scan with jobs",
			repo: &gitea.Repository{
				Name:     "test-repo",
				FullName: "owner/test-repo",
				Owner: &gitea.User{
					UserName: "owner",
				},
			},
			run: ActionWorkflowRun{ID: 123, HTMLURL: "https://example.com/run/123"},
			setupServer: func() *httptest.Server {
				callCount := 0
				return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					callCount++
					if callCount == 1 {
						// First call for jobs list
						resp := ActionJobsResponse{
							TotalCount: 1,
							Jobs: []ActionJob{
								{ID: 1, Name: "Job 1"},
							},
						}
						w.WriteHeader(http.StatusOK)
						_ = json.NewEncoder(w).Encode(resp)
					} else {
						// Subsequent calls for logs
						w.WriteHeader(http.StatusOK)
						_, _ = w.Write([]byte("log content"))
					}
				}))
			},
			expectPanic: false,
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

			if tt.expectPanic {
				assert.Panics(t, func() {
					scanWorkflowRunLogs(nil, tt.repo, tt.run)
				})
			} else {
				assert.NotPanics(t, func() {
					scanWorkflowRunLogs(nil, tt.repo, tt.run)
				})
			}
		})
	}
}

func TestDownloadAndScanArtifact(t *testing.T) {
	tests := []struct {
		name        string
		repo        *gitea.Repository
		run         ActionWorkflowRun
		artifact    ActionArtifact
		setupServer func() *httptest.Server
		expectPanic bool
	}{
		{
			name:        "nil repository",
			repo:        nil,
			run:         ActionWorkflowRun{ID: 123},
			artifact:    ActionArtifact{ID: 789, Name: "test-artifact"},
			setupServer: func() *httptest.Server { return nil },
			expectPanic: false,
		},
		{
			name: "artifact expired (410)",
			repo: &gitea.Repository{
				Name:     "test-repo",
				FullName: "owner/test-repo",
				Owner: &gitea.User{
					UserName: "owner",
				},
			},
			run:      ActionWorkflowRun{ID: 123, HTMLURL: "https://example.com/run/123"},
			artifact: ActionArtifact{ID: 789, Name: "expired-artifact"},
			setupServer: func() *httptest.Server {
				return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(http.StatusGone)
				}))
			},
			expectPanic: false,
		},
		{
			name: "artifact not found (404)",
			repo: &gitea.Repository{
				Name:     "test-repo",
				FullName: "owner/test-repo",
				Owner: &gitea.User{
					UserName: "owner",
				},
			},
			run:      ActionWorkflowRun{ID: 123, HTMLURL: "https://example.com/run/123"},
			artifact: ActionArtifact{ID: 789, Name: "missing-artifact"},
			setupServer: func() *httptest.Server {
				return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(http.StatusNotFound)
				}))
			},
			expectPanic: false,
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

			if tt.expectPanic {
				assert.Panics(t, func() {
					downloadAndScanArtifact(tt.repo, tt.run, tt.artifact)
				})
			} else {
				assert.NotPanics(t, func() {
					downloadAndScanArtifact(tt.repo, tt.run, tt.artifact)
				})
			}
		})
	}
}

// Pagination Tests

func TestListWorkflowRuns_Pagination(t *testing.T) {
	tests := []struct {
		name         string
		totalRuns    int
		pageSize     int
		expectedRuns int
		description  string
	}{
		{
			name:         "single page - less than limit",
			totalRuns:    30,
			pageSize:     50,
			expectedRuns: 30,
			description:  "When total runs < page size, should return all in one request",
		},
		{
			name:         "exact page boundary",
			totalRuns:    100,
			pageSize:     50,
			expectedRuns: 100,
			description:  "When total runs is exact multiple of page size",
		},
		{
			name:         "multiple pages",
			totalRuns:    125,
			pageSize:     50,
			expectedRuns: 125,
			description:  "Should paginate through 3 pages (50+50+25)",
		},
		{
			name:         "large dataset",
			totalRuns:    500,
			pageSize:     50,
			expectedRuns: 500,
			description:  "Should handle 10 pages of data",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			requestCount := 0
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				requestCount++

				// Parse pagination parameters
				page, _ := strconv.Atoi(r.URL.Query().Get("page"))
				limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))

				if page == 0 {
					page = 1
				}
				if limit == 0 {
					limit = 50
				}

				// Calculate what runs to return
				startIdx := (page - 1) * limit
				endIdx := startIdx + limit
				if endIdx > tt.totalRuns {
					endIdx = tt.totalRuns
				}

				if startIdx >= tt.totalRuns {
					// No more data
					resp := ActionWorkflowRunsResponse{
						TotalCount:   int64(tt.totalRuns),
						WorkflowRuns: []ActionWorkflowRun{},
					}
					w.WriteHeader(http.StatusOK)
					_ = json.NewEncoder(w).Encode(resp)
					return
				}

				// Generate runs for this page
				runs := make([]ActionWorkflowRun, endIdx-startIdx)
				for i := range runs {
					runs[i] = ActionWorkflowRun{
						ID:   int64(startIdx + i + 1),
						Name: fmt.Sprintf("Run %d", startIdx+i+1),
					}
				}

				resp := ActionWorkflowRunsResponse{
					TotalCount:   int64(tt.totalRuns),
					WorkflowRuns: runs,
				}
				w.WriteHeader(http.StatusOK)
				_ = json.NewEncoder(w).Encode(resp)
			}))
			defer server.Close()

			setupTestScanOptions()
			scanOptions.GiteaURL = server.URL
			scanOptions.RunsLimit = 0 // No limit

			repo := &gitea.Repository{
				Name:     "test-repo",
				FullName: "owner/test-repo",
				Owner:    &gitea.User{UserName: "owner"},
			}

			runs, err := ListWorkflowRuns(nil, repo)

			assert.NoError(t, err)
			assert.Len(t, runs, tt.expectedRuns, tt.description)

			// Verify correct number of API requests
			expectedRequests := (tt.totalRuns + tt.pageSize - 1) / tt.pageSize
			assert.Equal(t, expectedRequests, requestCount, "Should make correct number of paginated requests")

			// Verify IDs are sequential
			if len(runs) > 0 {
				assert.Equal(t, int64(1), runs[0].ID, "First run should have ID 1")
				assert.Equal(t, int64(tt.expectedRuns), runs[len(runs)-1].ID, "Last run should have correct ID")
			}
		})
	}
}

func TestListWorkflowJobs_Pagination(t *testing.T) {
	tests := []struct {
		name         string
		totalJobs    int
		expectedJobs int
		description  string
	}{
		{
			name:         "single page",
			totalJobs:    25,
			expectedJobs: 25,
			description:  "Should fetch all jobs in single request",
		},
		{
			name:         "two pages",
			totalJobs:    75,
			expectedJobs: 75,
			description:  "Should paginate across two pages (50+25)",
		},
		{
			name:         "multiple pages",
			totalJobs:    150,
			expectedJobs: 150,
			description:  "Should handle 3 pages (50+50+50)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pageRequests := 0
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				pageRequests++

				page, _ := strconv.Atoi(r.URL.Query().Get("page"))
				limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))

				if page == 0 {
					page = 1
				}
				if limit == 0 {
					limit = 50
				}

				startIdx := (page - 1) * limit
				endIdx := startIdx + limit
				if endIdx > tt.totalJobs {
					endIdx = tt.totalJobs
				}

				if startIdx >= tt.totalJobs {
					resp := ActionJobsResponse{
						TotalCount: int64(tt.totalJobs),
						Jobs:       []ActionJob{},
					}
					w.WriteHeader(http.StatusOK)
					_ = json.NewEncoder(w).Encode(resp)
					return
				}

				jobs := make([]ActionJob, endIdx-startIdx)
				for i := range jobs {
					jobs[i] = ActionJob{
						ID:     int64(startIdx + i + 1),
						Name:   fmt.Sprintf("Job %d", startIdx+i+1),
						Status: "completed",
					}
				}

				resp := ActionJobsResponse{
					TotalCount: int64(tt.totalJobs),
					Jobs:       jobs,
				}
				w.WriteHeader(http.StatusOK)
				_ = json.NewEncoder(w).Encode(resp)
			}))
			defer server.Close()

			setupTestScanOptions()
			scanOptions.GiteaURL = server.URL

			repo := &gitea.Repository{
				Name:     "test-repo",
				FullName: "owner/test-repo",
				Owner:    &gitea.User{UserName: "owner"},
			}
			run := ActionWorkflowRun{ID: 123}

			jobs, err := listWorkflowJobs(nil, repo, run)

			assert.NoError(t, err)
			assert.Len(t, jobs, tt.expectedJobs, tt.description)
			assert.Greater(t, pageRequests, 0, "Should make at least one request")
		})
	}
}

func TestListArtifacts_Pagination(t *testing.T) {
	tests := []struct {
		name              string
		totalArtifacts    int
		expectedArtifacts int
		description       string
	}{
		{
			name:              "no artifacts",
			totalArtifacts:    0,
			expectedArtifacts: 0,
			description:       "Should handle empty result set",
		},
		{
			name:              "single page",
			totalArtifacts:    10,
			expectedArtifacts: 10,
			description:       "Should return all artifacts in one page",
		},
		{
			name:              "exact page boundary",
			totalArtifacts:    50,
			expectedArtifacts: 50,
			description:       "Should handle exact page size",
		},
		{
			name:              "multiple pages",
			totalArtifacts:    120,
			expectedArtifacts: 120,
			description:       "Should paginate through 3 pages (50+50+20)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				page, _ := strconv.Atoi(r.URL.Query().Get("page"))
				limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))

				if page == 0 {
					page = 1
				}
				if limit == 0 {
					limit = 50
				}

				startIdx := (page - 1) * limit
				endIdx := startIdx + limit
				if endIdx > tt.totalArtifacts {
					endIdx = tt.totalArtifacts
				}

				if startIdx >= tt.totalArtifacts {
					resp := ActionArtifactsResponse{
						TotalCount: int64(tt.totalArtifacts),
						Artifacts:  []ActionArtifact{},
					}
					w.WriteHeader(http.StatusOK)
					_ = json.NewEncoder(w).Encode(resp)
					return
				}

				artifacts := make([]ActionArtifact, endIdx-startIdx)
				for i := range artifacts {
					artifacts[i] = ActionArtifact{
						ID:   int64(startIdx + i + 1),
						Name: fmt.Sprintf("artifact-%d.zip", startIdx+i+1),
						Size: 1024,
					}
				}

				resp := ActionArtifactsResponse{
					TotalCount: int64(tt.totalArtifacts),
					Artifacts:  artifacts,
				}
				w.WriteHeader(http.StatusOK)
				_ = json.NewEncoder(w).Encode(resp)
			}))
			defer server.Close()

			setupTestScanOptions()
			scanOptions.GiteaURL = server.URL

			repo := &gitea.Repository{
				Name:     "test-repo",
				FullName: "owner/test-repo",
				Owner:    &gitea.User{UserName: "owner"},
			}
			run := ActionWorkflowRun{ID: 123}

			artifacts, err := listArtifacts(repo, run)

			assert.NoError(t, err)
			assert.Len(t, artifacts, tt.expectedArtifacts, tt.description)

			// Verify sequential IDs if artifacts exist
			if len(artifacts) > 0 {
				assert.Equal(t, int64(1), artifacts[0].ID)
				assert.Equal(t, int64(tt.totalArtifacts), artifacts[len(artifacts)-1].ID)
			}
		})
	}
}

func TestListWorkflowRuns_PaginationWithRunsLimit(t *testing.T) {
	tests := []struct {
		name         string
		totalRuns    int
		runsLimit    int
		expectedRuns int
		description  string
	}{
		{
			name:         "limit less than page size",
			totalRuns:    100,
			runsLimit:    20,
			expectedRuns: 20,
			description:  "Should stop after reaching runs limit",
		},
		{
			name:         "limit equals page size",
			totalRuns:    100,
			runsLimit:    50,
			expectedRuns: 50,
			description:  "Should return exactly one page",
		},
		{
			name:         "limit spans multiple pages",
			totalRuns:    200,
			runsLimit:    75,
			expectedRuns: 75,
			description:  "Should fetch partial second page and stop",
		},
		{
			name:         "limit greater than total",
			totalRuns:    30,
			runsLimit:    100,
			expectedRuns: 30,
			description:  "Should return all available runs when limit exceeds total",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				page, _ := strconv.Atoi(r.URL.Query().Get("page"))
				limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))

				if page == 0 {
					page = 1
				}
				if limit == 0 {
					limit = 50
				}

				startIdx := (page - 1) * limit
				endIdx := startIdx + limit
				if endIdx > tt.totalRuns {
					endIdx = tt.totalRuns
				}

				runs := make([]ActionWorkflowRun, endIdx-startIdx)
				for i := range runs {
					runs[i] = ActionWorkflowRun{
						ID:   int64(startIdx + i + 1),
						Name: fmt.Sprintf("Run %d", startIdx+i+1),
					}
				}

				resp := ActionWorkflowRunsResponse{
					TotalCount:   int64(tt.totalRuns),
					WorkflowRuns: runs,
				}
				w.WriteHeader(http.StatusOK)
				_ = json.NewEncoder(w).Encode(resp)
			}))
			defer server.Close()

			setupTestScanOptions()
			scanOptions.GiteaURL = server.URL
			scanOptions.RunsLimit = tt.runsLimit

			repo := &gitea.Repository{
				Name:     "test-repo",
				FullName: "owner/test-repo",
				Owner:    &gitea.User{UserName: "owner"},
			}

			runs, err := ListWorkflowRuns(nil, repo)

			assert.NoError(t, err)
			assert.Len(t, runs, tt.expectedRuns, tt.description)
		})
	}
}

func TestListWorkflowRuns_PaginationArrayFormat(t *testing.T) {
	// Test pagination when API returns array format (no wrapper)
	requestCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++
		page, _ := strconv.Atoi(r.URL.Query().Get("page"))

		if page == 0 {
			page = 1
		}

		// Return fewer items on later pages
		itemsToReturn := 50
		if page > 2 {
			itemsToReturn = 10 // Last page with fewer items
		}
		if page > 3 {
			itemsToReturn = 0 // No more data
		}

		runs := make([]ActionWorkflowRun, itemsToReturn)
		for i := range runs {
			runs[i] = ActionWorkflowRun{
				ID:   int64((page-1)*50 + i + 1),
				Name: fmt.Sprintf("Run %d", (page-1)*50+i+1),
			}
		}

		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(runs)
	}))
	defer server.Close()

	setupTestScanOptions()
	scanOptions.GiteaURL = server.URL
	scanOptions.RunsLimit = 0

	repo := &gitea.Repository{
		Name:     "test-repo",
		FullName: "owner/test-repo",
		Owner:    &gitea.User{UserName: "owner"},
	}

	runs, err := ListWorkflowRuns(nil, repo)

	assert.NoError(t, err)
	assert.Equal(t, 110, len(runs), "Should fetch 50+50+10 runs from array format")
	assert.Equal(t, 3, requestCount, "Should make 3 requests before encountering short page")
}

func TestScanWorkflowArtifacts_WithArtifacts(t *testing.T) {
	artifactsCallCount := 0
	zipCallCount := 0

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v1/repos/owner/test-repo/actions/runs/123/artifacts":
			artifactsCallCount++
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{
				"total_count": 1,
				"artifacts": [
					{"id": 1, "name": "test-artifact", "size": 100}
				]
			}`))
		case "/api/v1/repos/owner/test-repo/actions/artifacts/1/zip":
			zipCallCount++
			buf := new(bytes.Buffer)
			zw := zip.NewWriter(buf)
			f, _ := zw.Create("test.txt")
			_, _ = f.Write([]byte("test content"))
			_ = zw.Close()
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write(buf.Bytes())
		}
	}))
	defer server.Close()

	setupTestScanOptions()
	scanOptions.GiteaURL = server.URL

	repo := &gitea.Repository{
		Name:     "test-repo",
		FullName: "owner/test-repo",
		Owner:    &gitea.User{UserName: "owner"},
	}
	run := ActionWorkflowRun{ID: 123, HTMLURL: server.URL + "/run/123"}

	assert.NotPanics(t, func() {
		scanWorkflowArtifacts(nil, repo, run)
	})

	assert.Equal(t, 1, artifactsCallCount, "Should call artifacts API once")
	assert.Equal(t, 1, zipCallCount, "Should download artifact zip once")
}

func TestScanWorkflowArtifacts_NoArtifacts(t *testing.T) {
	callCount := 0

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"total_count": 0, "artifacts": []}`))
	}))
	defer server.Close()

	setupTestScanOptions()
	scanOptions.GiteaURL = server.URL

	repo := &gitea.Repository{
		Name:     "test-repo",
		FullName: "owner/test-repo",
		Owner:    &gitea.User{UserName: "owner"},
	}
	run := ActionWorkflowRun{ID: 123}

	assert.NotPanics(t, func() {
		scanWorkflowArtifacts(nil, repo, run)
	})

	assert.Equal(t, 1, callCount, "Should call artifacts API to check for artifacts")
}

func TestScanWorkflowArtifacts_ArtifactListError(t *testing.T) {
	callCount := 0

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		w.WriteHeader(http.StatusForbidden)
	}))
	defer server.Close()

	setupTestScanOptions()
	scanOptions.GiteaURL = server.URL

	repo := &gitea.Repository{
		Name:     "test-repo",
		FullName: "owner/test-repo",
		Owner:    &gitea.User{UserName: "owner"},
	}
	run := ActionWorkflowRun{ID: 123}

	assert.NotPanics(t, func() {
		scanWorkflowArtifacts(nil, repo, run)
	})

	assert.Equal(t, 1, callCount, "Should attempt API call even if it fails")
}

func TestDownloadAndScanArtifact_SuccessfulZipDownload(t *testing.T) {
	downloadCallCount := 0

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		downloadCallCount++
		buf := new(bytes.Buffer)
		zw := zip.NewWriter(buf)
		f, _ := zw.Create("test.txt")
		_, _ = f.Write([]byte("test content"))
		_ = zw.Close()

		w.Header().Set("Content-Type", "application/zip")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(buf.Bytes())
	}))
	defer server.Close()

	setupTestScanOptions()
	scanOptions.GiteaURL = server.URL

	repo := &gitea.Repository{
		Name:     "test-repo",
		FullName: "owner/test-repo",
		Owner:    &gitea.User{UserName: "owner"},
	}
	run := ActionWorkflowRun{ID: 123, HTMLURL: server.URL + "/run/123"}
	artifact := ActionArtifact{ID: 789, Name: "test-artifact.zip"}

	assert.NotPanics(t, func() {
		downloadAndScanArtifact(repo, run, artifact)
	})

	assert.Equal(t, 1, downloadCallCount, "Should download artifact once")
}

func TestDownloadAndScanArtifact_302Redirect(t *testing.T) {
	redirectCallCount := 0

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		redirectCallCount++
		w.WriteHeader(http.StatusFound)
		w.Header().Set("Location", "/redirect-target")
	}))
	defer server.Close()

	setupTestScanOptions()
	scanOptions.GiteaURL = server.URL

	repo := &gitea.Repository{
		Name:     "test-repo",
		FullName: "owner/test-repo",
		Owner:    &gitea.User{UserName: "owner"},
	}
	run := ActionWorkflowRun{ID: 123, HTMLURL: server.URL + "/run/123"}
	artifact := ActionArtifact{ID: 789, Name: "test-artifact"}

	assert.NotPanics(t, func() {
		downloadAndScanArtifact(repo, run, artifact)
	})

	assert.GreaterOrEqual(t, redirectCallCount, 1, "Should attempt to download artifact")
}

func TestDownloadAndScanArtifact_BuildURLError(t *testing.T) {
	setupTestScanOptions()
	scanOptions.GiteaURL = "://invalid-url"

	repo := &gitea.Repository{
		Name:     "test-repo",
		FullName: "owner/test-repo",
		Owner:    &gitea.User{UserName: "owner"},
	}
	run := ActionWorkflowRun{ID: 123}
	artifact := ActionArtifact{ID: 789, Name: "test"}

	assert.NotPanics(t, func() {
		downloadAndScanArtifact(repo, run, artifact)
	})
}

func TestScanJobLogs_BuildURLError(t *testing.T) {
	setupTestScanOptions()
	scanOptions.GiteaURL = "://invalid-url"

	repo := &gitea.Repository{
		Name:     "test-repo",
		FullName: "owner/test-repo",
		Owner:    &gitea.User{UserName: "owner"},
	}
	run := ActionWorkflowRun{ID: 123, HTMLURL: "https://example.com/run/123"}
	job := ActionJob{ID: 456, Name: "test-job"}

	assert.NotPanics(t, func() {
		scanJobLogs(nil, repo, run, job)
	})
}

func TestScanJobLogs_404Response(t *testing.T) {
	callCount := 0

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	setupTestScanOptions()
	scanOptions.GiteaURL = server.URL

	repo := &gitea.Repository{
		Name:     "test-repo",
		FullName: "owner/test-repo",
		Owner:    &gitea.User{UserName: "owner"},
	}
	run := ActionWorkflowRun{ID: 123, HTMLURL: server.URL + "/run/123"}
	job := ActionJob{ID: 456, Name: "test-job"}

	assert.NotPanics(t, func() {
		scanJobLogs(nil, repo, run, job)
	})

	assert.GreaterOrEqual(t, callCount, 1, "Should attempt to fetch logs")
}

func TestScanJobLogs_NonOKStatus(t *testing.T) {
	callCount := 0

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		w.WriteHeader(http.StatusForbidden)
	}))
	defer server.Close()

	setupTestScanOptions()
	scanOptions.GiteaURL = server.URL

	repo := &gitea.Repository{
		Name:     "test-repo",
		FullName: "owner/test-repo",
		Owner:    &gitea.User{UserName: "owner"},
	}
	run := ActionWorkflowRun{ID: 123, HTMLURL: server.URL + "/run/123"}
	job := ActionJob{ID: 456, Name: "test-job"}

	assert.NotPanics(t, func() {
		scanJobLogs(nil, repo, run, job)
	})

	assert.GreaterOrEqual(t, callCount, 1, "Should attempt to fetch logs even with forbidden response")
}

func TestScanWorkflowRunLogs_NoJobs(t *testing.T) {
	callCount := 0

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"total_count": 0, "jobs": []}`))
	}))
	defer server.Close()

	setupTestScanOptions()
	scanOptions.GiteaURL = server.URL

	repo := &gitea.Repository{
		Name:     "test-repo",
		FullName: "owner/test-repo",
		Owner:    &gitea.User{UserName: "owner"},
	}
	run := ActionWorkflowRun{ID: 123, HTMLURL: server.URL + "/run/123"}

	assert.NotPanics(t, func() {
		scanWorkflowRunLogs(nil, repo, run)
	})

	assert.Equal(t, 1, callCount, "Should call API to list jobs")
}

func TestScanWorkflowRunLogs_JobsError(t *testing.T) {
	callCount := 0

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	setupTestScanOptions()
	scanOptions.GiteaURL = server.URL

	repo := &gitea.Repository{
		Name:     "test-repo",
		FullName: "owner/test-repo",
		Owner:    &gitea.User{UserName: "owner"},
	}
	run := ActionWorkflowRun{ID: 123, HTMLURL: server.URL + "/run/123"}

	assert.NotPanics(t, func() {
		scanWorkflowRunLogs(nil, repo, run)
	})

	assert.Equal(t, 1, callCount, "Should attempt to fetch jobs even if it fails")
}

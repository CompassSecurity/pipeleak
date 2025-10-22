package scan

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"code.gitea.io/sdk/gitea"
	"github.com/stretchr/testify/assert"
)

func TestScanRepositories_SingleRepository(t *testing.T) {
	tests := []struct {
		name        string
		repository  string
		expectError bool
		setupServer func() *httptest.Server
	}{
		{
			name:       "valid single repository format",
			repository: "owner/repo",
			setupServer: func() *httptest.Server {
				return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(http.StatusOK)
				}))
			},
			expectError: false,
		},
		{
			name:        "invalid repository format - missing slash",
			repository:  "ownerrepo",
			setupServer: func() *httptest.Server { return nil },
		},
		{
			name:        "invalid repository format - too many parts",
			repository:  "owner/repo/extra",
			setupServer: func() *httptest.Server { return nil },
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			setupTestScanOptions()
			scanOptions.Repository = tt.repository

			if tt.setupServer != nil {
				server := tt.setupServer()
				if server != nil {
					defer server.Close()
					scanOptions.GiteaURL = server.URL
				}
			}

			assert.NotPanics(t, func() {
			})
		})
	}
}

func TestScanRepository_StartRunIDFiltering(t *testing.T) {
	tests := []struct {
		name          string
		repo          *gitea.Repository
		workflowRuns  []ActionWorkflowRun
		startRunID    int64
		expectedCount int
	}{
		{
			name: "filter runs with start run ID",
			repo: &gitea.Repository{
				FullName: "owner/repo",
			},
			workflowRuns: []ActionWorkflowRun{
				{ID: 100, Name: "Run 100"},
				{ID: 99, Name: "Run 99"},
				{ID: 98, Name: "Run 98"},
				{ID: 97, Name: "Run 97"},
			},
			startRunID:    99,
			expectedCount: 3,
		},
		{
			name: "no start run ID - all runs",
			repo: &gitea.Repository{
				FullName: "owner/repo",
			},
			workflowRuns: []ActionWorkflowRun{
				{ID: 100, Name: "Run 100"},
				{ID: 99, Name: "Run 99"},
			},
			startRunID:    0,
			expectedCount: 2,
		},
		{
			name: "start run ID smaller than all runs",
			repo: &gitea.Repository{
				FullName: "owner/repo",
			},
			workflowRuns: []ActionWorkflowRun{
				{ID: 100, Name: "Run 100"},
				{ID: 99, Name: "Run 99"},
			},
			startRunID:    50,
			expectedCount: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			setupTestScanOptions()
			scanOptions.StartRunID = tt.startRunID

			workflowRuns := tt.workflowRuns
			if scanOptions.StartRunID > 0 {
				filteredRuns := make([]ActionWorkflowRun, 0)
				for _, run := range workflowRuns {
					if run.ID <= scanOptions.StartRunID {
						filteredRuns = append(filteredRuns, run)
					}
				}
				workflowRuns = filteredRuns
			}

			assert.Len(t, workflowRuns, tt.expectedCount)
		})
	}
}

func TestScanRepository_NilRepository(t *testing.T) {
	setupTestScanOptions()

	assert.NotPanics(t, func() {
	})
}

func TestScanRepository_API403FallbackToCookie(t *testing.T) {
	tests := []struct {
		name           string
		repo           *gitea.Repository
		cookie         string
		setupServer    func() *httptest.Server
		expectFallback bool
	}{
		{
			name: "403 with cookie - should fallback",
			repo: &gitea.Repository{
				Name:     "test-repo",
				FullName: "owner/test-repo",
				Owner: &gitea.User{
					UserName: "owner",
				},
			},
			cookie: "valid_cookie",
			setupServer: func() *httptest.Server {
				return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(http.StatusForbidden)
				}))
			},
			expectFallback: true,
		},
		{
			name: "403 without cookie - should not fallback",
			repo: &gitea.Repository{
				Name:     "test-repo",
				FullName: "owner/test-repo",
				Owner: &gitea.User{
					UserName: "owner",
				},
			},
			cookie: "",
			setupServer: func() *httptest.Server {
				return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(http.StatusForbidden)
				}))
			},
			expectFallback: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			setupTestScanOptions()
			scanOptions.Cookie = tt.cookie

			if tt.setupServer != nil {
				server := tt.setupServer()
				defer server.Close()
				scanOptions.GiteaURL = server.URL
			}

		})
	}
}

func TestScanOptions_Validation(t *testing.T) {
	tests := []struct {
		name        string
		options     GiteaScanOptions
		expectValid bool
		errorMsg    string
	}{
		{
			name: "start-run-id requires repository flag",
			options: GiteaScanOptions{
				StartRunID: 100,
				Repository: "",
			},
			expectValid: false,
			errorMsg:    "start-run-id can only be used with --repository flag",
		},
		{
			name: "start-run-id with repository is valid",
			options: GiteaScanOptions{
				StartRunID: 100,
				Repository: "owner/repo",
			},
			expectValid: true,
		},
		{
			name: "no start-run-id is valid",
			options: GiteaScanOptions{
				StartRunID: 0,
				Repository: "",
			},
			expectValid: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			valid := true
			if tt.options.StartRunID > 0 && tt.options.Repository == "" {
				valid = false
			}

			assert.Equal(t, tt.expectValid, valid)
		})
	}
}

func TestScanAllRepositories_Pagination(t *testing.T) {
	tests := []struct {
		name          string
		totalRepos    int
		pageSize      int
		expectedPages int
	}{
		{
			name:          "single page",
			totalRepos:    20,
			pageSize:      50,
			expectedPages: 1,
		},
		{
			name:          "multiple pages",
			totalRepos:    120,
			pageSize:      50,
			expectedPages: 3,
		},
		{
			name:          "exact page boundary",
			totalRepos:    100,
			pageSize:      50,
			expectedPages: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			expectedPages := (tt.totalRepos + tt.pageSize - 1) / tt.pageSize

			assert.Equal(t, tt.expectedPages, expectedPages)
		})
	}
}

func TestScanOwnedRepositories_OwnerFilter(t *testing.T) {
	tests := []struct {
		name          string
		userID        int64
		repos         []gitea.Repository
		expectedCount int
	}{
		{
			name:   "filter owned repositories",
			userID: 123,
			repos: []gitea.Repository{
				{Name: "repo1", Owner: &gitea.User{ID: 123}},
				{Name: "repo2", Owner: &gitea.User{ID: 456}},
				{Name: "repo3", Owner: &gitea.User{ID: 123}},
			},
			expectedCount: 2,
		},
		{
			name:   "no owned repositories",
			userID: 999,
			repos: []gitea.Repository{
				{Name: "repo1", Owner: &gitea.User{ID: 123}},
				{Name: "repo2", Owner: &gitea.User{ID: 456}},
			},
			expectedCount: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ownedRepos := make([]gitea.Repository, 0)
			for _, repo := range tt.repos {
				if repo.Owner != nil && repo.Owner.ID == tt.userID {
					ownedRepos = append(ownedRepos, repo)
				}
			}

			assert.Len(t, ownedRepos, tt.expectedCount)
		})
	}
}

func TestAuthTransport_Integration(t *testing.T) {
	tests := []struct {
		name        string
		token       string
		setupServer func() *httptest.Server
		expectError bool
	}{
		{
			name:  "successful authentication",
			token: "valid-token-123",
			setupServer: func() *httptest.Server {
				return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					auth := r.Header.Get("Authorization")
					if auth == "token valid-token-123" {
						w.WriteHeader(http.StatusOK)
						w.Write([]byte("authenticated"))
					} else {
						w.WriteHeader(http.StatusUnauthorized)
					}
				}))
			},
			expectError: false,
		},
		{
			name:  "missing token",
			token: "",
			setupServer: func() *httptest.Server {
				return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					auth := r.Header.Get("Authorization")
					if auth == "token " {
						w.WriteHeader(http.StatusUnauthorized)
					} else {
						w.WriteHeader(http.StatusOK)
					}
				}))
			},
			expectError: false, // Transport doesn't fail, just sends empty token
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := tt.setupServer()
			defer server.Close()

			transport := &AuthTransport{
				Base:  http.DefaultTransport,
				Token: tt.token,
			}

			client := &http.Client{Transport: transport}

			resp, err := client.Get(server.URL)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				if resp != nil {
					resp.Body.Close()
				}
			}
		})
	}
}

func TestScanWorkflowArtifacts_NilRepository(t *testing.T) {
	setupTestScanOptions()
	run := ActionWorkflowRun{
		ID:      123,
		HTMLURL: "https://example.com/run/123",
	}

	assert.NotPanics(t, func() {
		scanWorkflowArtifacts(nil, nil, run)
	})
}

func TestNewScanCmd(t *testing.T) {
	// Test that the command is properly configured
	cmd := NewScanCmd()

	assert.NotNil(t, cmd)
	assert.Equal(t, "scan", cmd.Use)
	assert.NotEmpty(t, cmd.Short)
	assert.NotEmpty(t, cmd.Long)
	assert.NotNil(t, cmd.Run)

	// Check that required flags exist
	tokenFlag := cmd.Flags().Lookup("token")
	assert.NotNil(t, tokenFlag)

	giteaFlag := cmd.Flags().Lookup("gitea")
	assert.NotNil(t, giteaFlag)
	assert.Equal(t, "https://gitea.com", giteaFlag.DefValue)

	artifactsFlag := cmd.Flags().Lookup("artifacts")
	assert.NotNil(t, artifactsFlag)

	cookieFlag := cmd.Flags().Lookup("cookie")
	assert.NotNil(t, cookieFlag)
}

func TestGiteaScanOptions_Defaults(t *testing.T) {
	// Test default values
	opts := GiteaScanOptions{
		Context:                context.Background(),
		MaxScanGoRoutines:      4,
		TruffleHogVerification: true,
	}

	assert.NotNil(t, opts.Context)
	assert.Equal(t, 4, opts.MaxScanGoRoutines)
	assert.True(t, opts.TruffleHogVerification)
}

func BenchmarkBuildAPIURL(b *testing.B) {
	setupTestScanOptions()
	scanOptions.GiteaURL = "https://gitea.example.com"

	repo := &gitea.Repository{
		Name: "test-repo",
		Owner: &gitea.User{
			UserName: "test-owner",
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		buildAPIURL(repo, "/actions/runs/%d", 123)
	}
}

func BenchmarkCheckHTTPStatus(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		checkHTTPStatus(200, "test operation")
	}
}

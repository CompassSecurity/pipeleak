package secrets

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"code.gitea.io/sdk/gitea"
	"github.com/stretchr/testify/assert"
)

// handleVersionEndpoint handles the version endpoint for SDK initialization
func handleVersionEndpoint(w http.ResponseWriter, r *http.Request) bool {
	if r.URL.Path == "/api/v1/version" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]string{"version": "1.20.0"})
		return true
	}
	return false
}

func TestConfig(t *testing.T) {
	cfg := Config{
		URL:   "https://gitea.example.com",
		Token: "test-token",
	}

	if cfg.URL != "https://gitea.example.com" {
		t.Errorf("Expected URL to be https://gitea.example.com, got %s", cfg.URL)
	}

	if cfg.Token != "test-token" {
		t.Errorf("Expected Token to be test-token, got %s", cfg.Token)
	}
}

func TestFetchOrgSecrets_Pagination(t *testing.T) {
	tests := []struct {
		name          string
		totalSecrets  int
		pageSize      int
		expectedPages int
	}{
		{
			name:          "single page",
			totalSecrets:  10,
			pageSize:      50,
			expectedPages: 1,
		},
		{
			name:          "exact page boundary",
			totalSecrets:  50,
			pageSize:      50,
			expectedPages: 2,
		},
		{
			name:          "multiple pages",
			totalSecrets:  125,
			pageSize:      50,
			expectedPages: 3,
		},
		{
			name:          "empty result",
			totalSecrets:  0,
			pageSize:      50,
			expectedPages: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pageRequests := 0

			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if handleVersionEndpoint(w, r) {
					return
				}

				if r.URL.Path == "/api/v1/orgs/test-org/actions/secrets" {
					pageStr := r.URL.Query().Get("page")
					limitStr := r.URL.Query().Get("limit")

					page, _ := strconv.Atoi(pageStr)
					limit, _ := strconv.Atoi(limitStr)

					if page == 0 {
						page = 1
					}
					if limit == 0 {
						limit = 50
					}

					pageRequests++

					start := (page - 1) * limit
					end := start + limit
					if end > tt.totalSecrets {
						end = tt.totalSecrets
					}

					var secrets []*gitea.Secret
					for i := start; i < end; i++ {
						secrets = append(secrets, &gitea.Secret{
							Name: fmt.Sprintf("SECRET_%d", i+1),
						})
					}

					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(http.StatusOK)
					_ = json.NewEncoder(w).Encode(secrets)
					return
				}

				w.WriteHeader(http.StatusNotFound)
			}))
			defer server.Close()

			client, err := gitea.NewClient(server.URL, gitea.SetToken("test-token"))
			assert.NoError(t, err)

			err = fetchOrgSecrets(client, "test-org")
			assert.NoError(t, err)

			assert.Equal(t, tt.expectedPages, pageRequests)
		})
	}
}

func TestFetchRepoSecrets_Pagination(t *testing.T) {
	tests := []struct {
		name          string
		totalSecrets  int
		pageSize      int
		expectedPages int
	}{
		{
			name:          "single page",
			totalSecrets:  10,
			pageSize:      50,
			expectedPages: 1,
		},
		{
			name:          "exact page boundary",
			totalSecrets:  50,
			pageSize:      50,
			expectedPages: 2,
		},
		{
			name:          "multiple pages",
			totalSecrets:  125,
			pageSize:      50,
			expectedPages: 3,
		},
		{
			name:          "empty result",
			totalSecrets:  0,
			pageSize:      50,
			expectedPages: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pageRequests := 0

			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if handleVersionEndpoint(w, r) {
					return
				}

				if r.URL.Path == "/api/v1/repos/owner/repo/actions/secrets" {
					pageStr := r.URL.Query().Get("page")
					limitStr := r.URL.Query().Get("limit")

					page, _ := strconv.Atoi(pageStr)
					limit, _ := strconv.Atoi(limitStr)

					if page == 0 {
						page = 1
					}
					if limit == 0 {
						limit = 50
					}

					pageRequests++

					start := (page - 1) * limit
					end := start + limit
					if end > tt.totalSecrets {
						end = tt.totalSecrets
					}

					var secrets []*gitea.Secret
					for i := start; i < end; i++ {
						secrets = append(secrets, &gitea.Secret{
							Name: fmt.Sprintf("SECRET_%d", i+1),
						})
					}

					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(http.StatusOK)
					_ = json.NewEncoder(w).Encode(secrets)
					return
				}

				w.WriteHeader(http.StatusNotFound)
			}))
			defer server.Close()

			client, err := gitea.NewClient(server.URL, gitea.SetToken("test-token"))
			assert.NoError(t, err)

			err = fetchRepoSecrets(client, "owner", "repo")
			assert.NoError(t, err)

			assert.Equal(t, tt.expectedPages, pageRequests)
		})
	}
}

func TestFetchOrgSecrets_ErrorHandling(t *testing.T) {
	tests := []struct {
		name        string
		statusCode  int
		expectError bool
	}{
		{
			name:        "successful response",
			statusCode:  http.StatusOK,
			expectError: false,
		},
		{
			name:        "404 not found",
			statusCode:  http.StatusNotFound,
			expectError: true,
		},
		{
			name:        "401 unauthorized",
			statusCode:  http.StatusUnauthorized,
			expectError: true,
		},
		{
			name:        "500 server error",
			statusCode:  http.StatusInternalServerError,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if handleVersionEndpoint(w, r) {
					return
				}

				if tt.statusCode == http.StatusOK {
					secrets := []*gitea.Secret{
						{Name: "SECRET1"},
					}
					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(http.StatusOK)
					_ = json.NewEncoder(w).Encode(secrets)
				} else {
					w.WriteHeader(tt.statusCode)
					_, _ = w.Write([]byte(`{"message": "Error"}`))
				}
			}))
			defer server.Close()

			client, err := gitea.NewClient(server.URL, gitea.SetToken("test-token"))
			assert.NoError(t, err)

			err = fetchOrgSecrets(client, "test-org")

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestFetchRepoSecrets_ErrorHandling(t *testing.T) {
	tests := []struct {
		name        string
		statusCode  int
		expectError bool
	}{
		{
			name:        "successful response",
			statusCode:  http.StatusOK,
			expectError: false,
		},
		{
			name:        "404 not found",
			statusCode:  http.StatusNotFound,
			expectError: true,
		},
		{
			name:        "401 unauthorized",
			statusCode:  http.StatusUnauthorized,
			expectError: true,
		},
		{
			name:        "500 server error",
			statusCode:  http.StatusInternalServerError,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if handleVersionEndpoint(w, r) {
					return
				}

				if tt.statusCode == http.StatusOK {
					secrets := []*gitea.Secret{
						{Name: "SECRET1"},
					}
					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(http.StatusOK)
					_ = json.NewEncoder(w).Encode(secrets)
				} else {
					w.WriteHeader(tt.statusCode)
					_, _ = w.Write([]byte(`{"message": "Error"}`))
				}
			}))
			defer server.Close()

			client, err := gitea.NewClient(server.URL, gitea.SetToken("test-token"))
			assert.NoError(t, err)

			err = fetchRepoSecrets(client, "owner", "repo")

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

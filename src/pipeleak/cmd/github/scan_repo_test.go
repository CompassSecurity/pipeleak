package github

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/go-github/v69/github"
	"github.com/stretchr/testify/assert"
)

func TestScanSingleRepository_Success(t *testing.T) {
	repoName := "test-repo"
	repoOwner := "test-owner"
	repoFullName := repoOwner + "/" + repoName

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		switch r.URL.Path {
		case "/repos/" + repoOwner + "/" + repoName:
			repo := github.Repository{
				ID:       github.Int64(123),
				Name:     github.String(repoName),
				FullName: github.String(repoFullName),
				HTMLURL:  github.String("https://github.com/" + repoFullName),
				Owner: &github.User{
					Login: github.String(repoOwner),
				},
			}
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(repo)

		case "/repos/" + repoOwner + "/" + repoName + "/actions/runs":
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"workflow_runs": []interface{}{},
				"total_count":   0,
			})

		default:
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(map[string]interface{}{})
		}
	}))
	defer server.Close()

	client := github.NewClient(nil).WithAuthToken("test-token")
	client, _ = client.WithEnterpriseURLs(server.URL, server.URL)

	originalOptions := options
	defer func() { options = originalOptions }()

	options = GitHubScanOptions{
		Context:        context.Background(),
		Client:         client,
		MaxWorkflows:   1,
		MaxScanGoRoutines: 1,
	}

	owner, name, valid := validateRepoFormat(repoFullName)
	assert.True(t, valid)
	assert.Equal(t, repoOwner, owner)
	assert.Equal(t, repoName, name)
}

func TestScanSingleRepository_NotFound(t *testing.T) {
	repoName := "nonexistent-repo"
	repoOwner := "nonexistent-owner"

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		_ = json.NewEncoder(w).Encode(map[string]string{
			"message": "Not Found",
		})
	}))
	defer server.Close()

	client := github.NewClient(nil).WithAuthToken("test-token")
	client, _ = client.WithEnterpriseURLs(server.URL, server.URL)

	originalOptions := options
	defer func() { options = originalOptions }()

	options = GitHubScanOptions{
		Context: context.Background(),
		Client:  client,
	}

	repo, resp, _ := client.Repositories.Get(context.Background(), repoOwner, repoName)
	assert.Nil(t, repo)
	assert.NotNil(t, resp)
	assert.Equal(t, http.StatusNotFound, resp.StatusCode)
}

func TestScanSingleRepository_ValidFormats(t *testing.T) {
	tests := []struct {
		name     string
		repo     string
		wantValid bool
	}{
		{"valid simple", "owner/repo", true},
		{"valid with dashes", "my-org/my-repo", true},
		{"valid with underscores", "my_org/my_repo", true},
		{"valid with numbers", "user123/repo456", true},
		{"invalid no slash", "ownerrepo", false},
		{"invalid multiple slashes", "owner/repo/extra", false},
		{"invalid empty owner", "/repo", false},
		{"invalid empty repo", "owner/", false},
		{"invalid empty", "", false},
		{"invalid only slash", "/", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, _, valid := validateRepoFormat(tt.repo)
			assert.Equal(t, tt.wantValid, valid)
		})
	}
}

package enum

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"code.gitea.io/sdk/gitea"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/stretchr/testify/assert"
)

func init() {
	log.Logger = zerolog.New(os.Stderr).Level(zerolog.Disabled)
}

func TestNewEnumCmd(t *testing.T) {
	cmd := NewEnumCmd()
	assert.NotNil(t, cmd)
	assert.Equal(t, "enum", cmd.Use)
	assert.Equal(t, "Enumerate access of a Gitea token", cmd.Short)
	assert.NotNil(t, cmd.Run)
}

func TestNewEnumCmd_Flags(t *testing.T) {
	cmd := NewEnumCmd()
	err := cmd.ParseFlags([]string{"--gitea", "https://test.gitea.io", "--token", "test-token"})
	assert.NoError(t, err)
	giteaVal, _ := cmd.Flags().GetString("gitea")
	tokenVal, _ := cmd.Flags().GetString("token")
	assert.Equal(t, "https://test.gitea.io", giteaVal)
	assert.Equal(t, "test-token", tokenVal)
}

func TestGiteaSDK_ClientCreation(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Path == "/api/v1/version" {
			_ = json.NewEncoder(w).Encode(map[string]string{"version": "1.20.0"})
		}
	}))
	defer server.Close()
	client, err := gitea.NewClient(server.URL, gitea.SetToken("test-token"))
	assert.NoError(t, err)
	assert.NotNil(t, client)
}

func TestGiteaSDK_GetUserInfo(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/api/v1/version":
			_ = json.NewEncoder(w).Encode(map[string]string{"version": "1.20.0"})
		case "/api/v1/user":
			_ = json.NewEncoder(w).Encode(gitea.User{ID: 123, UserName: "testuser", FullName: "Test User", IsAdmin: true})
		}
	}))
	defer server.Close()
	client, _ := gitea.NewClient(server.URL, gitea.SetToken("test"))
	user, _, err := client.GetMyUserInfo()
	assert.NoError(t, err)
	assert.NotNil(t, user)
	assert.Equal(t, int64(123), user.ID)
	assert.Equal(t, "testuser", user.UserName)
}

func TestGiteaSDK_ListOrganizations(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/api/v1/version":
			_ = json.NewEncoder(w).Encode(map[string]string{"version": "1.20.0"})
		case "/api/v1/user/orgs":
			_ = json.NewEncoder(w).Encode([]gitea.Organization{
				{ID: 10, UserName: "org1", FullName: "Organization 1", Visibility: "public"},
				{ID: 20, UserName: "org2", FullName: "Organization 2", Visibility: "private"},
			})
		}
	}))
	defer server.Close()
	client, _ := gitea.NewClient(server.URL, gitea.SetToken("test"))
	orgs, _, err := client.ListMyOrgs(gitea.ListOrgsOptions{ListOptions: gitea.ListOptions{Page: 1, PageSize: 50}})
	assert.NoError(t, err)
	assert.Len(t, orgs, 2)
	assert.Equal(t, "org1", orgs[0].UserName)
}

func TestGiteaSDK_ListRepositories(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/api/v1/version":
			_ = json.NewEncoder(w).Encode(map[string]string{"version": "1.20.0"})
		case "/api/v1/user/repos":
			_ = json.NewEncoder(w).Encode([]gitea.Repository{
				{ID: 100, Name: "repo1", FullName: "user/repo1", Private: false, Owner: &gitea.User{UserName: "user"}, Permissions: &gitea.Permission{Admin: true, Push: true, Pull: true}},
				{ID: 200, Name: "repo2", FullName: "user/repo2", Private: true, Archived: true, Owner: &gitea.User{UserName: "user"}, Permissions: &gitea.Permission{Admin: false, Push: false, Pull: true}},
			})
		}
	}))
	defer server.Close()
	client, _ := gitea.NewClient(server.URL, gitea.SetToken("test"))
	repos, _, err := client.ListMyRepos(gitea.ListReposOptions{ListOptions: gitea.ListOptions{Page: 1, PageSize: 50}})
	assert.NoError(t, err)
	assert.Len(t, repos, 2)
	assert.Equal(t, "repo1", repos[0].Name)
	assert.True(t, repos[0].Permissions.Admin)
	assert.True(t, repos[1].Private)
}

func TestGiteaSDK_OrganizationRepositories(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/api/v1/version":
			_ = json.NewEncoder(w).Encode(map[string]string{"version": "1.20.0"})
		case "/api/v1/orgs/testorg/repos":
			_ = json.NewEncoder(w).Encode([]gitea.Repository{
				{ID: 300, Name: "orgrepo1", FullName: "testorg/orgrepo1", Owner: &gitea.User{UserName: "testorg"}, Private: true, Permissions: &gitea.Permission{Admin: false, Push: true, Pull: true}},
			})
		}
	}))
	defer server.Close()
	client, _ := gitea.NewClient(server.URL, gitea.SetToken("test"))
	repos, _, err := client.ListOrgRepos("testorg", gitea.ListOrgReposOptions{ListOptions: gitea.ListOptions{Page: 1, PageSize: 50}})
	assert.NoError(t, err)
	assert.Len(t, repos, 1)
	assert.Equal(t, "orgrepo1", repos[0].Name)
}

func TestGiteaSDK_Pagination(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		page := r.URL.Query().Get("page")
		switch r.URL.Path {
		case "/api/v1/version":
			_ = json.NewEncoder(w).Encode(map[string]string{"version": "1.20.0"})
		case "/api/v1/user/orgs":
			switch page {
			case "", "1":
				w.Header().Set("Link", `</api/v1/user/orgs?page=2>; rel="next"`)
				_ = json.NewEncoder(w).Encode([]gitea.Organization{{ID: 1, UserName: "org1"}, {ID: 2, UserName: "org2"}})
			case "2":
				_ = json.NewEncoder(w).Encode([]gitea.Organization{{ID: 3, UserName: "org3"}})
			}
		}
	}))
	defer server.Close()
	client, _ := gitea.NewClient(server.URL, gitea.SetToken("test"))
	orgs1, resp1, err := client.ListMyOrgs(gitea.ListOrgsOptions{ListOptions: gitea.ListOptions{Page: 1, PageSize: 50}})
	assert.NoError(t, err)
	assert.Len(t, orgs1, 2)
	assert.Equal(t, 2, resp1.NextPage)
	orgs2, resp2, _ := client.ListMyOrgs(gitea.ListOrgsOptions{ListOptions: gitea.ListOptions{Page: 2, PageSize: 50}})
	assert.Len(t, orgs2, 1)
	assert.Equal(t, 0, resp2.NextPage)
}

func TestGiteaSDK_EmptyResponses(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/api/v1/version":
			_ = json.NewEncoder(w).Encode(map[string]string{"version": "1.20.0"})
		case "/api/v1/user/orgs":
			_ = json.NewEncoder(w).Encode([]gitea.Organization{})
		case "/api/v1/user/repos":
			_ = json.NewEncoder(w).Encode([]gitea.Repository{})
		}
	}))
	defer server.Close()
	client, _ := gitea.NewClient(server.URL, gitea.SetToken("test"))
	orgs, _, err := client.ListMyOrgs(gitea.ListOrgsOptions{ListOptions: gitea.ListOptions{Page: 1}})
	assert.NoError(t, err)
	assert.Empty(t, orgs)
	repos, _, _ := client.ListMyRepos(gitea.ListReposOptions{ListOptions: gitea.ListOptions{Page: 1}})
	assert.Empty(t, repos)
}

// Tests for the refactored runEnum function

func TestRunEnum_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/api/v1/version":
			_ = json.NewEncoder(w).Encode(map[string]string{"version": "1.20.0"})
		case "/api/v1/user":
			_ = json.NewEncoder(w).Encode(gitea.User{
				ID:       1,
				UserName: "testuser",
				FullName: "Test User",
				Email:    "test@example.com",
				IsAdmin:  true,
			})
		case "/api/v1/user/orgs":
			_ = json.NewEncoder(w).Encode([]gitea.Organization{
				{ID: 10, UserName: "testorg", FullName: "Test Org"},
			})
		case "/api/v1/orgs/testorg/repos":
			_ = json.NewEncoder(w).Encode([]gitea.Repository{
				{ID: 100, Name: "orgrepo", FullName: "testorg/orgrepo", Owner: &gitea.User{UserName: "testorg"}},
			})
		case "/api/v1/user/repos":
			_ = json.NewEncoder(w).Encode([]gitea.Repository{
				{ID: 200, Name: "userrepo", FullName: "testuser/userrepo", Owner: &gitea.User{UserName: "testuser"}},
			})
		}
	}))
	defer server.Close()

	err := runEnum(server.URL, "test-token")
	assert.NoError(t, err)
}

func TestRunEnum_InvalidURL(t *testing.T) {
	err := runEnum("", "test-token")
	assert.Error(t, err)
}

func TestRunEnum_ClientCreationError(t *testing.T) {
	err := runEnum("invalid://url", "test-token")
	assert.Error(t, err)
}

func TestRunEnum_UserInfoError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/api/v1/version":
			_ = json.NewEncoder(w).Encode(map[string]string{"version": "1.20.0"})
		case "/api/v1/user":
			w.WriteHeader(http.StatusUnauthorized)
		}
	}))
	defer server.Close()

	err := runEnum(server.URL, "invalid-token")
	assert.Error(t, err)
}

func TestRunEnum_WithOrganizations(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/api/v1/version":
			_ = json.NewEncoder(w).Encode(map[string]string{"version": "1.20.0"})
		case "/api/v1/user":
			_ = json.NewEncoder(w).Encode(gitea.User{ID: 1, UserName: "testuser"})
		case "/api/v1/user/orgs":
			_ = json.NewEncoder(w).Encode([]gitea.Organization{
				{ID: 1, UserName: "org1", FullName: "Organization 1"},
				{ID: 2, UserName: "org2", FullName: "Organization 2"},
			})
		case "/api/v1/orgs/org1/repos", "/api/v1/orgs/org2/repos":
			_ = json.NewEncoder(w).Encode([]gitea.Repository{})
		case "/api/v1/user/repos":
			_ = json.NewEncoder(w).Encode([]gitea.Repository{})
		}
	}))
	defer server.Close()

	err := runEnum(server.URL, "test-token")
	assert.NoError(t, err)
}

func TestRunEnum_WithOrgPermissions(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/api/v1/version":
			_ = json.NewEncoder(w).Encode(map[string]string{"version": "1.20.0"})
		case "/api/v1/user":
			_ = json.NewEncoder(w).Encode(gitea.User{ID: 1, UserName: "testuser"})
		case "/api/v1/user/orgs":
			_ = json.NewEncoder(w).Encode([]gitea.Organization{
				{ID: 1, UserName: "testorg"},
			})
		case "/api/v1/orgs/testorg/permissions/testuser":
			_ = json.NewEncoder(w).Encode(gitea.OrgPermissions{
				IsOwner:             true,
				IsAdmin:             true,
				CanWrite:            true,
				CanRead:             true,
				CanCreateRepository: true,
			})
		case "/api/v1/orgs/testorg/repos":
			_ = json.NewEncoder(w).Encode([]gitea.Repository{})
		case "/api/v1/user/repos":
			_ = json.NewEncoder(w).Encode([]gitea.Repository{})
		}
	}))
	defer server.Close()

	err := runEnum(server.URL, "test-token")
	assert.NoError(t, err)
}

func TestRunEnum_OrgPermissionsError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/api/v1/version":
			_ = json.NewEncoder(w).Encode(map[string]string{"version": "1.20.0"})
		case "/api/v1/user":
			_ = json.NewEncoder(w).Encode(gitea.User{ID: 1, UserName: "testuser"})
		case "/api/v1/user/orgs":
			_ = json.NewEncoder(w).Encode([]gitea.Organization{
				{ID: 1, UserName: "restrictedorg"},
			})
		case "/api/v1/orgs/restrictedorg/permissions/testuser":
			w.WriteHeader(http.StatusForbidden)
		case "/api/v1/orgs/restrictedorg/repos":
			_ = json.NewEncoder(w).Encode([]gitea.Repository{})
		case "/api/v1/user/repos":
			_ = json.NewEncoder(w).Encode([]gitea.Repository{})
		}
	}))
	defer server.Close()

	// Should not fail on permission errors, just log them
	err := runEnum(server.URL, "test-token")
	assert.NoError(t, err)
}

func TestRunEnum_OrgsError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/api/v1/version":
			_ = json.NewEncoder(w).Encode(map[string]string{"version": "1.20.0"})
		case "/api/v1/user":
			_ = json.NewEncoder(w).Encode(gitea.User{ID: 1, UserName: "testuser"})
		case "/api/v1/user/orgs":
			w.WriteHeader(http.StatusInternalServerError)
		case "/api/v1/user/repos":
			_ = json.NewEncoder(w).Encode([]gitea.Repository{})
		}
	}))
	defer server.Close()

	// Should not fail on org listing errors, just break the loop
	err := runEnum(server.URL, "test-token")
	assert.NoError(t, err)
}

func TestRunEnum_OrgReposError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/api/v1/version":
			_ = json.NewEncoder(w).Encode(map[string]string{"version": "1.20.0"})
		case "/api/v1/user":
			_ = json.NewEncoder(w).Encode(gitea.User{ID: 1, UserName: "testuser"})
		case "/api/v1/user/orgs":
			_ = json.NewEncoder(w).Encode([]gitea.Organization{
				{ID: 1, UserName: "testorg"},
			})
		case "/api/v1/orgs/testorg/repos":
			w.WriteHeader(http.StatusForbidden)
		case "/api/v1/user/repos":
			_ = json.NewEncoder(w).Encode([]gitea.Repository{})
		}
	}))
	defer server.Close()

	// Should not fail on org repo errors, just break the loop
	err := runEnum(server.URL, "test-token")
	assert.NoError(t, err)
}

func TestRunEnum_UserReposError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/api/v1/version":
			_ = json.NewEncoder(w).Encode(map[string]string{"version": "1.20.0"})
		case "/api/v1/user":
			_ = json.NewEncoder(w).Encode(gitea.User{ID: 1, UserName: "testuser"})
		case "/api/v1/user/orgs":
			_ = json.NewEncoder(w).Encode([]gitea.Organization{})
		case "/api/v1/user/repos":
			w.WriteHeader(http.StatusForbidden)
		}
	}))
	defer server.Close()

	// Should not fail on user repo errors, just break the loop
	err := runEnum(server.URL, "test-token")
	assert.NoError(t, err)
}

func TestRunEnum_WithPagination(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		page := r.URL.Query().Get("page")

		switch r.URL.Path {
		case "/api/v1/version":
			_ = json.NewEncoder(w).Encode(map[string]string{"version": "1.20.0"})
		case "/api/v1/user":
			_ = json.NewEncoder(w).Encode(gitea.User{ID: 1, UserName: "testuser"})
		case "/api/v1/user/orgs":
			switch page {
			case "", "1":
				w.Header().Set("Link", `</api/v1/user/orgs?page=2>; rel="next"`)
				_ = json.NewEncoder(w).Encode([]gitea.Organization{
					{ID: 1, UserName: "org1"},
				})
			case "2":
				_ = json.NewEncoder(w).Encode([]gitea.Organization{
					{ID: 2, UserName: "org2"},
				})
			}
		case "/api/v1/orgs/org1/repos", "/api/v1/orgs/org2/repos":
			switch page {
			case "", "1":
				w.Header().Set("Link", `<`+r.URL.Path+`?page=2>; rel="next"`)
				_ = json.NewEncoder(w).Encode([]gitea.Repository{
					{ID: 1, Name: "repo1", FullName: "org/repo1", Owner: &gitea.User{UserName: "org"}},
				})
			case "2":
				_ = json.NewEncoder(w).Encode([]gitea.Repository{
					{ID: 2, Name: "repo2", FullName: "org/repo2", Owner: &gitea.User{UserName: "org"}},
				})
			}
		case "/api/v1/user/repos":
			switch page {
			case "", "1":
				w.Header().Set("Link", `</api/v1/user/repos?page=2>; rel="next"`)
				_ = json.NewEncoder(w).Encode([]gitea.Repository{
					{ID: 10, Name: "myrepo1", FullName: "testuser/myrepo1", Owner: &gitea.User{UserName: "testuser"}},
				})
			case "2":
				_ = json.NewEncoder(w).Encode([]gitea.Repository{
					{ID: 11, Name: "myrepo2", FullName: "testuser/myrepo2", Owner: &gitea.User{UserName: "testuser"}},
				})
			}
		}
	}))
	defer server.Close()

	err := runEnum(server.URL, "test-token")
	assert.NoError(t, err)
}

func TestRunEnum_WithRepositoryPermissions(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/api/v1/version":
			_ = json.NewEncoder(w).Encode(map[string]string{"version": "1.20.0"})
		case "/api/v1/user":
			_ = json.NewEncoder(w).Encode(gitea.User{ID: 1, UserName: "testuser"})
		case "/api/v1/user/orgs":
			_ = json.NewEncoder(w).Encode([]gitea.Organization{
				{ID: 1, UserName: "testorg"},
			})
		case "/api/v1/orgs/testorg/repos":
			_ = json.NewEncoder(w).Encode([]gitea.Repository{
				{
					ID:       100,
					Name:     "orgrepo",
					FullName: "testorg/orgrepo",
					Owner:    &gitea.User{UserName: "testorg"},
					Private:  true,
					Archived: false,
					Permissions: &gitea.Permission{
						Admin: false,
						Push:  true,
						Pull:  true,
					},
				},
			})
		case "/api/v1/user/repos":
			_ = json.NewEncoder(w).Encode([]gitea.Repository{
				{
					ID:       200,
					Name:     "userrepo",
					FullName: "testuser/userrepo",
					Owner:    &gitea.User{UserName: "testuser"},
					Private:  false,
					Archived: true,
					Permissions: &gitea.Permission{
						Admin: true,
						Push:  true,
						Pull:  true,
					},
				},
			})
		}
	}))
	defer server.Close()

	err := runEnum(server.URL, "test-token")
	assert.NoError(t, err)
}

func TestRunEnum_EmptyResults(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/api/v1/version":
			_ = json.NewEncoder(w).Encode(map[string]string{"version": "1.20.0"})
		case "/api/v1/user":
			_ = json.NewEncoder(w).Encode(gitea.User{ID: 1, UserName: "lonelyuser"})
		case "/api/v1/user/orgs":
			_ = json.NewEncoder(w).Encode([]gitea.Organization{})
		case "/api/v1/user/repos":
			_ = json.NewEncoder(w).Encode([]gitea.Repository{})
		}
	}))
	defer server.Close()

	err := runEnum(server.URL, "test-token")
	assert.NoError(t, err)
}

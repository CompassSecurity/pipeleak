package devops

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestPaginatedResponse(t *testing.T) {
	t.Run("creates paginated response with projects", func(t *testing.T) {
		resp := PaginatedResponse[Project]{
			Count: 2,
			Value: []Project{
				{ID: "1", Name: "Project 1"},
				{ID: "2", Name: "Project 2"},
			},
		}
		
		assert.Equal(t, 2, resp.Count)
		assert.Len(t, resp.Value, 2)
		assert.Equal(t, "Project 1", resp.Value[0].Name)
	})

	t.Run("creates empty paginated response", func(t *testing.T) {
		resp := PaginatedResponse[Build]{
			Count: 0,
			Value: []Build{},
		}
		
		assert.Equal(t, 0, resp.Count)
		assert.Empty(t, resp.Value)
	})
}

func TestAuthenticatedUser(t *testing.T) {
	t.Run("creates authenticated user with all fields", func(t *testing.T) {
		now := time.Now()
		user := AuthenticatedUser{
			DisplayName:  "Test User",
			PublicAlias:  "testuser",
			EmailAddress: "test@example.com",
			CoreRevision: 1,
			TimeStamp:    now,
			ID:           "user-123",
			Revision:     5,
		}
		
		assert.Equal(t, "Test User", user.DisplayName)
		assert.Equal(t, "testuser", user.PublicAlias)
		assert.Equal(t, "test@example.com", user.EmailAddress)
		assert.Equal(t, "user-123", user.ID)
		assert.Equal(t, 5, user.Revision)
	})
}

func TestAccount(t *testing.T) {
	t.Run("creates account", func(t *testing.T) {
		account := Account{
			AccountID:   "acc-123",
			AccountURI:  "https://dev.azure.com/myorg",
			AccountName: "MyOrganization",
		}
		
		assert.Equal(t, "acc-123", account.AccountID)
		assert.Equal(t, "https://dev.azure.com/myorg", account.AccountURI)
		assert.Equal(t, "MyOrganization", account.AccountName)
	})
}

func TestProject(t *testing.T) {
	t.Run("creates project with all fields", func(t *testing.T) {
		now := time.Now()
		project := Project{
			ID:             "proj-123",
			Name:           "MyProject",
			URL:            "https://dev.azure.com/myorg/_apis/projects/proj-123",
			State:          "wellFormed",
			Revision:       10,
			Visibility:     "private",
			LastUpdateTime: now,
		}
		
		assert.Equal(t, "proj-123", project.ID)
		assert.Equal(t, "MyProject", project.Name)
		assert.Equal(t, "wellFormed", project.State)
		assert.Equal(t, "private", project.Visibility)
		assert.Equal(t, 10, project.Revision)
	})

	t.Run("creates public project", func(t *testing.T) {
		project := Project{
			ID:         "proj-pub",
			Name:       "PublicProject",
			Visibility: "public",
		}
		
		assert.Equal(t, "public", project.Visibility)
	})
}

func TestBuild(t *testing.T) {
	t.Run("creates build with links", func(t *testing.T) {
		build := Build{}
		build.Links.Self.Href = "https://dev.azure.com/myorg/_apis/build/builds/123"
		build.Links.Web.Href = "https://dev.azure.com/myorg/_build/results?buildId=123"
		
		assert.Equal(t, "https://dev.azure.com/myorg/_apis/build/builds/123", build.Links.Self.Href)
		assert.Equal(t, "https://dev.azure.com/myorg/_build/results?buildId=123", build.Links.Web.Href)
	})
}

func TestBuildLog(t *testing.T) {
	t.Run("creates build log", func(t *testing.T) {
		log := BuildLog{
			ID:        10,
			Type:      "Container",
			URL:       "https://dev.azure.com/myorg/_apis/build/builds/123/logs/10",
			LineCount: 150,
		}
		
		assert.Equal(t, 10, log.ID)
		assert.Equal(t, "Container", log.Type)
		assert.Equal(t, 150, log.LineCount)
	})
}

func TestArtifact(t *testing.T) {
	t.Run("creates artifact", func(t *testing.T) {
		artifact := Artifact{
			ID:   1,
			Name: "drop",
		}
		artifact.Resource.Type = "Container"
		artifact.Resource.Data = "artifact-data"
		artifact.Resource.DownloadURL = "https://dev.azure.com/myorg/_apis/build/builds/123/artifacts?artifactName=drop"
		
		assert.Equal(t, 1, artifact.ID)
		assert.Equal(t, "drop", artifact.Name)
		assert.Equal(t, "Container", artifact.Resource.Type)
		assert.Contains(t, artifact.Resource.DownloadURL, "artifactName=drop")
	})
}

func TestDevOpsScanOptions(t *testing.T) {
	t.Run("creates scan options with defaults", func(t *testing.T) {
		opts := DevOpsScanOptions{
			Username:               "testuser",
			AccessToken:            "token123",
			Verbose:                false,
			MaxScanGoRoutines:      4,
			TruffleHogVerification: true,
			MaxBuilds:              -1,
			Artifacts:              false,
		}
		
		assert.Equal(t, "testuser", opts.Username)
		assert.Equal(t, "token123", opts.AccessToken)
		assert.False(t, opts.Verbose)
		assert.Equal(t, 4, opts.MaxScanGoRoutines)
		assert.True(t, opts.TruffleHogVerification)
		assert.Equal(t, -1, opts.MaxBuilds)
		assert.False(t, opts.Artifacts)
	})

	t.Run("creates scan options with custom values", func(t *testing.T) {
		opts := DevOpsScanOptions{
			Username:               "admin",
			AccessToken:            "secret",
			Verbose:                true,
			ConfidenceFilter:       []string{"high", "medium"},
			MaxScanGoRoutines:      8,
			TruffleHogVerification: false,
			MaxBuilds:              10,
			Organization:           "myorg",
			Project:                "myproject",
			Artifacts:              true,
		}
		
		assert.Equal(t, "admin", opts.Username)
		assert.True(t, opts.Verbose)
		assert.Len(t, opts.ConfidenceFilter, 2)
		assert.Equal(t, 8, opts.MaxScanGoRoutines)
		assert.False(t, opts.TruffleHogVerification)
		assert.Equal(t, 10, opts.MaxBuilds)
		assert.Equal(t, "myorg", opts.Organization)
		assert.Equal(t, "myproject", opts.Project)
		assert.True(t, opts.Artifacts)
	})
}

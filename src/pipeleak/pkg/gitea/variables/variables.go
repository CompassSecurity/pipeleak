package variables

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"code.gitea.io/sdk/gitea"
	"github.com/CompassSecurity/pipeleak/pkg/httpclient"
	"github.com/hashicorp/go-retryablehttp"
	"github.com/rs/zerolog/log"
)

type Config struct {
	URL   string
	Token string
}

// clientContext holds both the SDK client and configuration needed for direct API calls
type clientContext struct {
	client     *gitea.Client
	httpClient *retryablehttp.Client
	url        string
}

func ListAllVariables(cfg Config) error {
	ctx, err := createClientContext(cfg)
	if err != nil {
		return fmt.Errorf("failed to create Gitea client: %w", err)
	}

	orgs, err := fetchOrganizations(ctx.client)
	if err != nil {
		return fmt.Errorf("failed to fetch organizations: %w", err)
	}

	log.Info().Int("count", len(orgs)).Msg("Found organizations")

	for _, org := range orgs {
		if err := processOrganization(ctx, org); err != nil {
			log.Warn().Err(err).Str("org", org.UserName).Msg("Failed to process organization")
			continue
		}
	}

	return nil
}

func createClientContext(cfg Config) (*clientContext, error) {
	authHeaders := map[string]string{"Authorization": "token " + cfg.Token}
	retryableClient := httpclient.GetPipeleakHTTPClient("", nil, authHeaders)

	client, err := gitea.NewClient(cfg.URL, gitea.SetToken(cfg.Token), gitea.SetHTTPClient(retryableClient.StandardClient()))
	if err != nil {
		return nil, err
	}

	return &clientContext{
		client:     client,
		httpClient: retryableClient,
		url:        cfg.URL,
	}, nil
}

func fetchOrganizations(client *gitea.Client) ([]*gitea.Organization, error) {
	var allOrgs []*gitea.Organization
	page := 1
	pageSize := 50

	for {
		orgs, resp, err := client.ListMyOrgs(gitea.ListOrgsOptions{
			ListOptions: gitea.ListOptions{
				Page:     page,
				PageSize: pageSize,
			},
		})
		if err != nil {
			return nil, err
		}

		allOrgs = append(allOrgs, orgs...)

		if resp == nil || len(orgs) < pageSize {
			break
		}
		page++
	}

	return allOrgs, nil
}

func processOrganization(ctx *clientContext, org *gitea.Organization) error {
	log.Debug().Str("org", org.UserName).Msg("Processing organization")

	if err := fetchOrgVariables(ctx.client, org.UserName); err != nil {
		log.Warn().Err(err).Str("org", org.UserName).Msg("Failed to fetch organization variables")
	}

	repos, err := fetchOrgRepositories(ctx.client, org.UserName)
	if err != nil {
		return fmt.Errorf("failed to fetch repositories for org %s: %w", org.UserName, err)
	}

	log.Debug().Str("org", org.UserName).Int("repo_count", len(repos)).Msg("Found repositories")

	for _, repo := range repos {
		if err := fetchRepoVariables(ctx, org.UserName, repo.Name); err != nil {
			log.Warn().Err(err).Str("org", org.UserName).Str("repo", repo.Name).Msg("Failed to fetch repository variables")
		}
	}

	return nil
}

func fetchOrgVariables(client *gitea.Client, orgName string) error {
	page := 1
	pageSize := 50

	for {
		variables, resp, err := client.ListOrgActionVariable(orgName, gitea.ListOrgActionVariableOption{
			ListOptions: gitea.ListOptions{
				Page:     page,
				PageSize: pageSize,
			},
		})
		if err != nil {
			return err
		}

		for _, v := range variables {
			log.Info().
				Str("org", orgName).
				Str("variable_name", v.Name).
				Str("type", "organization").
				Str("value", v.Data).
				Msg("Variable")
		}

		if resp == nil || len(variables) < pageSize {
			break
		}
		page++
	}

	return nil
}

func fetchOrgRepositories(client *gitea.Client, orgName string) ([]*gitea.Repository, error) {
	var allRepos []*gitea.Repository
	page := 1
	pageSize := 50

	for {
		repos, resp, err := client.ListOrgRepos(orgName, gitea.ListOrgReposOptions{
			ListOptions: gitea.ListOptions{
				Page:     page,
				PageSize: pageSize,
			},
		})
		if err != nil {
			return nil, err
		}

		allRepos = append(allRepos, repos...)

		if resp == nil || len(repos) < pageSize {
			break
		}
		page++
	}

	return allRepos, nil
}

// fetchRepoVariables fetches all variables for a specific repository using the Gitea API.
// The SDK doesn't provide a ListRepoActionVariable method, so we use a direct API call.
func fetchRepoVariables(ctx *clientContext, owner, repo string) error {
	page := 1
	pageSize := 50

	for {
		variables, err := listRepoActionVariables(ctx, owner, repo, page, pageSize)
		if err != nil {
			return err
		}

		for _, v := range variables {
			log.Info().
				Str("org", owner).
				Str("repo", repo).
				Str("variable_name", v.Name).
				Str("type", "repository").
				Str("value", v.Value).
				Msg("Variable")
		}

		if len(variables) < pageSize {
			break
		}
		page++
	}

	return nil
}

// listRepoActionVariables calls the Gitea API directly to list repository action variables.
// This implements the missing SDK method by making a direct HTTP request.
func listRepoActionVariables(ctx *clientContext, owner, repo string, page, pageSize int) ([]*gitea.RepoActionVariable, error) {
	url := fmt.Sprintf("%s/api/v1/repos/%s/%s/actions/variables?page=%d&limit=%d", ctx.url, owner, repo, page, pageSize)

	req, err := retryablehttp.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := ctx.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(body))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	var variables []*gitea.RepoActionVariable
	if err := json.Unmarshal(body, &variables); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return variables, nil
}

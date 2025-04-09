package devops

import (
	"net/url"
	"path"

	"github.com/rs/zerolog/log"

	"resty.dev/v3"
)

// https://learn.microsoft.com/en-us/rest/api/azure/devops/
type AzureDevOpsApiClient struct {
	Client resty.Client
}

func NewClient(username string, password string) AzureDevOpsApiClient {
	bbClient := AzureDevOpsApiClient{Client: *resty.New().SetBasicAuth(username, password).SetRedirectPolicy(resty.FlexibleRedirectPolicy(5))}
	bbClient.Client.AddRetryHooks(
		func(res *resty.Response, err error) {
			if 429 == res.StatusCode() {
				log.Info().Int("status", res.StatusCode()).Msg("Retrying request, we are rate limited")
			} else {
				log.Debug().Int("status", res.StatusCode()).Msg("Retrying request, not due to rate limit")
			}
		},
	)
	return bbClient
}

// https://learn.microsoft.com/en-us/rest/api/azure/devops/core/projects/list?view=azure-devops-rest-7.2&tabs=HTTP
func (a AzureDevOpsApiClient) ListRepositories(organization string) ([]Repository, *resty.Response, error) {
	reqUrl := ""
	u, err := url.Parse("https://dev.azure.com/")
	if err != nil {
		log.Fatal().Err(err).Msg("Unable to parse ListWorkspaceRepositoires url")
	}
	u.Path = path.Join(u.Path, organization, "_apis", "projects")
	reqUrl = u.String()

	resp := &PaginatedResponse[Repository]{}
	res, err := a.Client.R().
		SetQueryParam("api-version", "7.2-preview.4").
		SetResult(resp).
		Get(reqUrl)

	if res.StatusCode() == 404 || res.StatusCode() == 401 {
		log.Fatal().Int("status", res.StatusCode()).Str("organization", organization).Msg("Organization does not exist or you do not have access")
	}

	return resp.Value, res, err
}

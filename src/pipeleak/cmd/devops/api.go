package devops

import (
	"net/url"
	"path"
	"strconv"

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
				log.Info().Int("status", res.StatusCode()).Msg("Retrying request, not due to rate limit")
			}
		},
	)
	return bbClient
}

// https://learn.microsoft.com/en-us/rest/api/azure/devops/profile/profiles/get?view=azure-devops-rest-7.2&tabs=HTTP
func (a AzureDevOpsApiClient) GetAuthenticatedUser() (*AuthenticatedUser, *resty.Response, error) {
	u, err := url.Parse("https://app.vssps.visualstudio.com/_apis/profile/profiles/me")
	if err != nil {
		log.Fatal().Err(err).Msg("Unable to parse GetAuthenticatedUser url")
	}
	reqUrl := u.String()

	user := &AuthenticatedUser{}
	res, err := a.Client.R().
		SetQueryParam("api-version", "7.2-preview.3").
		SetResult(user).
		Get(reqUrl)

	if res.StatusCode() > 400 {
		log.Fatal().Int("status", res.StatusCode()).Str("response", res.String()).Msg("Failed fetching authenticated user")
	}

	return user, res, err
}

// https://learn.microsoft.com/en-us/rest/api/azure/devops/account/accounts/list?view=azure-devops-rest-7.2&tabs=HTTP
func (a AzureDevOpsApiClient) ListAccounts(ownerId string) ([]Account, *resty.Response, error) {
	u, err := url.Parse("https://app.vssps.visualstudio.com/_apis/accounts")
	if err != nil {
		log.Fatal().Err(err).Msg("Unable to parse ListAccounts url")
	}
	reqUrl := u.String()

	resp := &PaginatedResponse[Account]{}
	res, err := a.Client.R().
		SetQueryParam("api-version", "7.2-preview.1").
		SetQueryParam("ownerId", ownerId).
		SetResult(resp).
		Get(reqUrl)

	if res.StatusCode() > 400 {
		log.Fatal().Int("status", res.StatusCode()).Str("ownerId", ownerId).Msg("Fetching accounts failed")
	}

	return resp.Value, res, err
}

// https://learn.microsoft.com/en-us/rest/api/azure/devops/core/projects/list?view=azure-devops-rest-7.2&tabs=HTTP
func (a AzureDevOpsApiClient) ListProjects(continuationToken string, organization string) ([]Project, *resty.Response, string, error) {
	reqUrl := ""
	u, err := url.Parse("https://dev.azure.com/")
	if err != nil {
		log.Fatal().Err(err).Msg("Unable to parse ListProjects url")
	}
	u.Path = path.Join(u.Path, organization, "_apis", "projects")
	reqUrl = u.String()

	resp := &PaginatedResponse[Project]{}
	res, err := a.Client.R().
		SetQueryParam("api-version", "7.2-preview.4").
		SetQueryParam("$top", "100").
		SetQueryParam("continuationtoken", continuationToken).
		SetResult(resp).
		Get(reqUrl)

	if res.StatusCode() == 404 || res.StatusCode() == 401 {
		log.Fatal().Int("status", res.StatusCode()).Str("organization", organization).Msg("Projects list does not exist or you do not have access")
	}

	return resp.Value, res, res.Header().Get("x-ms-continuationtoken"), err
}

// https://learn.microsoft.com/en-us/rest/api/azure/devops/build/builds/list?view=azure-devops-rest-7.2
func (a AzureDevOpsApiClient) ListBuilds(continuationToken string, organization string, project string) ([]Build, *resty.Response, string, error) {
	reqUrl := ""
	u, err := url.Parse("https://dev.azure.com/")
	if err != nil {
		log.Fatal().Err(err).Msg("Unable to parse ListBuilds url")
	}

	u.Path = path.Join(u.Path, organization, project, "_apis", "build", "builds")
	reqUrl = u.String()

	resp := &PaginatedResponse[Build]{}
	res, err := a.Client.R().
		SetQueryParam("api-version", "7.2-preview.7").
		SetQueryParam("$top", "100").
		SetQueryParam("continuationtoken", continuationToken).
		SetResult(resp).
		Get(reqUrl)

	if res.StatusCode() == 404 || res.StatusCode() == 401 {
		log.Fatal().Int("status", res.StatusCode()).Str("project", project).Str("organization", organization).Msg("Build list does not exist or you do not have access")
	}

	return resp.Value, res, res.Header().Get("x-ms-continuationtoken"), err
}

// https://learn.microsoft.com/en-us/rest/api/azure/devops/build/builds/get-build-logs?view=azure-devops-rest-7.2
// this endpoint is NOT paged
func (a AzureDevOpsApiClient) ListBuildLogs(organization string, project string, buildId int) ([]BuildLog, *resty.Response, error) {
	reqUrl := ""
	u, err := url.Parse("https://dev.azure.com/")
	if err != nil {
		log.Fatal().Err(err).Msg("Unable to parse ListBuilds url")
	}

	u.Path = path.Join(u.Path, organization, project, "_apis", "build", "builds", strconv.Itoa(buildId), "logs")
	reqUrl = u.String()

	resp := &PaginatedResponse[BuildLog]{}
	res, err := a.Client.R().
		SetQueryParam("api-version", "7.2-preview.2").
		SetResult(resp).
		Get(reqUrl)

	if res.StatusCode() == 404 || res.StatusCode() == 401 {
		log.Fatal().Int("status", res.StatusCode()).Str("project", project).Str("organization", organization).Msg("Build log list does not exist or you do not have access")
	}

	return resp.Value, res, err
}

// https://learn.microsoft.com/en-us/rest/api/azure/devops/build/builds/get-build-log?view=azure-devops-rest-7.2
func (a AzureDevOpsApiClient) GetLog(organization string, project string, buildId int, logId int) ([]byte, *resty.Response, error) {
	reqUrl := ""
	u, err := url.Parse("https://dev.azure.com/")
	if err != nil {
		log.Fatal().Err(err).Msg("Unable to parse ListBuilds url")
	}

	u.Path = path.Join(u.Path, organization, project, "_apis", "build", "builds", strconv.Itoa(buildId), "logs", strconv.Itoa(logId))
	reqUrl = u.String()

	res, err := a.Client.R().
		SetQueryParam("api-version", "7.2-preview.2").
		Get(reqUrl)

	if res.StatusCode() == 404 || res.StatusCode() == 401 {
		log.Error().Int("status", res.StatusCode()).Str("project", project).Str("organization", organization).Msg("Log does not exist or you do not have access")
	}

	return res.Bytes(), res, err
}

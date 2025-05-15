package bitbucket

import (
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"path"
	"strconv"
	"time"

	"github.com/rs/zerolog/log"

	"resty.dev/v3"
)

// Docs: https://developer.atlassian.com/cloud/bitbucket/rest/intro/
type BitBucketApiClient struct {
	Client resty.Client
}

func NewClient(username string, password string, bitBucketCookie string) BitBucketApiClient {
	client := *resty.New().SetBasicAuth(username, password).SetRedirectPolicy(resty.FlexibleRedirectPolicy(5))
	if len(bitBucketCookie) > 0 {
		jar, _ := cookiejar.New(nil)
		targetURL, _ := url.Parse("https://bitbucket.org/")
		jar.SetCookies(targetURL, []*http.Cookie{
			{
				Name:  "cloud.session.token",
				Value: bitBucketCookie,
				Path:  "/!api/internal",
			},
		})
		client.SetCookieJar(jar)
		log.Debug().Msg("Added cloud.session.token to HTTP client")
	}

	bbClient := BitBucketApiClient{Client: client}
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

// https://developer.atlassian.com/cloud/bitbucket/rest/api-group-workspaces/#api-workspaces-get
func (a BitBucketApiClient) ListOwnedWorkspaces(nextPageUrl string) ([]Workspace, string, *resty.Response, error) {
	url := "https://api.bitbucket.org/2.0/workspaces"
	if nextPageUrl != "" {
		url = nextPageUrl
	}
	resp := &PaginatedResponse[Workspace]{}
	res, err := a.Client.R().
		SetQueryParam("sort", "-updated_on").
		SetResult(resp).
		Get(url)

	return resp.Values, resp.Next, res, err
}

// https://developer.atlassian.com/cloud/bitbucket/rest/api-group-repositories/#api-repositories-workspace-get
func (a BitBucketApiClient) ListWorkspaceRepositoires(nextPageUrl string, workspaceSlug string) ([]Repository, string, *resty.Response, error) {
	reqUrl := ""
	if nextPageUrl != "" {
		reqUrl = nextPageUrl
	} else {
		u, err := url.Parse("https://api.bitbucket.org/2.0/repositories/")
		if err != nil {
			log.Fatal().Err(err).Msg("Unable to parse ListWorkspaceRepositoires url")
		}
		u.Path = path.Join(u.Path, workspaceSlug)
		reqUrl = u.String()
	}

	resp := &PaginatedResponse[Repository]{}
	res, err := a.Client.R().
		SetQueryParam("sort", "-updated_on").
		SetResult(resp).
		Get(reqUrl)

	return resp.Values, resp.Next, res, err
}

// https://developer.atlassian.com/cloud/bitbucket/rest/api-group-repositories/#api-repositories-get
func (a BitBucketApiClient) ListPublicRepositories(nextPageUrl string, after time.Time) ([]PublicRepository, string, *resty.Response, error) {
	reqUrl := ""
	if nextPageUrl != "" {
		reqUrl = nextPageUrl
	} else {
		u, err := url.Parse("https://api.bitbucket.org/2.0/repositories/")
		if err != nil {
			log.Fatal().Err(err).Msg("Unable to parse ListPublicRepositories url")
		}
		reqUrl = u.String()
	}

	resp := &PaginatedResponse[PublicRepository]{}
	// only set it initially after that the next url does use the after query param for paging
	if nextPageUrl == "" {
		res, err := a.Client.R().SetResult(resp).SetQueryParam("after", after.Format(time.RFC3339)).Get(reqUrl)
		return resp.Values, resp.Next, res, err
	} else {
		res, err := a.Client.R().SetResult(resp).Get(reqUrl)
		return resp.Values, resp.Next, res, err
	}
}

// https://developer.atlassian.com/cloud/bitbucket/rest/api-group-pipelines/#api-repositories-workspace-repo-slug-pipelines-get
func (a BitBucketApiClient) ListRepositoryPipelines(nextPageUrl string, workspaceSlug string, repoSlug string) ([]Pipeline, string, *resty.Response, error) {
	reqUrl := ""
	if nextPageUrl != "" {
		reqUrl = nextPageUrl
	} else {
		u, err := url.Parse("https://api.bitbucket.org/2.0/repositories/")
		if err != nil {
			log.Fatal().Err(err).Msg("Unable to parse ListRepositoryPipelines url")
		}
		u.Path = path.Join(u.Path, workspaceSlug, repoSlug, "pipelines")
		reqUrl = u.String()
	}

	resp := &PaginatedResponse[Pipeline]{}
	res, err := a.Client.R().
		SetQueryParam("sort", "-created_on").
		SetResult(resp).
		Get(reqUrl)

	return resp.Values, resp.Next, res, err
}

// https://developer.atlassian.com/cloud/bitbucket/rest/api-group-pipelines/#api-repositories-workspace-repo-slug-pipelines-pipeline-uuid-steps-get
func (a BitBucketApiClient) ListPipelineSteps(nextPageUrl string, workspaceSlug string, repoSlug string, pipelineUUID string) ([]PipelineStep, string, *resty.Response, error) {
	reqUrl := ""
	if nextPageUrl != "" {
		reqUrl = nextPageUrl
	} else {
		u, err := url.Parse("https://api.bitbucket.org/2.0/repositories/")
		if err != nil {
			log.Fatal().Err(err).Msg("Unable to parse ListPipelineSteps url")
		}
		u.Path = path.Join(u.Path, workspaceSlug, repoSlug, "pipelines", pipelineUUID, "steps")
		reqUrl = u.String()
	}

	resp := &PaginatedResponse[PipelineStep]{}
	res, err := a.Client.R().
		SetResult(resp).
		Get(reqUrl)

	return resp.Values, resp.Next, res, err
}

// https://developer.atlassian.com/cloud/bitbucket/rest/api-group-pipelines/#api-repositories-workspace-repo-slug-pipelines-pipeline-uuid-steps-step-uuid-log-get
func (a BitBucketApiClient) GetStepLog(workspaceSlug string, repoSlug string, pipelineUUID string, stepUUID string) ([]byte, *resty.Response, error) {
	u, err := url.Parse("https://api.bitbucket.org/2.0/repositories/")
	if err != nil {
		log.Fatal().Err(err).Msg("Unable to parse GetStepLog url")
	}
	u.Path = path.Join(u.Path, workspaceSlug, repoSlug, "pipelines", pipelineUUID, "steps", stepUUID, "log")

	res, err := a.Client.R().
		Get(u.String())

	return res.Bytes(), res, err
}

// https://developer.atlassian.com/cloud/bitbucket/rest/api-group-downloads/#api-repositories-workspace-repo-slug-downloads-get
func (a BitBucketApiClient) ListDownloadArtifacts(nextPageUrl string, workspaceSlug string, repoSlug string) ([]DownloadArtifact, string, *resty.Response, error) {
	reqUrl := ""
	if nextPageUrl != "" {
		reqUrl = nextPageUrl
	} else {
		u, err := url.Parse("https://api.bitbucket.org/2.0/repositories/")
		if err != nil {
			log.Fatal().Err(err).Msg("Unable to parse ListDownloadArtifacts url")
		}
		u.Path = path.Join(u.Path, workspaceSlug, repoSlug, "downloads")
		reqUrl = u.String()
	}

	resp := &PaginatedResponse[DownloadArtifact]{}
	res, err := a.Client.R().
		SetResult(resp).
		Get(reqUrl)

	return resp.Values, resp.Next, res, err
}

func (a BitBucketApiClient) GetDownloadArtifact(url string) []byte {
	res, err := a.Client.R().Get(url)
	if err != nil {
		log.Error().Err(err).Str("url", url).Msg("Failed downloading Download Artifact")
		return []byte{}
	}

	bodyBytes, err := io.ReadAll(res.Body)
	if err != nil {
		log.Error().Err(err).Str("url", url).Msg("Failed reading Download Artifact response")
		return []byte{}
	}

	return bodyBytes
}

// Internal API: https://bitbucket.org/!api/internal/repositories/{workspace}/{repo}/pipelines/{buildNumber}/artifacts
func (a BitBucketApiClient) ListPipelineArtifacts(nextPageUrl string, workspaceSlug string, repoSlug string, buildNumber int) ([]Artifact, string, *resty.Response, error) {
	reqUrl := ""
	if nextPageUrl != "" {
		reqUrl = nextPageUrl
	} else {
		u, err := url.Parse("https://bitbucket.org/!api/internal/repositories/")
		if err != nil {
			log.Fatal().Err(err).Msg("Unable to parse ListPipelineArtifacst url")
		}
		u.Path = path.Join(u.Path, workspaceSlug, repoSlug, "pipelines", strconv.Itoa(buildNumber), "artifacts")
		reqUrl = u.String()
	}

	resp := &PaginatedResponse[Artifact]{}
	res, err := a.Client.R().
		SetResult(resp).
		Get(reqUrl)

	return resp.Values, resp.Next, res, err
}

// Internal API: https://bitbucket.org/!api/internal/repositories/{workspace}/{repo}/pipelines/{buildId}/artifacts/{ArtifactUUID}/content
func (a BitBucketApiClient) GetPipelineArtifact(workspaceSlug string, repoSlug string, buildNumber int, artifactUUID string) []byte {

	u, err := url.Parse("https://bitbucket.org/!api/internal/repositories/")
	if err != nil {
		log.Fatal().Err(err).Msg("Unable to parse GetPipelineArtifact url")
	}
	u.Path = path.Join(u.Path, workspaceSlug, repoSlug, "pipelines", strconv.Itoa(buildNumber), "artifacts", artifactUUID, "content")

	res, err := a.Client.R().Get(u.String())
	if err != nil {
		log.Error().Err(err).Str("url", u.String()).Msg("Failed downloading pipeline artifact")
		return []byte{}
	}

	bodyBytes, err := io.ReadAll(res.Body)
	if err != nil {
		log.Error().Err(err).Str("url", u.String()).Msg("Failed reading pipeline artifact response")
		return []byte{}
	}

	return bodyBytes
}

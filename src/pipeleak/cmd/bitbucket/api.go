package bitbucket

import (
	"io"
	"net/http"
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

func NewClient(username string, password string) BitBucketApiClient {
	bbClient := BitBucketApiClient{Client: *resty.New().SetBasicAuth(username, password).SetRedirectPolicy(resty.FlexibleRedirectPolicy(5))}
	bbClient.Client.AddResponseMiddleware(func(c *resty.Client, res *resty.Response) error {
		// rateLimit := res.Header().Get("X-RateLimit-Limit")
		// resource := res.Header().Get("X-RateLimit-Resource")
		// nearLimit := res.Header().Get("X-RateLimit-NearLimit")

		//log.Info().Any("asdf", res.Header()).Str("rateLimit", rateLimit).Str("resource", resource).Str("nearLimit", nearLimit).Msg("Rate Limiter Status")
		// perform logic here

		// cascade error downstream
		// return errors.New("hey error occurred")

		return nil
	})
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
	c := a.Client.R().SetResult(resp)
	// only set it initially after that the next url does use the after query param for paging
	if nextPageUrl == "" {
		c = c.SetQueryParam("after", after.Format(time.RFC3339))
	}
	res, err := c.Get(reqUrl)

	return resp.Values, resp.Next, res, err
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
		EnableTrace().
		Get(u.String())

	return res.Bytes(), res, err
}

// INTERNAL API: https://bitbucket.org/!api/internal/repositories/jfrtest/secrets/pipelines/4/artifacts
func (a BitBucketApiClient) ListArtifacts(nextPageUrl string, workspaceSlug string, repoSlug string, pipelineId int) ([]Artifact, string, *resty.Response, error) {
	reqUrl := ""
	if nextPageUrl != "" {
		reqUrl = nextPageUrl
	} else {
		u, err := url.Parse("https://bitbucket.org/!api/internal/repositories/")
		if err != nil {
			log.Fatal().Err(err).Msg("Unable to parse LisArtifacts url")
		}
		u.Path = path.Join(u.Path, workspaceSlug, repoSlug, "pipelines", strconv.Itoa(pipelineId), "artifacts")
		reqUrl = u.String()
	}

	log.Trace().Str("url", reqUrl).Msg("Fetch artifact url")

	resp := &PaginatedResponse[Artifact]{}
	res, err := a.Client.R().
		SetResult(resp).
		SetCookie(&http.Cookie{Name: "cloud.session.token", Value: "REPLACE_MEE"}).
		Get(reqUrl)

	defer res.Body.Close()

	bodyBytes, err := io.ReadAll(res.Body)
	if err != nil {
		log.Fatal()
	}
	bodyString := string(bodyBytes)
	log.Trace().Str("resp", bodyString).Int("code", res.StatusCode()).Any("parsed", resp.Values).Msg("art repsonse")

	return resp.Values, resp.Next, res, err
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

// https://developer.atlassian.com/cloud/bitbucket/rest/api-group-downloads/#api-repositories-workspace-repo-slug-downloads-filename-get

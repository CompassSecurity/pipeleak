package bitbucket

import (
	"net/url"
	"path"

	"github.com/rs/zerolog/log"

	"resty.dev/v3"
)

// Docs: https://developer.atlassian.com/cloud/bitbucket/rest/intro/

type BitBucketApiClient struct {
	Client resty.Client
}

func NewClient(username string, password string) BitBucketApiClient {
	return BitBucketApiClient{Client: *resty.New().SetBasicAuth(username, password)}
}

// https://developer.atlassian.com/cloud/bitbucket/rest/api-group-workspaces/#api-workspaces-get
func (a BitBucketApiClient) ListOwnedWorkspaces() ([]Workspace, *resty.Response, error) {
	resp := &PaginatedResponse[Workspace]{}
	res, err := a.Client.R().
		EnableTrace().
		SetResult(resp).
		Get("https://api.bitbucket.org/2.0/workspaces")

	return resp.Values, res, err
}

// https://developer.atlassian.com/cloud/bitbucket/rest/api-group-repositories/#api-repositories-workspace-get
func (a BitBucketApiClient) ListWorkspaceRepositoires(workspaceSlug string) ([]Repository, *resty.Response, error) {
	u, err := url.Parse("https://api.bitbucket.org/2.0/repositories/")
	if err != nil {
		log.Fatal().Err(err).Msg("Unable to parse ListWorkspaceRepositoires url")
	}
	u.Path = path.Join(u.Path, workspaceSlug)

	resp := &PaginatedResponse[Repository]{}
	res, err := a.Client.R().
		EnableTrace().
		SetResult(resp).
		Get(u.String())

	return resp.Values, res, err
}

// https://developer.atlassian.com/cloud/bitbucket/rest/api-group-pipelines/#api-repositories-workspace-repo-slug-pipelines-get
func (a BitBucketApiClient) ListRepositoryPipelines(workspaceSlug string, repoSlug string) ([]Pipeline, *resty.Response, error) {
	u, err := url.Parse("https://api.bitbucket.org/2.0/repositories/")
	if err != nil {
		log.Fatal().Err(err).Msg("Unable to parse ListRepositoryPipelines url")
	}
	u.Path = path.Join(u.Path, workspaceSlug, repoSlug, "pipelines")

	resp := &PaginatedResponse[Pipeline]{}
	res, err := a.Client.R().
		EnableTrace().
		SetResult(resp).
		Get(u.String())

	return resp.Values, res, err
}

// https://developer.atlassian.com/cloud/bitbucket/rest/api-group-pipelines/#api-repositories-workspace-repo-slug-pipelines-pipeline-uuid-steps-get
func (a BitBucketApiClient) ListPipelineSteps(workspaceSlug string, repoSlug string, pipelineUUID string) ([]PipelineStep, *resty.Response, error) {
	u, err := url.Parse("https://api.bitbucket.org/2.0/repositories/")
	if err != nil {
		log.Fatal().Err(err).Msg("Unable to parse ListPipelineSteps url")
	}
	u.Path = path.Join(u.Path, workspaceSlug, repoSlug, "pipelines", pipelineUUID, "steps")

	resp := &PaginatedResponse[PipelineStep]{}
	res, err := a.Client.R().
		EnableTrace().
		SetResult(resp).
		Get(u.String())

	return resp.Values, res, err
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

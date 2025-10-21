package gitea

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"

	"code.gitea.io/sdk/gitea"
	"github.com/hashicorp/go-retryablehttp"
	"github.com/rs/zerolog/log"
)

type GiteaScanOptions struct {
	Token                  string
	GiteaURL               string
	Artifacts              bool
	Verbose                bool
	ConfidenceFilter       []string
	MaxScanGoRoutines      int
	TruffleHogVerification bool
	Owned                  bool
	Organization           string
	Repository             string
	Cookie                 string
	RunsLimit              int
	Context                context.Context
	Client                 *gitea.Client
	HttpClient             *retryablehttp.Client
}

type AuthTransport struct {
	Base  http.RoundTripper
	Token string
}

func (t *AuthTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	req2 := req.Clone(req.Context())
	req2.Header.Set("Authorization", "token "+t.Token)
	return t.Base.RoundTrip(req2)
}

type ActionWorkflowRun struct {
	ID         int64  `json:"id"`
	Name       string `json:"name"`
	Title      string `json:"title"`
	Status     string `json:"status"`
	Event      string `json:"event"`
	CreatedAt  string `json:"created_at"`
	UpdatedAt  string `json:"updated_at"`
	RunNumber  int64  `json:"run_number"`
	WorkflowID string `json:"workflow_id"`
	HeadBranch string `json:"head_branch"`
	HeadSha    string `json:"head_sha"`
	URL        string `json:"url"`
	HTMLURL    string `json:"html_url"`
}

type ActionWorkflowRunsResponse struct {
	TotalCount   int64               `json:"total_count"`
	WorkflowRuns []ActionWorkflowRun `json:"workflow_runs"`
}

type ActionArtifact struct {
	ID                 int64  `json:"id"`
	Name               string `json:"name"`
	Size               int64  `json:"size"`
	CreatedAt          string `json:"created_at"`
	ExpiredAt          string `json:"expired_at"`
	WorkflowRunID      int64  `json:"workflow_run_id"`
	ArchiveDownloadURL string `json:"archive_download_url"`
}

type ActionArtifactsResponse struct {
	TotalCount int64            `json:"total_count"`
	Artifacts  []ActionArtifact `json:"artifacts"`
}

type ActionJob struct {
	ID          int64  `json:"id"`
	Name        string `json:"name"`
	Status      string `json:"status"`
	Conclusion  string `json:"conclusion"`
	StartedAt   string `json:"started_at"`
	CompletedAt string `json:"completed_at"`
	RunID       int64  `json:"run_id"`
	TaskID      int64  `json:"task_id"`
}

type ActionJobsResponse struct {
	TotalCount int64       `json:"total_count"`
	Jobs       []ActionJob `json:"jobs"`
}

func listWorkflowRuns(client *gitea.Client, repo *gitea.Repository) ([]ActionWorkflowRun, error) {
	// Gitea Actions API: GET /repos/{owner}/{repo}/actions/runs
	// Note: This endpoint may not be available in all Gitea versions
	// The SDK doesn't have this method yet, so we make a direct API call

	var allRuns []ActionWorkflowRun
	page := 1
	limit := 50

	for {
		link, err := url.Parse(scanOptions.GiteaURL)
		if err != nil {
			return nil, err
		}
		link.Path = fmt.Sprintf("/api/v1/repos/%s/%s/actions/runs", repo.Owner.UserName, repo.Name)

		q := link.Query()
		q.Set("page", strconv.Itoa(page))
		q.Set("limit", strconv.Itoa(limit))
		link.RawQuery = q.Encode()

		resp, err := scanOptions.HttpClient.Get(link.String())
		if err != nil {
			return nil, err
		}

		if resp.StatusCode == 404 {
			// Actions not enabled or endpoint not available
			resp.Body.Close()
			return allRuns, nil
		}

		if resp.StatusCode != 200 {
			resp.Body.Close()
			return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
		}

		body, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			return nil, err
		}

		var runsResp ActionWorkflowRunsResponse
		if err := json.Unmarshal(body, &runsResp); err != nil {
			// Try parsing as array directly (older API format)
			var runs []ActionWorkflowRun
			if err2 := json.Unmarshal(body, &runs); err2 != nil {
				return nil, fmt.Errorf("failed to parse workflow runs: %w", err)
			}

			allRuns = append(allRuns, runs...)

			// Check if we've reached the runs limit
			if scanOptions.RunsLimit > 0 && len(allRuns) >= scanOptions.RunsLimit {
				log.Debug().Str("repo", repo.FullName).Int("limit", scanOptions.RunsLimit).Msg("Reached runs limit, stopping pagination")
				return allRuns[:scanOptions.RunsLimit], nil
			}

			if len(runs) < limit {
				break
			}
		} else {
			allRuns = append(allRuns, runsResp.WorkflowRuns...)

			// Check if we've reached the runs limit
			if scanOptions.RunsLimit > 0 && len(allRuns) >= scanOptions.RunsLimit {
				log.Debug().Str("repo", repo.FullName).Int("limit", scanOptions.RunsLimit).Msg("Reached runs limit, stopping pagination")
				return allRuns[:scanOptions.RunsLimit], nil
			}

			if len(allRuns) >= int(runsResp.TotalCount) || len(runsResp.WorkflowRuns) < limit {
				break
			}
		}

		page++
	}

	return allRuns, nil
}

func scanWorkflowRunLogs(client *gitea.Client, repo *gitea.Repository, run ActionWorkflowRun) {
	jobs, err := listWorkflowJobs(client, repo, run)
	if err != nil {
		log.Error().Err(err).Str("url", run.HTMLURL).Msg("failed to list workflow jobs")
		return
	}

	if len(jobs) == 0 {
		log.Debug().Str("url", run.HTMLURL).Msg("No jobs found for workflow run")
		return
	}

	for _, job := range jobs {
		scanJobLogs(client, repo, run, job)
	}
}

func listWorkflowJobs(client *gitea.Client, repo *gitea.Repository, run ActionWorkflowRun) ([]ActionJob, error) {
	// Gitea Actions API: GET /repos/{owner}/{repo}/actions/runs/{run}/jobs

	var allJobs []ActionJob
	page := 1
	limit := 50

	for {
		link, err := url.Parse(scanOptions.GiteaURL)
		if err != nil {
			return nil, err
		}
		link.Path = fmt.Sprintf("/api/v1/repos/%s/%s/actions/runs/%d/jobs", repo.Owner.UserName, repo.Name, run.ID)

		q := link.Query()
		q.Set("page", strconv.Itoa(page))
		q.Set("limit", strconv.Itoa(limit))
		link.RawQuery = q.Encode()

		resp, err := scanOptions.HttpClient.Get(link.String())
		if err != nil {
			return nil, err
		}

		if resp.StatusCode == 404 {
			resp.Body.Close()
			return allJobs, nil
		}

		if resp.StatusCode != 200 {
			resp.Body.Close()
			return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
		}

		body, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			return nil, err
		}

		var jobsResp ActionJobsResponse
		if err := json.Unmarshal(body, &jobsResp); err != nil {
			var jobs []ActionJob
			if err2 := json.Unmarshal(body, &jobs); err2 != nil {
				return nil, fmt.Errorf("failed to parse jobs: %w", err)
			}
			allJobs = append(allJobs, jobs...)
			if len(jobs) < limit {
				break
			}
		} else {
			allJobs = append(allJobs, jobsResp.Jobs...)
			if len(allJobs) >= int(jobsResp.TotalCount) || len(jobsResp.Jobs) < limit {
				break
			}
		}

		page++
	}

	return allJobs, nil
}

func scanJobLogs(client *gitea.Client, repo *gitea.Repository, run ActionWorkflowRun, job ActionJob) {
	// Gitea Actions API: GET /repos/{owner}/{repo}/actions/jobs/{job_id}/logs
	jobURL := fmt.Sprintf("%s/%s/actions/runs/%d/jobs/%d", scanOptions.GiteaURL, repo.FullName, run.ID, job.ID)
	urlStr, err := buildAPIURL(repo, "/actions/jobs/%d/logs", job.ID)
	if err != nil {
		log.Error().Err(err).Int64("job_id", job.ID).Str("url", jobURL).Msg("failed to build URL")
		return
	}

	resp, err := makeHTTPRequest(urlStr)
	if err != nil {
		log.Error().Err(err).Int64("job_id", job.ID).Str("url", jobURL).Msg("failed to download logs")
		return
	}

	ctx := logContext{Repo: repo.FullName, RunID: run.ID, JobID: job.ID}

	if resp.StatusCode == 404 {
		log.Debug().Int64("job_id", job.ID).Str("url", jobURL).Msg("Logs not found or expired")
		return
	}

	if err := checkHTTPStatus(resp.StatusCode, "download logs"); err != nil {
		logHTTPError(resp.StatusCode, "download logs", ctx)
		return
	}

	scanLogs(resp.Body, repo, run, job.ID, job.Name)
}

func scanWorkflowArtifacts(client *gitea.Client, repo *gitea.Repository, run ActionWorkflowRun) {
	artifacts, err := listArtifacts(repo, run)
	if err != nil {
		log.Error().Err(err).Str("url", run.HTMLURL).Msg("failed to fetch artifacts")
		return
	}

	if len(artifacts) == 0 {
		log.Debug().Str("url", run.HTMLURL).Msg("No artifacts found")
		return
	}

	log.Debug().Str("url", run.HTMLURL).Int("count", len(artifacts)).Msg("Found artifacts")

	for _, artifact := range artifacts {
		log.Debug().
			Str("url", run.HTMLURL).
			Str("artifact", artifact.Name).
			Msg("Downloading and scanning artifact")

		downloadAndScanArtifact(client, repo, run, artifact)
	}
}

func listArtifacts(repo *gitea.Repository, run ActionWorkflowRun) ([]ActionArtifact, error) {
	// Gitea Actions API: GET /repos/{owner}/{repo}/actions/runs/{run_id}/artifacts
	var allArtifacts []ActionArtifact
	page := 1
	limit := 50

	for {
		link, err := url.Parse(scanOptions.GiteaURL)
		if err != nil {
			return nil, err
		}
		link.Path = fmt.Sprintf("/api/v1/repos/%s/%s/actions/runs/%d/artifacts", repo.Owner.UserName, repo.Name, run.ID)

		q := link.Query()
		q.Set("page", strconv.Itoa(page))
		q.Set("limit", strconv.Itoa(limit))
		link.RawQuery = q.Encode()

		resp, err := scanOptions.HttpClient.Get(link.String())
		if err != nil {
			return nil, err
		}

		if resp.StatusCode == 404 {
			resp.Body.Close()
			return allArtifacts, nil
		}

		if resp.StatusCode != 200 {
			resp.Body.Close()
			return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
		}

		body, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			return nil, err
		}

		var artifactsResp ActionArtifactsResponse
		if err := json.Unmarshal(body, &artifactsResp); err != nil {
			var artifacts []ActionArtifact
			if err2 := json.Unmarshal(body, &artifacts); err2 != nil {
				return nil, fmt.Errorf("failed to parse artifacts: %w", err)
			}
			allArtifacts = append(allArtifacts, artifacts...)
			if len(artifacts) < limit {
				break
			}
		} else {
			allArtifacts = append(allArtifacts, artifactsResp.Artifacts...)
			if len(allArtifacts) >= int(artifactsResp.TotalCount) || len(artifactsResp.Artifacts) < limit {
				break
			}
		}

		page++
	}

	return allArtifacts, nil
}

func downloadAndScanArtifact(client *gitea.Client, repo *gitea.Repository, run ActionWorkflowRun, artifact ActionArtifact) {
	// Gitea Actions API: GET /repos/{owner}/{repo}/actions/artifacts/{artifact_id}/zip
	// This endpoint returns a 302 redirect to the actual blob URL
	urlStr, err := buildAPIURL(repo, "/actions/artifacts/%d/zip", artifact.ID)
	if err != nil {
		log.Error().Err(err).Str("repo", repo.FullName).Int64("artifact_id", artifact.ID).Str("url", run.HTMLURL).Msg("failed to build URL")
		return
	}

	resp, err := makeHTTPRequest(urlStr)
	if err != nil {
		log.Error().Err(err).Str("repo", repo.FullName).Int64("artifact_id", artifact.ID).Str("url", run.HTMLURL).Msg("failed to download artifact")
		return
	}

	if resp.StatusCode == 404 || resp.StatusCode == 410 {
		log.Debug().Str("repo", repo.FullName).Int64("artifact_id", artifact.ID).Str("url", run.HTMLURL).Msg("Artifact expired or not found")
		return
	}

	if resp.StatusCode != 200 && resp.StatusCode != 302 {
		log.Error().Int("status", resp.StatusCode).Str("repo", repo.FullName).Int64("artifact_id", artifact.ID).Str("url", run.HTMLURL).Msg("failed to download artifact")
		return
	}

	processZipArtifact(resp.Body, repo, run, artifact.Name)
}

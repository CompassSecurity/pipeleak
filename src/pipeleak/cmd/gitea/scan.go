package gitea

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"

	"code.gitea.io/sdk/gitea"
	"github.com/CompassSecurity/pipeleak/helper"
	"github.com/CompassSecurity/pipeleak/scanner"
	"github.com/h2non/filetype"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"github.com/wandb/parallel"
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
	Context                context.Context
	Client                 *gitea.Client
	HttpClient             *http.Client
}

var scanOptions = GiteaScanOptions{}

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

func NewScanCmd() *cobra.Command {
	scanCmd := &cobra.Command{
		Use:   "scan",
		Short: "Scan Gitea Actions",
		Long:  `Scan Gitea Actions workflow runs and artifacts for secrets`,
		Example: `
# Scan all accessible repositories and their artifacts
pipeleak gitea scan --token gitea_token_xxxxx --gitea https://gitea.example.com --artifacts

# Scan without downloading artifacts
pipeleak gitea scan --token gitea_token_xxxxx --gitea https://gitea.example.com

# Scan only repositories owned by the user
pipeleak gitea scan --token gitea_token_xxxxx --gitea https://gitea.example.com --owned

# Scan all repositories of a specific organization
pipeleak gitea scan --token gitea_token_xxxxx --gitea https://gitea.example.com --organization my-org
		`,
		Run: Scan,
	}

	scanCmd.Flags().StringVarP(&scanOptions.Token, "token", "t", "", "Gitea personal access token")
	err := scanCmd.MarkFlagRequired("token")
	if err != nil {
		log.Fatal().Msg("Unable to require token flag")
	}

	scanCmd.Flags().StringVarP(&scanOptions.GiteaURL, "gitea", "g", "https://gitea.com", "Base Gitea URL (e.g. https://gitea.example.com)")

	scanCmd.Flags().BoolVarP(&scanOptions.Artifacts, "artifacts", "a", false, "Download and scan workflow artifacts")
	scanCmd.Flags().BoolVarP(&scanOptions.Owned, "owned", "o", false, "Scan only repositories owned by the user")
	scanCmd.Flags().StringVarP(&scanOptions.Organization, "organization", "", "", "Scan all repositories of a specific organization")
	scanCmd.Flags().StringSliceVarP(&scanOptions.ConfidenceFilter, "confidence", "", []string{}, "Filter for confidence level, separate by comma if multiple. See documentation for more info.")
	scanCmd.PersistentFlags().IntVarP(&scanOptions.MaxScanGoRoutines, "threads", "", 4, "Nr of threads used to scan")
	scanCmd.PersistentFlags().BoolVarP(&scanOptions.TruffleHogVerification, "truffleHogVerification", "", true, "Enable TruffleHog credential verification to actively test found credentials and only report verified ones (enabled by default, disable with --truffleHogVerification=false)")
	scanCmd.PersistentFlags().BoolVarP(&scanOptions.Verbose, "verbose", "v", false, "Verbose logging")

	return scanCmd
}

func Scan(cmd *cobra.Command, args []string) {
	helper.SetLogLevel(scanOptions.Verbose)
	go helper.ShortcutListeners(scanStatus)

	_, err := url.ParseRequestURI(scanOptions.GiteaURL)
	if err != nil {
		log.Fatal().Err(err).Msg("The provided Gitea URL is not a valid URL")
	}

	scanOptions.Context = context.Background()
	scanOptions.Client, err = gitea.NewClient(scanOptions.GiteaURL, gitea.SetToken(scanOptions.Token))
	if err != nil {
		log.Fatal().Err(err).Msg("Failed creating Gitea client")
	}

	scanOptions.HttpClient = helper.GetPipeleakHTTPClient()

	scanner.InitRules(scanOptions.ConfidenceFilter)
	if !scanOptions.TruffleHogVerification {
		log.Info().Msg("TruffleHog verification is disabled")
	}

	scanRepositories(scanOptions.Client)
	log.Info().Msg("Scan Finished, Bye Bye 🏳️‍🌈🔥")
}

func scanStatus() *zerolog.Event {
	return log.Info().Str("status", "scanning... ✨✨ nothing more yet ✨✨")
}

func scanRepositories(client *gitea.Client) {
	if scanOptions.Organization != "" {
		log.Info().Str("organization", scanOptions.Organization).Msg("Scanning organization repositories")
		scanOrganizationRepositories(client, scanOptions.Organization)
	} else if scanOptions.Owned {
		log.Info().Msg("Scanning user owned repositories")
		scanOwnedRepositories(client)
	} else {
		log.Info().Msg("Scanning all accessible repositories")
		scanAllRepositories(client)
	}

	log.Info().Msg("Completed scanning all accessible repositories")
}

func scanAllRepositories(client *gitea.Client) {
	opt := gitea.ListReposOptions{
		ListOptions: gitea.ListOptions{
			Page:     1,
			PageSize: 50,
		},
	}

	for {
		repos, resp, err := client.ListMyRepos(opt)
		if err != nil {
			log.Error().Err(err).Int("page", opt.Page).Msg("failed to list repos")
			break
		}

		if len(repos) == 0 {
			break
		}

		log.Info().Int("count", len(repos)).Int("page", opt.Page).Msg("Processing repositories page")

		for _, repo := range repos {
			log.Debug().Str("repo", repo.FullName).Str("url", repo.HTMLURL).Msg("Scanning repository")
			scanRepository(client, repo)
		}

		if resp == nil || resp.NextPage == 0 {
			break
		}

		opt.Page = resp.NextPage
	}
}

func scanOwnedRepositories(client *gitea.Client) {
	// Get current user info
	user, _, err := client.GetMyUserInfo()
	if err != nil {
		log.Error().Err(err).Msg("failed to get user info")
		return
	}

	opt := gitea.ListReposOptions{
		ListOptions: gitea.ListOptions{
			Page:     1,
			PageSize: 50,
		},
	}

	for {
		repos, resp, err := client.ListMyRepos(opt)
		if err != nil {
			log.Error().Err(err).Int("page", opt.Page).Msg("failed to list repos")
			break
		}

		if len(repos) == 0 {
			break
		}

		log.Info().Int("count", len(repos)).Int("page", opt.Page).Msg("Processing repositories page")

		for _, repo := range repos {
			// Filter to only include repos owned by the current user
			if repo.Owner != nil && repo.Owner.ID == user.ID {
				log.Debug().Str("repo", repo.FullName).Str("url", repo.HTMLURL).Msg("Scanning repository")
				scanRepository(client, repo)
			}
		}

		if resp == nil || resp.NextPage == 0 {
			break
		}

		opt.Page = resp.NextPage
	}
}

func scanOrganizationRepositories(client *gitea.Client, orgName string) {
	opt := gitea.ListOrgReposOptions{
		ListOptions: gitea.ListOptions{
			Page:     1,
			PageSize: 50,
		},
	}

	for {
		repos, resp, err := client.ListOrgRepos(orgName, opt)
		if err != nil {
			log.Error().Err(err).Str("organization", orgName).Int("page", opt.Page).Msg("failed to list organization repos")
			break
		}

		if len(repos) == 0 {
			break
		}

		log.Info().Int("count", len(repos)).Int("page", opt.Page).Str("organization", orgName).Msg("Processing organization repositories page")

		for _, repo := range repos {
			log.Debug().Str("repo", repo.FullName).Str("url", repo.HTMLURL).Msg("Scanning repository")
			scanRepository(client, repo)
		}

		if resp == nil || resp.NextPage == 0 {
			break
		}

		opt.Page = resp.NextPage
	}
}

func scanRepository(client *gitea.Client, repo *gitea.Repository) {
	workflowRuns, err := listWorkflowRuns(client, repo)
	if err != nil {
		log.Error().Err(err).Str("repo", repo.FullName).Msg("failed to list workflow runs")
		return
	}

	if len(workflowRuns) == 0 {
		log.Debug().Str("repo", repo.FullName).Msg("No workflow runs found")
		return
	}

	log.Info().Str("repo", repo.FullName).Int("runs", len(workflowRuns)).Msg("Found workflow runs")

	for _, run := range workflowRuns {
		log.Debug().
			Str("repo", repo.FullName).
			Int64("run_id", run.ID).
			Str("status", run.Status).
			Str("name", run.Name).
			Msg("scanning pipeline run")

		scanWorkflowRunLogs(client, repo, run)

		if scanOptions.Artifacts {
			scanWorkflowArtifacts(client, repo, run)
		}
	}
}

func listWorkflowRuns(client *gitea.Client, repo *gitea.Repository) ([]ActionWorkflowRun, error) {
	// Gitea Actions API: GET /repos/{owner}/{repo}/actions/runs
	// Note: This endpoint may not be available in all Gitea versions
	// The SDK doesn't have this method yet, so we make a direct API call

	var allRuns []ActionWorkflowRun
	page := 1
	limit := 50

	for {
		apiPath := fmt.Sprintf("/api/v1/repos/%s/%s/actions/runs", repo.Owner.UserName, repo.Name)
		link, err := url.Parse(apiPath)
		if err != nil {
			return nil, err
		}

		q := link.Query()
		q.Set("page", fmt.Sprintf("%d", page))
		q.Set("limit", fmt.Sprintf("%d", limit))
		link.RawQuery = q.Encode()

		fullURL := scanOptions.GiteaURL + link.String()

		req, err := http.NewRequestWithContext(scanOptions.Context, "GET", fullURL, nil)
		if err != nil {
			return nil, err
		}

		req.Header.Set("Authorization", "token "+scanOptions.Token)
		req.Header.Set("Content-Type", "application/json")

		resp, err := scanOptions.HttpClient.Do(req)
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
			if len(runs) < limit {
				break
			}
		} else {
			allRuns = append(allRuns, runsResp.WorkflowRuns...)
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
		log.Error().Err(err).Str("repo", repo.FullName).Int64("run_id", run.ID).Msg("failed to list workflow jobs")
		return
	}

	if len(jobs) == 0 {
		log.Debug().Str("repo", repo.FullName).Int64("run_id", run.ID).Msg("No jobs found for workflow run")
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
		apiPath := fmt.Sprintf("/api/v1/repos/%s/%s/actions/runs/%d/jobs", repo.Owner.UserName, repo.Name, run.ID)
		link, err := url.Parse(apiPath)
		if err != nil {
			return nil, err
		}

		q := link.Query()
		q.Set("page", fmt.Sprintf("%d", page))
		q.Set("limit", fmt.Sprintf("%d", limit))
		link.RawQuery = q.Encode()

		fullURL := scanOptions.GiteaURL + link.String()

		req, err := http.NewRequestWithContext(scanOptions.Context, "GET", fullURL, nil)
		if err != nil {
			return nil, err
		}

		req.Header.Set("Authorization", "token "+scanOptions.Token)
		req.Header.Set("Content-Type", "application/json")

		resp, err := scanOptions.HttpClient.Do(req)
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
	apiPath := fmt.Sprintf("/api/v1/repos/%s/%s/actions/jobs/%d/logs", repo.Owner.UserName, repo.Name, job.ID)
	link, err := url.Parse(apiPath)
	if err != nil {
		log.Error().Err(err).Str("repo", repo.FullName).Int64("run_id", run.ID).Int64("job_id", job.ID).Msg("failed to parse URL")
		return
	}

	fullURL := scanOptions.GiteaURL + link.String()

	req, err := http.NewRequestWithContext(scanOptions.Context, "GET", fullURL, nil)
	if err != nil {
		log.Error().Err(err).Str("repo", repo.FullName).Int64("run_id", run.ID).Int64("job_id", job.ID).Msg("failed to create request for logs")
		return
	}

	req.Header.Set("Authorization", "token "+scanOptions.Token)

	resp, err := scanOptions.HttpClient.Do(req)
	if err != nil {
		log.Error().Err(err).Str("repo", repo.FullName).Int64("run_id", run.ID).Int64("job_id", job.ID).Msg("failed to download logs")
		return
	}

	defer resp.Body.Close()

	if resp.StatusCode == 404 {
		log.Debug().Str("repo", repo.FullName).Int64("run_id", run.ID).Int64("job_id", job.ID).Msg("Logs not found or expired")
		return
	}

	if resp.StatusCode != 200 {
		log.Error().Int("status", resp.StatusCode).Str("repo", repo.FullName).Int64("run_id", run.ID).Int64("job_id", job.ID).Msg("failed to download logs")
		return
	}

	logBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Error().Err(err).Str("repo", repo.FullName).Int64("run_id", run.ID).Int64("job_id", job.ID).Msg("failed to read log bytes")
		return
	}

	findings, err := scanner.DetectHits(logBytes, scanOptions.MaxScanGoRoutines, scanOptions.TruffleHogVerification)
	if err != nil {
		log.Debug().Err(err).Str("repo", repo.FullName).Int64("run_id", run.ID).Int64("job_id", job.ID).Msg("Failed detecting secrets in logs")
		return
	}

	for _, finding := range findings {
		log.Warn().
			Str("confidence", finding.Pattern.Pattern.Confidence).
			Str("ruleName", finding.Pattern.Pattern.Name).
			Str("value", finding.Text).
			Str("repo", repo.FullName).
			Int64("run_id", run.ID).
			Int64("job_id", job.ID).
			Str("job_name", job.Name).
			Str("url", run.HTMLURL).
			Msg("HIT")
	}
}

func scanWorkflowArtifacts(client *gitea.Client, repo *gitea.Repository, run ActionWorkflowRun) {
	artifacts, err := listArtifacts(repo, run)
	if err != nil {
		log.Error().Err(err).Str("repo", repo.FullName).Int64("run_id", run.ID).Msg("failed to fetch artifacts")
		return
	}

	if len(artifacts) == 0 {
		log.Debug().Str("repo", repo.FullName).Int64("run_id", run.ID).Msg("No artifacts found")
		return
	}

	log.Debug().Str("repo", repo.FullName).Int64("run_id", run.ID).Int("count", len(artifacts)).Msg("Found artifacts")

	for _, artifact := range artifacts {
		log.Debug().
			Str("repo", repo.FullName).
			Int64("run_id", run.ID).
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
		apiPath := fmt.Sprintf("/api/v1/repos/%s/%s/actions/runs/%d/artifacts", repo.Owner.UserName, repo.Name, run.ID)
		link, err := url.Parse(apiPath)
		if err != nil {
			return nil, err
		}

		q := link.Query()
		q.Set("page", fmt.Sprintf("%d", page))
		q.Set("limit", fmt.Sprintf("%d", limit))
		link.RawQuery = q.Encode()

		fullURL := scanOptions.GiteaURL + link.String()

		req, err := http.NewRequestWithContext(scanOptions.Context, "GET", fullURL, nil)
		if err != nil {
			return nil, err
		}

		req.Header.Set("Authorization", "token "+scanOptions.Token)
		req.Header.Set("Content-Type", "application/json")

		resp, err := scanOptions.HttpClient.Do(req)
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
	apiPath := fmt.Sprintf("/api/v1/repos/%s/%s/actions/artifacts/%d/zip", repo.Owner.UserName, repo.Name, artifact.ID)
	link, err := url.Parse(apiPath)
	if err != nil {
		log.Error().Err(err).Str("repo", repo.FullName).Int64("artifact_id", artifact.ID).Msg("failed to parse URL")
		return
	}

	fullURL := scanOptions.GiteaURL + link.String()

	req, err := http.NewRequestWithContext(scanOptions.Context, "GET", fullURL, nil)
	if err != nil {
		log.Error().Err(err).Str("repo", repo.FullName).Int64("artifact_id", artifact.ID).Msg("failed to create artifact download request")
		return
	}

	req.Header.Set("Authorization", "token "+scanOptions.Token)

	resp, err := scanOptions.HttpClient.Do(req)
	if err != nil {
		log.Error().Err(err).Str("repo", repo.FullName).Int64("artifact_id", artifact.ID).Msg("failed to download artifact")
		return
	}

	defer resp.Body.Close()

	if resp.StatusCode == 404 || resp.StatusCode == 410 {
		log.Debug().Str("repo", repo.FullName).Int64("artifact_id", artifact.ID).Msg("Artifact expired or not found")
		return
	}

	if resp.StatusCode != 200 && resp.StatusCode != 302 {
		log.Error().Int("status", resp.StatusCode).Str("repo", repo.FullName).Int64("artifact_id", artifact.ID).Msg("failed to download artifact")
		return
	}

	artifactBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Error().Err(err).Str("repo", repo.FullName).Int64("artifact_id", artifact.ID).Msg("failed to read artifact bytes")
		return
	}

	zipReader, err := zip.NewReader(bytes.NewReader(artifactBytes), int64(len(artifactBytes)))
	if err != nil {
		log.Debug().Err(err).Str("repo", repo.FullName).Int64("artifact_id", artifact.ID).Msg("Artifact is not a zip, scanning directly")
		scanArtifactContent(artifactBytes, repo, run, artifact.Name, "")
		return
	}

	ctx := scanOptions.Context
	group := parallel.Limited(ctx, scanOptions.MaxScanGoRoutines)

	for _, file := range zipReader.File {
		group.Go(func(ctx context.Context) {
			fc, err := file.Open()
			if err != nil {
				log.Error().Err(err).Str("file", file.Name).Msg("Unable to open file in artifact zip")
				return
			}
			defer fc.Close()

			content, err := io.ReadAll(fc)
			if err != nil {
				log.Error().Err(err).Str("file", file.Name).Msg("Unable to read file in artifact zip")
				return
			}

			scanArtifactContent(content, repo, run, artifact.Name, file.Name)
		})
	}

	group.Wait()
}

func scanArtifactContent(content []byte, repo *gitea.Repository, run ActionWorkflowRun, artifactName string, fileName string) {
	kind, _ := filetype.Match(content)

	displayName := artifactName
	if fileName != "" {
		displayName = fmt.Sprintf("%s/%s", artifactName, fileName)
	}

	if filetype.IsArchive(content) {
		scanner.HandleArchiveArtifact(displayName, content, run.HTMLURL, run.Name, scanOptions.TruffleHogVerification)
	} else if kind != filetype.Unknown {
		log.Trace().Str("file", displayName).Str("type", kind.MIME.Value).Msg("Skipping unknown file type")
	} else {
		log.Info().Str("file", displayName).Str("type", kind.MIME.Value).Msg("Not an archive file type, scanning as text")
		scanner.DetectFileHits(content, run.HTMLURL, run.Name, displayName, repo.FullName, scanOptions.TruffleHogVerification)
	}
}

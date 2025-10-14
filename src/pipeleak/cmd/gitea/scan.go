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
	Context                context.Context
	Client                 *gitea.Client
	HttpClient             *http.Client
}

var scanOptions = GiteaScanOptions{}

// ActionWorkflowRun represents a Gitea Actions workflow run
type ActionWorkflowRun struct {
	ID          int64  `json:"id"`
	Name        string `json:"name"`
	Title       string `json:"title"`
	Status      string `json:"status"`
	Event       string `json:"event"`
	CreatedAt   string `json:"created_at"`
	UpdatedAt   string `json:"updated_at"`
	RunNumber   int64  `json:"run_number"`
	WorkflowID  string `json:"workflow_id"`
	HeadBranch  string `json:"head_branch"`
	HeadSha     string `json:"head_sha"`
	URL         string `json:"url"`
	HTMLURL     string `json:"html_url"`
}

// ActionWorkflowRunsResponse represents the response from listing workflow runs
type ActionWorkflowRunsResponse struct {
	TotalCount   int64               `json:"total_count"`
	WorkflowRuns []ActionWorkflowRun `json:"workflow_runs"`
}

// ActionArtifact represents a workflow run artifact
type ActionArtifact struct {
	ID                 int64  `json:"id"`
	Name               string `json:"name"`
	Size               int64  `json:"size"`
	CreatedAt          string `json:"created_at"`
	ExpiredAt          string `json:"expired_at"`
	WorkflowRunID      int64  `json:"workflow_run_id"`
	ArchiveDownloadURL string `json:"archive_download_url"`
}

// ActionArtifactsResponse represents the response from listing artifacts
type ActionArtifactsResponse struct {
	TotalCount int64            `json:"total_count"`
	Artifacts  []ActionArtifact `json:"artifacts"`
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
		`,
		Run: Scan,
	}

	scanCmd.Flags().StringVarP(&scanOptions.Token, "token", "t", "", "Gitea personal access token")
	err := scanCmd.MarkFlagRequired("token")
	if err != nil {
		log.Fatal().Msg("Unable to require token flag")
	}

	scanCmd.Flags().StringVarP(&scanOptions.GiteaURL, "gitea", "g", "", "Base Gitea URL (e.g. https://gitea.example.com)")
	err = scanCmd.MarkFlagRequired("gitea")
	if err != nil {
		log.Fatal().Msg("Unable to require gitea flag")
	}

	scanCmd.Flags().BoolVarP(&scanOptions.Artifacts, "artifacts", "a", false, "Download and scan workflow artifacts")
	scanCmd.Flags().StringSliceVarP(&scanOptions.ConfidenceFilter, "confidence", "", []string{}, "Filter for confidence level, separate by comma if multiple. See readme for more info.")
	scanCmd.PersistentFlags().IntVarP(&scanOptions.MaxScanGoRoutines, "threads", "", 4, "Nr of threads used to scan")
	scanCmd.PersistentFlags().BoolVarP(&scanOptions.TruffleHogVerification, "truffleHogVerification", "", true, "Enable the TruffleHog credential verification, will actively test the found credentials and only report those. Disable with --truffleHogVerification=false")
	scanCmd.PersistentFlags().BoolVarP(&scanOptions.Verbose, "verbose", "v", false, "Verbose logging")

	return scanCmd
}

func Scan(cmd *cobra.Command, args []string) {
	helper.SetLogLevel(scanOptions.Verbose)
	go helper.ShortcutListeners(scanStatus)

	// Validate URL
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

	log.Info().Msg("Starting Gitea Actions scan")
	scanRepositories(scanOptions.Client)
	log.Info().Msg("Scan Finished, Bye Bye 🏳️‍🌈🔥")
}

func scanStatus() *zerolog.Event {
	return log.Info().Str("status", "scanning...")
}

func scanRepositories(client *gitea.Client) {
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

	log.Info().Msg("Completed scanning all accessible repositories")
}

func scanRepository(client *gitea.Client, repo *gitea.Repository) {
	// List workflow runs for this repository
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
		log.Info().
			Str("repo", repo.FullName).
			Int64("run_id", run.ID).
			Str("status", run.Status).
			Str("name", run.Name).
			Msg("scanning pipeline run")

		// Download and scan workflow run logs
		scanWorkflowRunLogs(client, repo, run)

		// If artifacts flag is set, download and scan artifacts
		if scanOptions.Artifacts {
			scanWorkflowArtifacts(client, repo, run)
		}
	}
}

func listWorkflowRuns(client *gitea.Client, repo *gitea.Repository) ([]ActionWorkflowRun, error) {
	// Gitea Actions API: GET /repos/{owner}/{repo}/actions/runs
	// Note: This endpoint may not be available in all Gitea versions
	// The SDK doesn't have this method yet, so we make a direct API call

	apiPath := fmt.Sprintf("/api/v1/repos/%s/%s/actions/runs", repo.Owner.UserName, repo.Name)

	var allRuns []ActionWorkflowRun

	for page := 1; ; page++ {
		pageURL := fmt.Sprintf("%s%s?page=%d&limit=50", scanOptions.GiteaURL, apiPath, page)
		
		req, err := http.NewRequestWithContext(scanOptions.Context, "GET", pageURL, nil)
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
			if len(runs) < 50 {
				break
			}
		} else {
			allRuns = append(allRuns, runsResp.WorkflowRuns...)
			if len(runsResp.WorkflowRuns) < 50 {
				break
			}
		}

		// Safety check to avoid infinite loops
		if page >= 100 {
			log.Warn().Str("repo", repo.FullName).Msg("Reached maximum page limit for workflow runs")
			break
		}
	}

	return allRuns, nil
}

func scanWorkflowRunLogs(client *gitea.Client, repo *gitea.Repository, run ActionWorkflowRun) {
	// Gitea Actions API: GET /repos/{owner}/{repo}/actions/runs/{run_id}/logs
	apiPath := fmt.Sprintf("/api/v1/repos/%s/%s/actions/runs/%d/logs", repo.Owner.UserName, repo.Name, run.ID)
	fullURL := fmt.Sprintf("%s%s", scanOptions.GiteaURL, apiPath)

	req, err := http.NewRequestWithContext(scanOptions.Context, "GET", fullURL, nil)
	if err != nil {
		log.Error().Err(err).Str("repo", repo.FullName).Int64("run_id", run.ID).Msg("failed to create request for logs")
		return
	}
	req.Header.Set("Authorization", "token "+scanOptions.Token)

	resp, err := scanOptions.HttpClient.Do(req)
	if err != nil {
		log.Error().Err(err).Str("repo", repo.FullName).Int64("run_id", run.ID).Msg("failed to download logs")
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode == 404 {
		log.Debug().Str("repo", repo.FullName).Int64("run_id", run.ID).Msg("Logs not found or expired")
		return
	}

	if resp.StatusCode != 200 {
		log.Error().Int("status", resp.StatusCode).Str("repo", repo.FullName).Int64("run_id", run.ID).Msg("failed to download logs")
		return
	}

	logBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Error().Err(err).Str("repo", repo.FullName).Int64("run_id", run.ID).Msg("failed to read log bytes")
		return
	}

	// Scan the logs for secrets
	findings, err := scanner.DetectHits(logBytes, scanOptions.MaxScanGoRoutines, scanOptions.TruffleHogVerification)
	if err != nil {
		log.Debug().Err(err).Str("repo", repo.FullName).Int64("run_id", run.ID).Msg("Failed detecting secrets in logs")
		return
	}

	for _, finding := range findings {
		log.Warn().
			Str("confidence", finding.Pattern.Pattern.Confidence).
			Str("ruleName", finding.Pattern.Pattern.Name).
			Str("value", finding.Text).
			Str("repo", repo.FullName).
			Int64("run_id", run.ID).
			Str("url", run.HTMLURL).
			Msg("HIT")
	}
}

func scanWorkflowArtifacts(client *gitea.Client, repo *gitea.Repository, run ActionWorkflowRun) {
	// List artifacts for this workflow run
	artifacts, err := listArtifacts(client, repo, run)
	if err != nil {
		log.Error().Err(err).Str("repo", repo.FullName).Int64("run_id", run.ID).Msg("failed to fetch artifacts")
		return
	}

	if len(artifacts) == 0 {
		log.Debug().Str("repo", repo.FullName).Int64("run_id", run.ID).Msg("No artifacts found")
		return
	}

	log.Info().Str("repo", repo.FullName).Int64("run_id", run.ID).Int("count", len(artifacts)).Msg("Found artifacts")

	for _, artifact := range artifacts {
		log.Debug().
			Str("repo", repo.FullName).
			Int64("run_id", run.ID).
			Str("artifact", artifact.Name).
			Msg("Downloading and scanning artifact")

		downloadAndScanArtifact(client, repo, run, artifact)
	}
}

func listArtifacts(client *gitea.Client, repo *gitea.Repository, run ActionWorkflowRun) ([]ActionArtifact, error) {
	// Gitea Actions API: GET /repos/{owner}/{repo}/actions/runs/{run_id}/artifacts
	apiPath := fmt.Sprintf("/api/v1/repos/%s/%s/actions/runs/%d/artifacts", repo.Owner.UserName, repo.Name, run.ID)
	fullURL := fmt.Sprintf("%s%s", scanOptions.GiteaURL, apiPath)

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
	defer resp.Body.Close()

	if resp.StatusCode == 404 {
		// No artifacts or endpoint not available
		return []ActionArtifact{}, nil
	}

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var artifactsResp ActionArtifactsResponse
	if err := json.Unmarshal(body, &artifactsResp); err != nil {
		// Try parsing as array directly
		var artifacts []ActionArtifact
		if err2 := json.Unmarshal(body, &artifacts); err2 != nil {
			return nil, fmt.Errorf("failed to parse artifacts: %w", err)
		}
		return artifacts, nil
	}

	return artifactsResp.Artifacts, nil
}

func downloadAndScanArtifact(client *gitea.Client, repo *gitea.Repository, run ActionWorkflowRun, artifact ActionArtifact) {
	// Download artifact as zip
	// Gitea Actions API: GET /repos/{owner}/{repo}/actions/artifacts/{artifact_id}
	apiPath := fmt.Sprintf("/api/v1/repos/%s/%s/actions/artifacts/%d", repo.Owner.UserName, repo.Name, artifact.ID)
	fullURL := fmt.Sprintf("%s%s", scanOptions.GiteaURL, apiPath)

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

	if resp.StatusCode != 200 {
		log.Error().Int("status", resp.StatusCode).Str("repo", repo.FullName).Int64("artifact_id", artifact.ID).Msg("failed to download artifact")
		return
	}

	artifactBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Error().Err(err).Str("repo", repo.FullName).Int64("artifact_id", artifact.ID).Msg("failed to read artifact bytes")
		return
	}

	// Try to parse as ZIP archive
	zipReader, err := zip.NewReader(bytes.NewReader(artifactBytes), int64(len(artifactBytes)))
	if err != nil {
		log.Debug().Err(err).Str("repo", repo.FullName).Int64("artifact_id", artifact.ID).Msg("Artifact is not a zip, scanning directly")
		// Not a zip, scan directly
		scanArtifactContent(artifactBytes, repo, run, artifact.Name, "")
		return
	}

	// Process files in the zip
	ctx := scanOptions.Context
	group := parallel.Limited(ctx, scanOptions.MaxScanGoRoutines)

	for _, file := range zipReader.File {
		file := file // capture for closure
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

	// Skip known binary file types
	if kind == filetype.Unknown {
		// Scan for secrets
		scanner.DetectFileHits(content, run.HTMLURL, run.Name, displayName, repo.FullName, scanOptions.TruffleHogVerification)
	} else if filetype.IsArchive(content) {
		// Handle nested archives
		scanner.HandleArchiveArtifact(displayName, content, run.HTMLURL, run.Name, scanOptions.TruffleHogVerification)
	} else {
		log.Trace().Str("file", displayName).Str("type", kind.MIME.Value).Msg("Skipping known file type")
	}
}

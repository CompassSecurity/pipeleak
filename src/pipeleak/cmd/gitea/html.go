package gitea

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"sync/atomic"

	"code.gitea.io/sdk/gitea"
	"github.com/rs/zerolog/log"
	"github.com/wandb/parallel"
)

func validateCookie() {
	urlStr, err := buildGiteaURL("/issues")
	if err != nil {
		log.Fatal().Err(err).Msg("Failed parsing Gitea URL for cookie validation")
	}

	resp, err := makeHTTPRequest(urlStr)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed cookie validation request")
	}

	log.Debug().Int("status", resp.StatusCode).Msg("Cookie validation response status code")

	// Check if response contains login page indicators
	if strings.Contains(string(resp.Body), "/user/login") {
		log.Fatal().Msg("Cookie validation failed - redirected to login page, cookie is invalid or expired")
	} else {
		log.Info().Msg("Cookie validation succeeded")
	}
}

// scanRepositoryWithCookie uses HTML scraping with cookie authentication as fallback when API returns 403
func scanRepositoryWithCookie(repo *gitea.Repository) {
	log.Debug().Str("repo", repo.FullName).Msg("Using cookie-based HTML scraping for workflow runs")

	// Get the latest run ID from the HTML actions page
	latestRunID, err := getLatestRunIDFromHTML(repo)
	if err != nil {
		log.Error().Err(err).Str("repo", repo.FullName).Msg("failed to get latest run ID from HTML")
		return
	}

	if latestRunID == 0 {
		log.Debug().Str("repo", repo.FullName).Msg("Actions disabled or no runs found")
		return
	}

	log.Debug().Str("repo", repo.FullName).Int64("latest_run_id", latestRunID).Msg("Found latest run ID, scanning backwards in parallel")

	ctx := context.Background()
	group := parallel.Limited(ctx, scanOptions.MaxScanGoRoutines)
	var failedCounter int32
	var scannedCounter int32

	_, cancel := context.WithCancel(ctx)
	defer cancel()

	for i := latestRunID; i > 0; i-- {
		// Check if we've reached the runs limit
		if scanOptions.RunsLimit > 0 && int(atomic.LoadInt32(&scannedCounter)) >= scanOptions.RunsLimit {
			log.Debug().Str("repo", repo.FullName).Int("limit", scanOptions.RunsLimit).Msg("Reached runs limit, stopping scan")
			cancel()
			break
		}

		// stop early if too many failures
		if atomic.LoadInt32(&failedCounter) > 5 {
			log.Debug().Msg("Too many failures, aborting scan loop.")
			cancel()
			break
		}

		runID := i
		group.Go(func(ctx context.Context) {
			select {
			case <-ctx.Done():
				// canceled: stop early
				return
			default:
			}

			log.Trace().Str("repo", repo.FullName).Int64("run_id", runID).Msg("Checking run ID")

			ok := scanJobLogsWithCookie(repo, runID, 0)
			if ok {
				atomic.AddInt32(&scannedCounter, 1)
			} else {
				atomic.AddInt32(&failedCounter, 1)
			}
		})
	}

	group.Wait()
}

// getLatestRunIDFromHTML fetches the actions page and extracts the latest run ID
func getLatestRunIDFromHTML(repo *gitea.Repository) (int64, error) {
	// Construct URL: https://gitea.com/owner/repo/actions
	urlStr, err := buildGiteaURL("/%s/actions", repo.FullName)
	if err != nil {
		return 0, err
	}

	resp, err := makeHTTPRequest(urlStr)
	if err != nil {
		return 0, err
	}

	if err := checkHTTPStatus(resp.StatusCode, "fetch actions page"); err != nil {
		if resp.StatusCode == 404 {
			return 0, nil // Actions disabled
		}
		return 0, err
	}

	// Parse HTML to find run IDs
	// Look for patterns like: /owner/repo/actions/runs/123
	runIDPattern := regexp.MustCompile(fmt.Sprintf(`/%s/actions/runs/(\d+)`, regexp.QuoteMeta(repo.FullName)))
	matches := runIDPattern.FindAllStringSubmatch(string(resp.Body), -1)

	if len(matches) == 0 {
		return 0, nil
	}

	// Get the latest run ID
	latestRunIDStr := matches[0][1]
	latestRunID, err := strconv.ParseInt(latestRunIDStr, 10, 64)
	if err != nil {
		return 0, err
	}

	return latestRunID, nil
}

// scanJobLogsWithCookie fetches and scans job logs using cookie authentication
func scanJobLogsWithCookie(repo *gitea.Repository, runID int64, jobID int64) bool {
	urlStr, err := buildGiteaURL("/%s/actions/runs/%d/jobs/%d/logs", repo.FullName, runID, jobID)
	if err != nil {
		log.Error().Err(err).Str("repo", repo.FullName).Int64("run_id", runID).Int64("job_id", jobID).Msg("failed to build URL")
		return false
	}

	resp, err := makeHTTPRequest(urlStr)
	if err != nil {
		log.Error().Err(err).Str("repo", repo.FullName).Int64("run_id", runID).Int64("job_id", jobID).Msg("failed to download logs")
		return false
	}

	ctx := logContext{Repo: repo.FullName, RunID: runID, JobID: jobID}

	if resp.StatusCode == 404 {
		log.Debug().Str("repo", repo.FullName).Int64("run_id", runID).Int64("job_id", jobID).Msg("Logs not found")
		return false
	}

	if err := checkHTTPStatus(resp.StatusCode, "download logs"); err != nil {
		logHTTPError(resp.StatusCode, "download logs", ctx)
		return false
	}

	// Build run URL for findings
	runURLParsed, err := url.Parse(scanOptions.GiteaURL)
	if err != nil {
		log.Error().Err(err).Msg("failed to parse Gitea URL")
		return false
	}
	runURLParsed.Path = fmt.Sprintf("/%s/actions/runs/%d", repo.FullName, runID)
	runURL := runURLParsed.String()

	// Create minimal run for scanning
	run := ActionWorkflowRun{
		ID:      runID,
		Name:    fmt.Sprintf("Run %d", runID),
		HTMLURL: runURL,
	}

	log.Debug().Str("url", run.HTMLURL).Msg("Scanning logs")
	scanLogs(resp.Body, repo, run, jobID, "")

	if scanOptions.Artifacts {
		scanArtifactsWithCookie(repo, runID, runURL)
	}

	return true
}

// scanArtifactsWithCookie downloads and scans artifacts for a workflow run using cookie authentication
func scanArtifactsWithCookie(repo *gitea.Repository, runID int64, runURL string) {
	// Get the run HTML page to extract artifact URLs
	artifactURLs, err := getArtifactURLsFromRunHTML(repo, runID)
	if err != nil {
		log.Error().Err(err).Str("repo", repo.FullName).Int64("run_id", runID).Msg("failed to get artifact URLs from HTML")
		return
	}

	if len(artifactURLs) == 0 {
		log.Trace().Str("repo", repo.FullName).Int64("run_id", runID).Msg("No artifacts found in run")
		return
	}

	log.Debug().Str("repo", repo.FullName).Int64("run_id", runID).Int("count", len(artifactURLs)).Msg("Found artifacts in run")

	// Create a minimal ActionWorkflowRun for scanArtifactContent
	run := ActionWorkflowRun{
		ID:      runID,
		Name:    fmt.Sprintf("Run %d", runID),
		HTMLURL: runURL,
	}

	ctx := scanOptions.Context
	group := parallel.Limited(ctx, scanOptions.MaxScanGoRoutines)

	for artifactName, artifactURL := range artifactURLs {
		name := artifactName
		url := artifactURL
		group.Go(func(ctx context.Context) {
			downloadAndScanArtifactWithCookie(repo, run, name, url)
		})
	}

	group.Wait()
}

// getArtifactURLsFromRunHTML parses the run HTML page and extracts artifact download URLs
func getArtifactURLsFromRunHTML(repo *gitea.Repository, runID int64) (map[string]string, error) {

	// Fetch CSRF token
	csrfToken, err := fetchCsrfToken()
	if err != nil || len(csrfToken) == 0 {
		log.Error().Err(err).Str("repo", repo.FullName).Int64("run_id", runID).Msg("failed to fetch CSRF token")
		return nil, fmt.Errorf("failed to fetch CSRF token: %w", err)
	}

	// Construct URL: https://gitea.com/owner/repo/actions/runs/{run_id}/jobs/0
	// This endpoint returns job information including artifacts
	urlStr, err := buildGiteaURL("/%s/actions/runs/%d/jobs/0", repo.FullName, runID)
	if err != nil {
		return nil, err
	}

	// Create POST request with CSRF token
	headers := map[string]string{
		"Content-Type": "application/json",
		"x-csrf-token": csrfToken,
		"Accept":       "*/*",
	}
	requestBody := []byte(`{"logCursors":[]}`)

	resp, err := makeHTTPPostRequest(urlStr, requestBody, headers)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch job data: %w", err)
	}

	if err := checkHTTPStatus(resp.StatusCode, "fetch job data"); err != nil {
		return nil, err
	}

	// Parse JSON response to extract artifacts
	var jobData struct {
		Artifacts []struct {
			Name   string `json:"name"`
			Size   int64  `json:"size"`
			Status string `json:"status"`
		} `json:"artifacts"`
	}

	if err := json.Unmarshal(resp.Body, &jobData); err != nil {
		log.Debug().Err(err).Str("body", string(resp.Body)).Msg("Failed to parse job data JSON")
		return nil, fmt.Errorf("failed to parse job data: %w", err)
	}

	// Build artifact URLs
	artifactURLs := make(map[string]string)

	for _, artifact := range jobData.Artifacts {
		// Build download URL: /owner/repo/actions/runs/{run_id}/artifacts/{artifact_name}
		artifactURL, err := url.Parse(scanOptions.GiteaURL)
		if err != nil {
			continue
		}
		// Use artifact name directly in the download URL
		artifactURL.Path = fmt.Sprintf("/%s/actions/runs/%d/artifacts/%s", repo.FullName, runID, url.PathEscape(artifact.Name))

		artifactURLs[artifact.Name] = artifactURL.String()
		log.Debug().Str("name", artifact.Name).Int64("size", artifact.Size).Str("url", artifactURL.String()).Msg("Found artifact")
	}

	return artifactURLs, nil
}

// downloadAndScanArtifactWithCookie downloads and scans a single artifact using cookie authentication
func downloadAndScanArtifactWithCookie(repo *gitea.Repository, run ActionWorkflowRun, artifactName string, artifactURL string) {
	log.Warn().Str("repo", repo.FullName).Int64("run_id", run.ID).Str("artifact", artifactName).Msg("Downloading artifact with cookie")

	resp, err := makeHTTPRequest(artifactURL)
	if err != nil {
		log.Error().Err(err).Str("repo", repo.FullName).Int64("run_id", run.ID).Str("artifact", artifactName).Msg("failed to download artifact")
		return
	}

	if resp.StatusCode == 404 || resp.StatusCode == 410 {
		log.Debug().Str("repo", repo.FullName).Int64("run_id", run.ID).Str("artifact", artifactName).Msg("Artifact expired or not found")
		return
	}

	if err := checkHTTPStatus(resp.StatusCode, "download artifact"); err != nil {
		log.Error().Int("status", resp.StatusCode).Str("repo", repo.FullName).Int64("run_id", run.ID).Str("artifact", artifactName).Msg("failed to download artifact")
		return
	}

	processZipArtifact(resp.Body, repo, run, artifactName)
}

// fetchCsrfToken fetches the CSRF token from the Gitea homepage
func fetchCsrfToken() (string, error) {
	urlStr, err := buildGiteaURL("/issues")
	if err != nil {
		return "", fmt.Errorf("failed to parse Gitea URL: %w", err)
	}

	resp, err := makeHTTPRequest(urlStr)
	if err != nil {
		return "", fmt.Errorf("failed to fetch homepage: %w", err)
	}

	if err := checkHTTPStatus(resp.StatusCode, "fetch CSRF token"); err != nil {
		return "", err
	}

	// Extract CSRF token using regex
	// Looking for: csrfToken: 'TOKEN_VALUE',
	csrfPattern := regexp.MustCompile(`csrfToken:\s*['"]([^'"]+)['"]`)
	matches := csrfPattern.FindSubmatch(resp.Body)

	if len(matches) < 2 {
		return "", fmt.Errorf("CSRF token not found in response")
	}

	csrfToken := string(matches[1])
	log.Trace().Str("csrf_token", csrfToken).Msg("Fetched CSRF token")

	return csrfToken, nil
}

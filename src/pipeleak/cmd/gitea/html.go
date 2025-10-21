package gitea

import (
	"archive/zip"
	"bytes"
	"context"
	"fmt"
	"io"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"sync/atomic"

	"code.gitea.io/sdk/gitea"
	"github.com/CompassSecurity/pipeleak/scanner"
	"github.com/rs/zerolog/log"
	"github.com/wandb/parallel"
)

func validateCookie() {
	issuesURL, err := url.Parse(scanOptions.GiteaURL)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed parsing Gitea URL for cookie validation")
	}

	issuesURL.Path = "/issues"

	resp, err := scanOptions.HttpClient.Get(issuesURL.String())
	if err != nil {
		log.Fatal().Err(err).Msg("Failed cookie validation request")
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed reading cookie validation response")
	}

	log.Debug().Int("status", resp.StatusCode).Msg("Cookie validation response status code")

	// Check if response contains login page indicators
	bodyStr := string(body)
	if strings.Contains(bodyStr, "/user/login") {
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
	link, err := url.Parse(scanOptions.GiteaURL)
	if err != nil {
		return 0, err
	}
	link.Path = fmt.Sprintf("/%s/actions", repo.FullName)

	resp, err := scanOptions.HttpClient.Get(link.String())
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == 403 {
		return 0, fmt.Errorf("access forbidden, check your cookie")
	}

	if resp.StatusCode == 404 {
		return 0, nil
	}

	if resp.StatusCode != 200 {
		return 0, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return 0, err
	}

	// Parse HTML to find run IDs
	// Look for patterns like: /owner/repo/actions/runs/123
	runIDPattern := regexp.MustCompile(fmt.Sprintf(`/%s/actions/runs/(\d+)`, regexp.QuoteMeta(repo.FullName)))
	matches := runIDPattern.FindAllStringSubmatch(string(body), -1)

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
	link, err := url.Parse(scanOptions.GiteaURL)
	if err != nil {
		log.Error().Err(err).Str("repo", repo.FullName).Int64("run_id", runID).Int64("job_id", jobID).Msg("failed to parse URL")
		return false
	}
	link.Path = fmt.Sprintf("/%s/actions/runs/%d/jobs/%d/logs", repo.FullName, runID, jobID)

	resp, err := scanOptions.HttpClient.Get(link.String())
	if err != nil {
		log.Error().Err(err).Str("repo", repo.FullName).Int64("run_id", runID).Int64("job_id", jobID).Msg("failed to download logs")
		return false
	}

	defer resp.Body.Close()

	if resp.StatusCode == 404 {
		log.Debug().Str("repo", repo.FullName).Int64("run_id", runID).Int64("job_id", jobID).Msg("Logs not found")
		return false
	}

	if resp.StatusCode != 200 {
		log.Error().Int("status", resp.StatusCode).Str("repo", repo.FullName).Int64("run_id", runID).Int64("job_id", jobID).Msg("failed to download logs")
		return false
	}

	logBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Error().Err(err).Str("repo", repo.FullName).Int64("run_id", runID).Int64("job_id", jobID).Msg("failed to read log bytes")
		return false
	}

	findings, err := scanner.DetectHits(logBytes, scanOptions.MaxScanGoRoutines, scanOptions.TruffleHogVerification)
	if err != nil {
		log.Debug().Err(err).Str("repo", repo.FullName).Int64("run_id", runID).Int64("job_id", jobID).Msg("Failed detecting secrets in logs")
		return false
	}

	runURLParsed, err := url.Parse(scanOptions.GiteaURL)
	if err != nil {
		log.Error().Err(err).Msg("failed to parse Gitea URL")
		return false
	}
	runURLParsed.Path = fmt.Sprintf("/%s/actions/runs/%d", repo.FullName, runID)
	runURL := runURLParsed.String()

	for _, finding := range findings {
		log.Warn().
			Str("confidence", finding.Pattern.Pattern.Confidence).
			Str("ruleName", finding.Pattern.Pattern.Name).
			Str("value", finding.Text).
			Str("repo", repo.FullName).
			Int64("run_id", runID).
			Int64("job_id", jobID).
			Str("url", runURL).
			Msg("HIT")
	}

	// Scan artifacts if --artifacts flag is set
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
		log.Debug().Str("repo", repo.FullName).Int64("run_id", runID).Msg("No artifacts found in run")
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

	csrfToken, err := fetchCsrfToken()
	if err != nil || len(csrfToken) == 0 {
		log.Error().Err(err).Str("repo", repo.FullName).Int64("run_id", runID).Msg("failed to fetch CSRF token")
		return nil, fmt.Errorf("failed to fetch CSRF token")
	}

	log.Error().Str("csrf_token", csrfToken).Msg("Using CSRF token to download artifact")
	// Construct URL: https://gitea.com/owner/repo/actions/runs/{run_id}
	link, err := url.Parse(scanOptions.GiteaURL)
	if err != nil {
		return nil, err
	}
	link.Path = fmt.Sprintf("/%s/actions/runs/%d", repo.FullName, runID)

	resp, err := scanOptions.HttpClient.Get(link.String())
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == 404 {
		return nil, fmt.Errorf("run not found")
	}

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	_, err = io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	artifactURLs := make(map[string]string)

	return artifactURLs, nil
}

// downloadAndScanArtifactWithCookie downloads and scans a single artifact using cookie authentication
func downloadAndScanArtifactWithCookie(repo *gitea.Repository, run ActionWorkflowRun, artifactName string, artifactURL string) {
	log.Warn().Str("repo", repo.FullName).Int64("run_id", run.ID).Str("artifact", artifactName).Msg("Downloading artifact with cookie")

	resp, err := scanOptions.HttpClient.Get(artifactURL)
	if err != nil {
		log.Error().Err(err).Str("repo", repo.FullName).Int64("run_id", run.ID).Str("artifact", artifactName).Msg("failed to download artifact")
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode == 404 || resp.StatusCode == 410 {
		log.Debug().Str("repo", repo.FullName).Int64("run_id", run.ID).Str("artifact", artifactName).Msg("Artifact expired or not found")
		return
	}

	if resp.StatusCode != 200 {
		log.Error().Int("status", resp.StatusCode).Str("repo", repo.FullName).Int64("run_id", run.ID).Str("artifact", artifactName).Msg("failed to download artifact")
		return
	}

	artifactBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Error().Err(err).Str("repo", repo.FullName).Int64("run_id", run.ID).Str("artifact", artifactName).Msg("failed to read artifact bytes")
		return
	}

	// Try to read as ZIP first
	zipReader, err := zip.NewReader(bytes.NewReader(artifactBytes), int64(len(artifactBytes)))
	if err != nil {
		// Not a ZIP file, scan directly
		log.Debug().Str("repo", repo.FullName).Int64("run_id", run.ID).Str("artifact", artifactName).Msg("Artifact is not a zip, scanning directly")
		scanArtifactContent(artifactBytes, repo, run, artifactName, "")
		return
	}

	// Process ZIP file contents
	ctx := scanOptions.Context
	group := parallel.Limited(ctx, scanOptions.MaxScanGoRoutines)

	for _, file := range zipReader.File {
		f := file
		group.Go(func(ctx context.Context) {
			fc, err := f.Open()
			if err != nil {
				log.Debug().Err(err).Str("file", f.Name).Msg("Unable to open file in artifact zip")
				return
			}
			defer fc.Close()

			content, err := io.ReadAll(fc)
			if err != nil {
				log.Debug().Err(err).Str("file", f.Name).Msg("Unable to read file in artifact zip")
				return
			}

			scanArtifactContent(content, repo, run, artifactName, f.Name)
		})
	}

	group.Wait()
}

// fetchCsrfToken fetches the CSRF token from the Gitea homepage
func fetchCsrfToken() (string, error) {
	// Construct the base Gitea URL
	link, err := url.Parse(scanOptions.GiteaURL)
	if err != nil {
		return "", fmt.Errorf("failed to parse Gitea URL: %w", err)
	}

	// Fetch the homepage
	resp, err := scanOptions.HttpClient.Get(link.String() + "/issues")
	if err != nil {
		return "", fmt.Errorf("failed to fetch homepage: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return "", fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	// Read the response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response body: %w", err)
	}

	// Extract CSRF token using regex
	// Looking for: csrfToken: 'TOKEN_VALUE',
	csrfPattern := regexp.MustCompile(`csrfToken:\s*['"]([^'"]+)['"]`)
	matches := csrfPattern.FindSubmatch(body)

	if len(matches) < 2 {
		return "", fmt.Errorf("CSRF token not found in response")
	}

	csrfToken := string(matches[1])
	log.Debug().Str("csrf_token", csrfToken).Msg("Fetched CSRF token")

	return csrfToken, nil
}

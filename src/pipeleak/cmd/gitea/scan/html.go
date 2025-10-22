package scan

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

	resp, err := makeHTTPGetRequest(urlStr)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed cookie validation request")
	}

	log.Debug().Int("status", resp.StatusCode).Msg("Cookie validation response status code")

	if strings.Contains(string(resp.Body), "/user/login") {
		log.Fatal().Msg("Cookie validation failed - redirected to login page, cookie is invalid or expired")
	} else {
		log.Info().Msg("Cookie validation succeeded")
	}
}

func scanRepositoryWithCookie(repo *gitea.Repository) {
	if repo == nil {
		log.Error().Msg("Cannot scan repository: repository is nil")
		return
	}

	log.Debug().Str("repo", repo.FullName).Msg("Using cookie-based HTML scraping for workflow runs")

	var startRunID int64

	if scanOptions.StartRunID > 0 {
		startRunID = scanOptions.StartRunID
		log.Debug().Str("repo", repo.FullName).Int64("start_run_id", startRunID).Msg("Starting from specified run ID")
	} else {
		latestRunID, err := getLatestRunIDFromHTML(repo)
		if err != nil {
			log.Error().Err(err).Str("repo", repo.FullName).Msg("failed to get latest run ID from HTML")
			return
		}

		if latestRunID == 0 {
			log.Debug().Str("repo", repo.FullName).Msg("Actions disabled or no runs found")
			return
		}

		startRunID = latestRunID
	}

	startRunURL := fmt.Sprintf("%s/%s/actions/runs/%d", scanOptions.GiteaURL, repo.FullName, startRunID)
	log.Debug().Str("repo", repo.FullName).Int64("start_run_id", startRunID).Str("url", startRunURL).Msg("Scanning from run ID")

	ctx := context.Background()
	group := parallel.Limited(ctx, scanOptions.MaxScanGoRoutines)
	var scannedCounter int32
	var consecutiveFailures int32
	var lastSuccessfulRunID int64

	_, cancel := context.WithCancel(ctx)
	defer cancel()

	// Track results by run ID to determine if failures are consecutive
	type runResult struct {
		runID   int64
		success bool
	}
	resultsChan := make(chan runResult, 100)

	monitorDone := make(chan struct{})
	go func() {
		defer close(monitorDone)
		consecutiveCount := 0
		nextExpectedRunID := startRunID
		resultBuffer := make(map[int64]bool)

		for result := range resultsChan {
			resultBuffer[result.runID] = result.success

			for {
				success, exists := resultBuffer[nextExpectedRunID]
				if !exists {
					break
				}

				delete(resultBuffer, nextExpectedRunID)

				if success {
					consecutiveCount = 0
					atomic.StoreInt64(&lastSuccessfulRunID, nextExpectedRunID)
					atomic.StoreInt32(&consecutiveFailures, 0)
				} else {
					consecutiveCount++
					atomic.StoreInt32(&consecutiveFailures, int32(consecutiveCount))

					if consecutiveCount >= 5 {
						log.Debug().
							Str("repo", repo.FullName).
							Int("consecutive_failures", consecutiveCount).
							Int64("last_successful_run", atomic.LoadInt64(&lastSuccessfulRunID)).
							Int64("failed_at_run", nextExpectedRunID).
							Msg("Too many consecutive failures, aborting scan loop")
						cancel()
					}
				}

				nextExpectedRunID--
			}
		}
	}()

	for i := startRunID; i > 0; i-- {
		if scanOptions.RunsLimit > 0 && int(atomic.LoadInt32(&scannedCounter)) >= scanOptions.RunsLimit {
			log.Debug().Str("repo", repo.FullName).Int("limit", scanOptions.RunsLimit).Msg("Reached runs limit, stopping scan")
			cancel()
			break
		}

		if atomic.LoadInt32(&consecutiveFailures) >= 5 {
			log.Debug().Str("repo", repo.FullName).Msg("Stopping due to consecutive failures")
			cancel()
			break
		}

		runID := i
		group.Go(func(ctx context.Context) {
			select {
			case <-ctx.Done():
				return
			default:
			}

			runURL := fmt.Sprintf("%s/%s/actions/runs/%d", scanOptions.GiteaURL, repo.FullName, runID)
			log.Trace().Str("repo", repo.FullName).Int64("run_id", runID).Str("url", runURL).Msg("Checking run ID")

			ok := scanJobLogsWithCookie(repo, runID, 0)
			if ok {
				atomic.AddInt32(&scannedCounter, 1)
			}

			select {
			case resultsChan <- runResult{runID: runID, success: ok}:
			case <-ctx.Done():
			}
		})
	}

	group.Wait()
	close(resultsChan)
	<-monitorDone
}

func getLatestRunIDFromHTML(repo *gitea.Repository) (int64, error) {
	if repo == nil {
		return 0, fmt.Errorf("repository is nil")
	}

	urlStr, err := buildGiteaURL("/%s/actions", repo.FullName)
	if err != nil {
		return 0, err
	}

	resp, err := makeHTTPGetRequest(urlStr)
	if err != nil {
		return 0, err
	}

	if err := checkHTTPStatus(resp.StatusCode, "fetch actions page"); err != nil {
		if resp.StatusCode == 404 {
			return 0, nil
		}
		return 0, err
	}

	runIDPattern := regexp.MustCompile(fmt.Sprintf(`/%s/actions/runs/(\d+)`, regexp.QuoteMeta(repo.FullName)))
	matches := runIDPattern.FindAllStringSubmatch(string(resp.Body), -1)

	if len(matches) == 0 {
		return 0, nil
	}

	latestRunIDStr := matches[0][1]
	latestRunID, err := strconv.ParseInt(latestRunIDStr, 10, 64)
	if err != nil {
		return 0, err
	}

	return latestRunID, nil
}

func scanJobLogsWithCookie(repo *gitea.Repository, runID int64, jobID int64) bool {
	if repo == nil {
		log.Error().Msg("Cannot scan job logs: repository is nil")
		return false
	}

	jobURL := fmt.Sprintf("%s/%s/actions/runs/%d/jobs/%d", scanOptions.GiteaURL, repo.FullName, runID, jobID)
	urlStr, err := buildGiteaURL("/%s/actions/runs/%d/jobs/%d/logs", repo.FullName, runID, jobID)
	if err != nil {
		log.Error().Err(err).Str("url", jobURL).Msg("failed to build URL")
		return false
	}

	resp, err := makeHTTPGetRequest(urlStr)
	if err != nil {
		log.Error().Err(err).Str("url", jobURL).Msg("failed to download logs")
		return false
	}

	ctx := logContext{Repo: repo.FullName, RunID: runID, JobID: jobID}

	if resp.StatusCode == 404 {
		log.Debug().Str("url", jobURL).Msg("Logs not found")
		return false
	}

	if err := checkHTTPStatus(resp.StatusCode, "download logs"); err != nil {
		logHTTPError(resp.StatusCode, "download logs", ctx)
		return false
	}

	runURLParsed, err := url.Parse(scanOptions.GiteaURL)
	if err != nil {
		log.Error().Err(err).Msg("failed to parse Gitea URL")
		return false
	}
	runURLParsed.Path = fmt.Sprintf("/%s/actions/runs/%d", repo.FullName, runID)
	runURL := runURLParsed.String()

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

func scanArtifactsWithCookie(repo *gitea.Repository, runID int64, runURL string) {
	if repo == nil {
		log.Error().Msg("Cannot scan artifacts: repository is nil")
		return
	}

	artifactURLs, err := getArtifactURLsFromRunHTML(repo, runID)
	if err != nil {
		log.Error().Err(err).Str("repo", repo.FullName).Int64("run_id", runID).Str("url", runURL).Msg("failed to get artifact URLs from HTML")
		return
	}

	if len(artifactURLs) == 0 {
		log.Trace().Str("url", runURL).Msg("No artifacts found in run")
		return
	}

	log.Debug().Str("url", runURL).Int("count", len(artifactURLs)).Msg("Found artifacts in run")

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

func getArtifactURLsFromRunHTML(repo *gitea.Repository, runID int64) (map[string]string, error) {
	if repo == nil {
		return nil, fmt.Errorf("repository is nil")
	}

	runURL := fmt.Sprintf("%s/%s/actions/runs/%d", scanOptions.GiteaURL, repo.FullName, runID)
	csrfToken, err := fetchCsrfToken()
	if err != nil || len(csrfToken) == 0 {
		log.Error().Err(err).Str("url", runURL).Msg("failed to fetch CSRF token")
		return nil, fmt.Errorf("failed to fetch CSRF token: %w", err)
	}

	urlStr, err := buildGiteaURL("/%s/actions/runs/%d/jobs/0", repo.FullName, runID)
	if err != nil {
		return nil, err
	}

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

	artifactURLs := make(map[string]string)

	for _, artifact := range jobData.Artifacts {
		artifactURL, err := url.Parse(scanOptions.GiteaURL)
		if err != nil {
			continue
		}
		artifactURL.Path = fmt.Sprintf("/%s/actions/runs/%d/artifacts/%s", repo.FullName, runID, url.PathEscape(artifact.Name))
		artifactURLs[artifact.Name] = artifactURL.String()
		log.Debug().Str("name", artifact.Name).Int64("size", artifact.Size).Str("url", artifactURL.String()).Msg("Found artifact")
	}

	return artifactURLs, nil
}

func downloadAndScanArtifactWithCookie(repo *gitea.Repository, run ActionWorkflowRun, artifactName string, artifactURL string) {
	if repo == nil {
		log.Error().Msg("Cannot download artifact: repository is nil")
		return
	}

	log.Trace().Str("artifact", artifactName).Str("url", run.HTMLURL).Msg("Downloading artifact with cookie")

	resp, err := makeHTTPGetRequest(artifactURL)
	if err != nil {
		log.Error().Err(err).Str("artifact", artifactName).Str("url", run.HTMLURL).Msg("failed to download artifact")
		return
	}

	if resp.StatusCode == 404 || resp.StatusCode == 410 {
		log.Debug().Str("artifact", artifactName).Str("url", run.HTMLURL).Msg("Artifact expired or not found")
		return
	}

	if err := checkHTTPStatus(resp.StatusCode, "download artifact"); err != nil {
		log.Error().Int("status", resp.StatusCode).Str("artifact", artifactName).Str("url", run.HTMLURL).Msg("failed to download artifact")
		return
	}

	processZipArtifact(resp.Body, repo, run, artifactName)
}

func fetchCsrfToken() (string, error) {
	urlStr, err := buildGiteaURL("/issues")
	if err != nil {
		return "", fmt.Errorf("failed to parse Gitea URL: %w", err)
	}

	resp, err := makeHTTPGetRequest(urlStr)
	if err != nil {
		return "", fmt.Errorf("failed to fetch homepage: %w", err)
	}

	if err := checkHTTPStatus(resp.StatusCode, "fetch CSRF token"); err != nil {
		return "", err
	}

	csrfPattern := regexp.MustCompile(`csrfToken:\s*['"]([^'"]+)['"]`)
	matches := csrfPattern.FindSubmatch(resp.Body)

	if len(matches) < 2 {
		return "", fmt.Errorf("CSRF token not found in response")
	}

	csrfToken := string(matches[1])

	return csrfToken, nil
}

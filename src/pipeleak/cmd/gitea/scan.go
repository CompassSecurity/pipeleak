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
	"regexp"
	"strconv"
	"strings"
	"sync/atomic"

	"code.gitea.io/sdk/gitea"
	"github.com/CompassSecurity/pipeleak/helper"
	"github.com/CompassSecurity/pipeleak/scanner"
	"github.com/h2non/filetype"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"github.com/wandb/parallel"
)

var scanOptions = GiteaScanOptions{}

func NewScanCmd() *cobra.Command {
	scanCmd := &cobra.Command{
		Use:   "scan",
		Short: "Scan Gitea Actions",
		Long: `Scan Gitea Actions workflow runs and artifacts for secrets

### Cookie Authentication

Due to differences between Gitea Actions API and UI access rights validation, a session cookie may be required in some cases.
The Actions API and UI are not yet fully in sync, causing some repositories to return 403 errors via API even when accessible through the UI.

To obtain the cookie:
1. Open your Gitea instance in a web browser
2. Open Developer Tools (F12)
3. Navigate to Application/Storage > Cookies
4. Find and copy the value of the 'i_like_gitea' cookie
5. Use it with the --cookie flag
`,
		Example: `
# Scan all accessible repositories (including public) and their artifacts
pipeleak gitea scan --token gitea_token_xxxxx --gitea https://gitea.example.com --artifacts --cookie your_cookie_value

# Scan without downloading artifacts
pipeleak gitea scan --token gitea_token_xxxxx --gitea https://gitea.example.com --cookie your_cookie_value

# Scan only repositories owned by the user
pipeleak gitea scan --token gitea_token_xxxxx --gitea https://gitea.example.com --owned --cookie your_cookie_value

# Scan all repositories of a specific organization
pipeleak gitea scan --token gitea_token_xxxxx --gitea https://gitea.example.com --organization my-org --cookie your_cookie_value

# Scan a specific repository
pipeleak gitea scan --token gitea_token_xxxxx --gitea https://gitea.example.com --repository owner/repo-name --cookie your_cookie_value
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
	scanCmd.Flags().StringVarP(&scanOptions.Repository, "repository", "r", "", "Scan a specific repository (format: owner/repo)")
	scanCmd.Flags().StringVarP(&scanOptions.Cookie, "cookie", "", "", "Gitea session cookie (i_like_gitea) for fallback authentication when API returns 403")
	scanCmd.Flags().IntVarP(&scanOptions.RunsLimit, "runs-limit", "", 0, "Limit the number of workflow runs to scan per repository (0 = unlimited)")
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

	authHeaders := map[string]string{"Authorization": "token " + scanOptions.Token}
	scanOptions.HttpClient = helper.GetPipeleakHTTPClient("", nil, authHeaders)

	httpClient := &http.Client{
		Transport: &AuthTransport{
			Base:  http.DefaultTransport,
			Token: scanOptions.Token,
		},
	}

	scanOptions.HttpClient.StandardClient().Transport = httpClient.Transport

	if scanOptions.Cookie != "" {
		validateCookie()
		scanOptions.HttpClient = helper.GetPipeleakHTTPClient(
			scanOptions.GiteaURL,
			[]*http.Cookie{
				{Name: "i_like_gitea", Value: scanOptions.Cookie, Secure: true},
			},
			authHeaders,
		)

		if err != nil {
			log.Fatal().Err(err).Msg("Failed to parse Gitea URL for cookie debugging")
		}
	}

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
func validateCookie() {
	issuesURL, err := url.Parse(scanOptions.GiteaURL)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed parsing Gitea URL for cookie validation")
	}
	issuesURL.Path = "/issues"

	req, err := http.NewRequest("GET", issuesURL.String(), nil)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed creating cookie validation request")
	}

	// Temporarily add cookie for validation (before cookieTransport is set)
	req.Header.Set("Cookie", fmt.Sprintf("i_like_gitea=%s", scanOptions.Cookie))

	// Use a standard HTTP client that doesn't automatically retry to properly detect redirects
	client := &http.Client{
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			// Don't follow redirects, we want to examine them
			return http.ErrUseLastResponse
		},
	}

	resp, err := client.Do(req)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed validating cookie")
	}
	defer resp.Body.Close()

	log.Debug().Int("status", resp.StatusCode).Msg("Cookie validation response status code")

	// Check for redirect to login page
	if resp.StatusCode == http.StatusSeeOther || resp.StatusCode == http.StatusFound {
		location := resp.Header.Get("Location")
		if strings.Contains(location, "/user/login") {
			log.Fatal().Str("location", location).Msg("Cookie validation failed - redirected to login page, cookie is invalid or expired")
		}
		log.Warn().Str("location", location).Msg("Unexpected redirect during cookie validation")
	}

	// Check status code
	switch resp.StatusCode {
	case http.StatusOK:
		log.Info().Msg("Cookie validation successful")
	case http.StatusSeeOther, http.StatusFound:
		// Already handled above
	default:
		log.Fatal().Int("status", resp.StatusCode).Msg("Cookie validation failed - unexpected status code")
	}
}

func scanRepositories(client *gitea.Client) {
	if scanOptions.Repository != "" {
		log.Info().Str("repository", scanOptions.Repository).Msg("Scan")
		scanSingleRepository(client, scanOptions.Repository)
	} else if scanOptions.Organization != "" {
		log.Info().Str("organization", scanOptions.Organization).Msg("Scan organization")
		scanOrganizationRepositories(client, scanOptions.Organization)
	} else if scanOptions.Owned {
		log.Info().Msg("Scan user owned")
		scanOwnedRepositories(client)
	} else {
		log.Info().Msg("Scan all")
		scanAllRepositories(client)
	}
}

func scanSingleRepository(client *gitea.Client, repoFullName string) {
	// Parse owner/repo format
	parts := strings.Split(repoFullName, "/")
	if len(parts) != 2 {
		log.Error().Str("repository", repoFullName).Msg("Invalid repository format, expected owner/repo")
		return
	}

	owner := parts[0]
	repoName := parts[1]

	// Get the specific repository
	repo, _, err := client.GetRepo(owner, repoName)
	if err != nil {
		log.Error().Err(err).Str("repository", repoFullName).Msg("failed to get repository")
		return
	}

	log.Info().Str("url", repo.HTMLURL).Msg("Scanning repository")
	scanRepository(client, repo)
}

func scanAllRepositories(client *gitea.Client) {
	// Use SearchRepos to get all accessible repositories (including public ones)
	// Empty keyword searches all repositories accessible with the current token
	opt := gitea.SearchRepoOptions{
		Sort:  "updated",
		Order: "desc",
		ListOptions: gitea.ListOptions{
			Page:     1,
			PageSize: 50,
		},
	}

	for {
		repos, resp, err := client.SearchRepos(opt)
		if err != nil {
			log.Error().Err(err).Int("page", opt.Page).Msg("failed to search repos")
			break
		}

		if len(repos) == 0 {
			break
		}

		log.Info().Int("count", len(repos)).Int("page", opt.Page).Msg("Processing repositories page")

		for _, repo := range repos {
			log.Debug().Str("url", repo.HTMLURL).Msg("Scanning repository")
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
				log.Debug().Str("url", repo.HTMLURL).Msg("Scanning repository")
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
	// Note: Gitea does not support nested organizations (organizations within organizations)
	// All repositories directly under the specified organization will be scanned
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
			log.Debug().Str("url", repo.HTMLURL).Msg("Scanning repository")
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
		// Check if it's a 403 error - this indicates the current user doesn't have API access
		// but might have UI access (API and UI access rights are not yet fully synchronized in Gitea)
		// When cookie is provided, fall back to HTML scraping which uses UI-level authentication
		if strings.Contains(err.Error(), "403") && scanOptions.Cookie != "" {
			log.Debug().Str("repo", repo.FullName).Msg("API returned 403, falling back to HTML scraping with cookie")
			scanRepositoryWithCookie(repo)
			return
		}
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
	link, err := url.Parse(scanOptions.GiteaURL)
	if err != nil {
		log.Error().Err(err).Str("repo", repo.FullName).Int64("run_id", run.ID).Int64("job_id", job.ID).Msg("failed to parse URL")
		return
	}
	link.Path = fmt.Sprintf("/api/v1/repos/%s/%s/actions/jobs/%d/logs", repo.Owner.UserName, repo.Name, job.ID)

	resp, err := scanOptions.HttpClient.Get(link.String())
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
	link, err := url.Parse(scanOptions.GiteaURL)
	if err != nil {
		log.Error().Err(err).Str("repo", repo.FullName).Int64("artifact_id", artifact.ID).Msg("failed to parse URL")
		return
	}
	link.Path = fmt.Sprintf("/api/v1/repos/%s/%s/actions/artifacts/%d/zip", repo.Owner.UserName, repo.Name, artifact.ID)

	resp, err := scanOptions.HttpClient.Get(link.String())
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
				log.Debug().Err(err).Str("file", file.Name).Msg("Unable to open file in artifact zip")
				return
			}
			defer fc.Close()

			content, err := io.ReadAll(fc)
			if err != nil {
				log.Debug().Err(err).Str("file", file.Name).Msg("Unable to read file in artifact zip")
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

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	// Parse HTML to find artifact download links
	// Pattern: <a href="/owner/repo/actions/runs/{run_id}/artifacts/{artifact_name}">
	// Example: <a href="/frj/secret-actions/actions/runs/123/artifacts/my-artifact">
	artifactPattern := regexp.MustCompile(fmt.Sprintf(`<a[^>]+href="(/%s/actions/runs/%d/artifacts/([^"]+))"`, regexp.QuoteMeta(repo.FullName), runID))
	matches := artifactPattern.FindAllStringSubmatch(string(body), -1)

	artifactURLs := make(map[string]string)
	for _, match := range matches {
		if len(match) >= 3 {
			artifactPath := match[1]
			artifactName := match[2]
			// Construct full URL
			fullURL, err := url.Parse(scanOptions.GiteaURL)
			if err != nil {
				log.Error().Err(err).Msg("failed to parse Gitea URL")
				continue
			}
			fullURL.Path = artifactPath
			artifactURLs[artifactName] = fullURL.String()
		}
	}

	return artifactURLs, nil
}

// downloadAndScanArtifactWithCookie downloads and scans a single artifact using cookie authentication
func downloadAndScanArtifactWithCookie(repo *gitea.Repository, run ActionWorkflowRun, artifactName string, artifactURL string) {
	log.Debug().Str("repo", repo.FullName).Int64("run_id", run.ID).Str("artifact", artifactName).Msg("Downloading artifact with cookie")

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

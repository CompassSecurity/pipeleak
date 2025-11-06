package scan

import (
	"context"
	"net/http"
	"net/url"
	"strings"

	"code.gitea.io/sdk/gitea"
	gounits "github.com/docker/go-units"
	"github.com/CompassSecurity/pipeleak/pkg/httpclient"
	"github.com/CompassSecurity/pipeleak/pkg/logging"
	"github.com/CompassSecurity/pipeleak/pkg/scan/runner"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
)

var scanOptions = GiteaScanOptions{}
var maxArtifactSize string

func NewScanCmd() *cobra.Command {
	scanCmd := &cobra.Command{
		Use:   "scan",
		Short: "Scan Gitea Actions",
		Long: `Scan Gitea Actions workflow runs and artifacts for secrets
### Token Authentication

You can create a personal access token in Gitea by navigating to your user settings, selecting "Applications", and then "Generate New Token". 

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

# Scan a specific repository but limit the number of workflow runs to scan
pipeleak gitea scan --token gitea_token_xxxxx --gitea https://gitea.example.com --repository owner/repo-name --runs-limit 20 --cookie your_cookie_value
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
	scanCmd.PersistentFlags().StringVarP(&maxArtifactSize, "max-artifact-size", "", "500Mb", "Max file size of an artifact to be included in scanning. Larger files are skipped. Format: https://pkg.go.dev/github.com/docker/go-units#FromHumanSize")
	scanCmd.Flags().BoolVarP(&scanOptions.Owned, "owned", "o", false, "Scan only repositories owned by the user")
	scanCmd.Flags().StringVarP(&scanOptions.Organization, "organization", "", "", "Scan all repositories of a specific organization")
	scanCmd.Flags().StringVarP(&scanOptions.Repository, "repository", "r", "", "Scan a specific repository (format: owner/repo)")
	scanCmd.Flags().StringVarP(&scanOptions.Cookie, "cookie", "c", "", "Gitea session cookie (i_like_gitea). Needed when scanning where you are NOT the owner of the repository")
	scanCmd.Flags().IntVarP(&scanOptions.RunsLimit, "runs-limit", "", 0, "Limit the number of workflow runs to scan per repository (0 = unlimited)")
	scanCmd.Flags().Int64VarP(&scanOptions.StartRunID, "start-run-id", "", 0, "Start scanning from a specific run ID (only valid with --repository flag, 0 = start from latest)")
	scanCmd.Flags().StringSliceVarP(&scanOptions.ConfidenceFilter, "confidence", "", []string{}, "Filter for confidence level, separate by comma if multiple. See documentation for more info.")
	scanCmd.PersistentFlags().IntVarP(&scanOptions.MaxScanGoRoutines, "threads", "", 4, "Nr of threads used to scan")
	scanCmd.PersistentFlags().BoolVarP(&scanOptions.TruffleHogVerification, "truffleHogVerification", "", true, "Enable TruffleHog credential verification to actively test found credentials and only report verified ones (enabled by default, disable with --truffleHogVerification=false)")

	return scanCmd
}

func Scan(cmd *cobra.Command, args []string) {
	go logging.ShortcutListeners(scanStatus)

	if scanOptions.StartRunID > 0 && scanOptions.Repository == "" {
		log.Fatal().Msg("--start-run-id can only be used with --repository flag")
	}

	_, err := url.ParseRequestURI(scanOptions.GiteaURL)
	if err != nil {
		log.Fatal().Err(err).Msg("The provided Gitea URL is not a valid URL")
	}

	scanOptions.MaxArtifactSize = parseFileSize(maxArtifactSize)

	scanOptions.Context = context.Background()
	scanOptions.Client, err = gitea.NewClient(scanOptions.GiteaURL, gitea.SetToken(scanOptions.Token))
	if err != nil {
		log.Fatal().Err(err).Msg("Failed creating Gitea client")
	}

	authHeaders := map[string]string{"Authorization": "token " + scanOptions.Token}

	if scanOptions.Cookie != "" {
		scanOptions.HttpClient = httpclient.GetPipeleakHTTPClient(
			scanOptions.GiteaURL,
			[]*http.Cookie{
				{
					Name:   "i_like_gitea",
					Value:  scanOptions.Cookie,
					Path:   "/",
					Domain: "",
				},
			},
			authHeaders,
		)

		validateCookie()
	} else {
		scanOptions.HttpClient = httpclient.GetPipeleakHTTPClient("", nil, authHeaders)

		httpClient := &http.Client{
			Transport: &AuthTransport{
				Base:  http.DefaultTransport,
				Token: scanOptions.Token,
			},
		}

		scanOptions.HttpClient.StandardClient().Transport = httpClient.Transport
	}

	runner.InitScanner(scanOptions.ConfidenceFilter)
	if !scanOptions.TruffleHogVerification {
		log.Info().Msg("TruffleHog verification is disabled")
	}

	scanRepositories(scanOptions.Client)
	log.Info().Msg("Scan Finished, Bye Bye 🏳️‍🌈🔥")
}

func parseFileSize(size string) int64 {
	byteSize, err := gounits.FromHumanSize(size)
	if err != nil {
		log.Fatal().Err(err).Str("size", size).Msg("Failed parsing flag")
	}

	return byteSize
}

func scanStatus() *zerolog.Event {
	return log.Info().Str("status", "scanning... ✨✨ nothing more yet ✨✨")
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
	parts := strings.Split(repoFullName, "/")
	if len(parts) != 2 {
		log.Error().Str("repository", repoFullName).Msg("Invalid repository format, expected owner/repo")
		return
	}

	owner := parts[0]
	repoName := parts[1]

	repo, _, err := client.GetRepo(owner, repoName)
	if err != nil {
		log.Error().Err(err).Str("repository", repoFullName).Msg("failed to get repository")
		return
	}

	if repo == nil {
		log.Error().Str("repository", repoFullName).Msg("repository not found (nil response)")
		return
	}

	log.Info().Str("url", repo.HTMLURL).Msg("Scanning repository")
	scanRepository(client, repo)
}

func scanAllRepositories(client *gitea.Client) {
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
	user, _, err := client.GetMyUserInfo()
	if err != nil {
		log.Error().Err(err).Msg("failed to get user info")
		return
	}

	if user == nil {
		log.Error().Msg("failed to get user info (nil response)")
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
	if repo == nil {
		log.Error().Msg("Cannot scan repository: repository is nil")
		return
	}

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

	if scanOptions.StartRunID > 0 {
		filteredRuns := make([]ActionWorkflowRun, 0)
		for _, run := range workflowRuns {
			if run.ID <= scanOptions.StartRunID {
				filteredRuns = append(filteredRuns, run)
			}
		}
		workflowRuns = filteredRuns

		if len(workflowRuns) == 0 {
			log.Debug().Str("repo", repo.FullName).Int64("start_run_id", scanOptions.StartRunID).Msg("No workflow runs found with ID <= start-run-id")
			return
		}

		log.Info().Str("repo", repo.FullName).Int("runs", len(workflowRuns)).Int64("start_run_id", scanOptions.StartRunID).Msg("Found workflow runs from specified run ID")
	} else {
		log.Info().Str("repo", repo.FullName).Int("runs", len(workflowRuns)).Msg("Found workflow runs")
	}

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

package github

import (
	"context"
	"io"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/CompassSecurity/pipeleak/pkg/format"
	"github.com/CompassSecurity/pipeleak/pkg/httpclient"
	"github.com/CompassSecurity/pipeleak/pkg/logging"
	artifactproc "github.com/CompassSecurity/pipeleak/pkg/scan/artifact"
	"github.com/CompassSecurity/pipeleak/pkg/scan/logline"
	"github.com/CompassSecurity/pipeleak/pkg/scan/result"
	"github.com/CompassSecurity/pipeleak/pkg/scan/runner"
	"github.com/gofri/go-github-ratelimit/v2/github_ratelimit"
	"github.com/gofri/go-github-ratelimit/v2/github_ratelimit/github_primary_ratelimit"
	"github.com/gofri/go-github-ratelimit/v2/github_ratelimit/github_secondary_ratelimit"
	"github.com/google/go-github/v69/github"
	"github.com/hashicorp/go-retryablehttp"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
)

type GitHubScanOptions struct {
	AccessToken            string
	ConfidenceFilter       []string
	MaxScanGoRoutines      int
	TruffleHogVerification bool
	MaxWorkflows           int
	Organization           string
	Owned                  bool
	User                   string
	Public                 bool
	SearchQuery            string
	Artifacts              bool
	GitHubURL              string
	Repo                   string
	MaxArtifactSize        int64
	Context                context.Context
	Client                 *github.Client
	HttpClient             *retryablehttp.Client
}

var options = GitHubScanOptions{}
var maxArtifactSize string

func NewScanCmd() *cobra.Command {
	scanCmd := &cobra.Command{
		Use:   "scan [no options!]",
		Short: "Scan GitHub Actions",
		Long:  `Scan GitHub Actions workflow runs and artifacts for secrets`,
		Example: `
# Scan owned repositories including their artifacts
pipeleak gh scan --token github_pat_xxxxxxxxxxx --artifacts --owned

# Scan repositories of an organization
pipeleak gh scan --token github_pat_xxxxxxxxxxx --artifacts --maxWorkflows 10 --org apache

# Scan public repositories
pipeleak gh scan --token github_pat_xxxxxxxxxxx --artifacts --maxWorkflows 10 --public

# Scan by search term
pipeleak gh scan --token github_pat_xxxxxxxxxxx --artifacts --maxWorkflows 10 --search iac

# Scan repositories of a user
pipeleak gh scan --token github_pat_xxxxxxxxxxx --artifacts --user firefart

# Scan a single repository
pipeleak gh scan --token github_pat_xxxxxxxxxxx --artifacts --repo owner/repo
		`,
		Run: Scan,
	}
	scanCmd.Flags().StringVarP(&options.AccessToken, "token", "t", "", "GitHub Personal Access Token - https://github.com/settings/tokens")
	err := scanCmd.MarkFlagRequired("token")
	if err != nil {
		log.Fatal().Msg("Unable to require token flag")
	}

	scanCmd.Flags().StringSliceVarP(&options.ConfidenceFilter, "confidence", "", []string{}, "Filter for confidence level, separate by comma if multiple. See readme for more info.")
	scanCmd.PersistentFlags().IntVarP(&options.MaxScanGoRoutines, "threads", "", 4, "Nr of threads used to scan")
	scanCmd.PersistentFlags().BoolVarP(&options.TruffleHogVerification, "truffle-hog-verification", "", true, "Enable the TruffleHog credential verification, will actively test the found credentials and only report those. Disable with --truffle-hog-verification=false")
	scanCmd.PersistentFlags().IntVarP(&options.MaxWorkflows, "max-workflows", "", -1, "Max. number of workflows to scan per repository")
	scanCmd.PersistentFlags().BoolVarP(&options.Artifacts, "artifacts", "a", false, "Scan workflow artifacts")
	scanCmd.PersistentFlags().StringVarP(&maxArtifactSize, "max-artifact-size", "", "500Mb", "Max file size of an artifact to be included in scanning. Larger files are skipped. Format: https://pkg.go.dev/github.com/docker/go-units#FromHumanSize")
	scanCmd.Flags().StringVarP(&options.Organization, "org", "", "", "GitHub organization name to scan")
	scanCmd.Flags().StringVarP(&options.User, "user", "", "", "GitHub user name to scan")
	scanCmd.PersistentFlags().BoolVarP(&options.Owned, "owned", "", false, "Scan user onwed projects only")
	scanCmd.PersistentFlags().BoolVarP(&options.Public, "public", "p", false, "Scan all public repositories")
	scanCmd.Flags().StringVarP(&options.SearchQuery, "search", "s", "", "GitHub search query")
	scanCmd.Flags().StringVarP(&options.Repo, "repo", "r", "", "Scan a single repository in the format owner/repo")
	scanCmd.Flags().StringVarP(&options.GitHubURL, "github", "g", "https://api.github.com", "GitHub API base URL")
	scanCmd.MarkFlagsMutuallyExclusive("owned", "org", "user", "public", "search", "repo")

	return scanCmd
}

func Scan(cmd *cobra.Command, args []string) {
	go logging.ShortcutListeners(scanStatus)

	byteSize, err := format.ParseHumanSize(maxArtifactSize)
	if err != nil {
		log.Fatal().Err(err).Str("size", maxArtifactSize).Msg("Failed parsing max-artifact-size flag")
	}
	options.MaxArtifactSize = byteSize

	options.Context = context.WithValue(context.Background(), github.BypassRateLimitCheck, true)
	options.Client = setupClient(options.AccessToken, options.GitHubURL)
	options.HttpClient = httpclient.GetPipeleakHTTPClient("", nil, nil)
	scan(options.Client)
	log.Info().Msg("Scan Finished, Bye Bye 🏳️‍🌈🔥")
}

func setupClient(accessToken string, baseURL string) *github.Client {
	if baseURL == "" {
		baseURL = "https://api.github.com/"
	}
	rateLimiter := github_ratelimit.New(nil,
		github_primary_ratelimit.WithLimitDetectedCallback(func(ctx *github_primary_ratelimit.CallbackContext) {
			resetTime := ctx.ResetTime.Add(time.Duration(time.Second * 30))
			log.Info().Str("category", string(ctx.Category)).Time("reset", resetTime).Msg("Primary rate limit detected, will resume automatically")
			time.Sleep(time.Until(resetTime))
			log.Info().Str("category", string(ctx.Category)).Msg("Resuming")
		}),
		github_secondary_ratelimit.WithLimitDetectedCallback(func(ctx *github_secondary_ratelimit.CallbackContext) {
			resetTime := ctx.ResetTime.Add(time.Duration(time.Second * 30))
			log.Info().Time("reset", *ctx.ResetTime).Dur("totalSleep", *ctx.TotalSleepTime).Msg("Secondary rate limit detected, will resume automatically")
			time.Sleep(time.Until(resetTime))
			log.Info().Msg("Resuming")
		}),
	)

	client := github.NewClient(&http.Client{Transport: rateLimiter}).WithAuthToken(accessToken)
	if baseURL != "https://api.github.com/" {
		client, _ = client.WithEnterpriseURLs(baseURL, baseURL)
	}
	return client
}

func scan(client *github.Client) {
	runner.InitScanner(options.ConfidenceFilter)

	if options.Repo != "" {
		log.Info().Str("repository", options.Repo).Msg("Scanning single repository")
		scanSingleRepository(client, options.Repo)
	} else if options.Owned {
		log.Info().Msg("Scanning authenticated user's owned repositories actions")
		scanRepositories(client)
	} else if options.User != "" {
		log.Info().Str("users", options.User).Msg("Scanning user's repositories actions")
		scanRepositories(client)
	} else if options.SearchQuery != "" {
		log.Info().Str("query", options.SearchQuery).Msg("Searching repositories")
		searchRepositories(client, options.SearchQuery)
	} else if options.Public {
		log.Info().Msg("Scanning most recent public repositories")
		id := identifyNewestPublicProjectId(client)
		scanAllPublicRepositories(client, id)
	} else {
		log.Info().Str("organization", options.Organization).Msg("Scanning organization repositories actions")
		scanRepositories(client)
	}
}

func scanStatus() *zerolog.Event {
	rateLimit, resp, err := options.Client.RateLimit.Get(options.Context)
	if resp == nil {
		return log.Info().Str("rateLimit", "You're rate limited, just wait ✨")
	}

	if err != nil {
		log.Fatal().Stack().Err(err).Msg("Failed fetching rate limit stats")
	}

	return log.Info().Int("coreRateLimitRemaining", rateLimit.Core.Remaining).Time("coreRateLimitReset", rateLimit.Core.Reset.Time).Int("searchRateLimitRemaining", rateLimit.Search.Remaining).Time("searchRateLimitReset", rateLimit.Search.Reset.Time)
}

func searchRepositories(client *github.Client, query string) {
	searchOpt := github.SearchOptions{}
	for {
		searchResults, resp, err := client.Search.Repositories(options.Context, query, &searchOpt)
		if err != nil {
			log.Fatal().Stack().Err(err).Msg("Failed searching repositories")
		}

		for _, repo := range searchResults.Repositories {
			log.Debug().Str("name", *repo.Name).Str("url", *repo.HTMLURL).Msg("Scan")
			iterateWorkflowRuns(client, repo)
		}

		if resp.NextPage == 0 {
			break
		}
		searchOpt.Page = resp.NextPage
	}
}

func scanAllPublicRepositories(client *github.Client, latestProjectId int64) {
	opt := &github.RepositoryListAllOptions{
		Since: latestProjectId,
	}

	// iterating through the repos in reverse must take into account, that missing ids prevent easy pagination as they create holes in the list.
	// thus we keep a temporary cache of the ids of the last 5 pages and check if we alredy scanned the repo id, or skip them.
	tmpIdCache := make(map[int64]struct{})
	pageCounter := 0
	for opt.Since >= 0 {
		if pageCounter > 4 {
			pageCounter = 0
			tmpIdCache = deleteHighestXKeys(tmpIdCache, 100)
		}

		repos, _, err := client.Repositories.ListAll(options.Context, opt)
		if err != nil {
			log.Fatal().Stack().Err(err).Msg("Failed fetching authenticated user repos")
		}

		sort.SliceStable(repos, func(i, j int) bool {
			return *repos[i].ID > *repos[j].ID
		})

		for _, repo := range repos {
			_, ok := tmpIdCache[*repo.ID]
			if ok {
				continue
			} else {
				tmpIdCache[*repo.ID] = struct{}{}
			}

			log.Debug().Str("url", *repo.HTMLURL).Msg("Scan")
			iterateWorkflowRuns(client, repo)
			opt.Since = *repo.ID
		}

		// 100 = page size, ideally no ids miss thus we cannot go higher
		opt.Since = opt.Since - 100
		pageCounter = pageCounter + 1
	}
}

func deleteHighestXKeys(m map[int64]struct{}, nrKeys int) map[int64]struct{} {
	if len(m) < nrKeys {
		return make(map[int64]struct{})
	}

	keys := make([]int64, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}

	sort.Slice(keys, func(i, j int) bool {
		return keys[i] > keys[j]
	})

	for i := 0; i < nrKeys; i++ {
		delete(m, keys[i])
	}
	return m
}

func scanRepositories(client *github.Client) {
	if options.Organization != "" {
		scanOrgRepositories(client, options.Organization)
	} else if options.User != "" {
		scanUserRepositories(client, options.User)
	} else {
		scanAuthenticatedUserRepositories(client, options.Owned)
	}
}

func validateRepoFormat(repo string) (owner, name string, valid bool) {
	parts := strings.Split(repo, "/")
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return "", "", false
	}
	return parts[0], parts[1], true
}

func scanSingleRepository(client *github.Client, repoFullName string) {
	owner, name, valid := validateRepoFormat(repoFullName)
	if !valid {
		log.Fatal().Str("repo", repoFullName).Msg("Invalid repository format. Expected: owner/repo")
	}

	repo, resp, err := client.Repositories.Get(options.Context, owner, name)
	if resp != nil && resp.StatusCode == 404 {
		log.Fatal().Str("repo", repoFullName).Msg("Repository not found")
	}
	if err != nil {
		log.Fatal().Stack().Err(err).Str("repo", repoFullName).Msg("Failed fetching repository")
	}

	log.Debug().Str("name", *repo.Name).Str("url", *repo.HTMLURL).Msg("Scan")
	iterateWorkflowRuns(client, repo)
}

func scanOrgRepositories(client *github.Client, organization string) {
	opt := &github.RepositoryListByOrgOptions{
		Sort:        "updated",
		ListOptions: github.ListOptions{PerPage: 100},
	}
	for {
		repos, resp, err := client.Repositories.ListByOrg(options.Context, organization, opt)
		if err != nil {
			log.Fatal().Stack().Err(err).Msg("Failed fetching organization repos")
		}
		for _, repo := range repos {
			log.Debug().Str("name", *repo.Name).Str("url", *repo.HTMLURL).Msg("Scan")
			iterateWorkflowRuns(client, repo)
		}
		if resp.NextPage == 0 {
			break
		}
		opt.Page = resp.NextPage
	}
}

func scanUserRepositories(client *github.Client, user string) {
	opt := &github.RepositoryListByUserOptions{
		Sort:        "updated",
		ListOptions: github.ListOptions{PerPage: 100},
	}
	for {
		repos, resp, err := client.Repositories.ListByUser(options.Context, user, opt)
		if err != nil {
			log.Fatal().Stack().Err(err).Msg("Failed fetching user repos")
		}
		for _, repo := range repos {
			log.Debug().Str("name", *repo.Name).Str("url", *repo.HTMLURL).Msg("Scan")
			iterateWorkflowRuns(client, repo)
		}
		if resp.NextPage == 0 {
			break
		}
		opt.Page = resp.NextPage
	}
}

func scanAuthenticatedUserRepositories(client *github.Client, owned bool) {
	affiliation := "owner,collaborator,organization_member"
	if owned {
		affiliation = "owner"
	}
	opt := &github.RepositoryListByAuthenticatedUserOptions{
		ListOptions: github.ListOptions{PerPage: 100},
		Affiliation: affiliation,
	}
	for {
		repos, resp, err := client.Repositories.ListByAuthenticatedUser(options.Context, opt)
		if err != nil {
			log.Fatal().Stack().Err(err).Msg("Failed fetching authenticated user repos")
		}
		for _, repo := range repos {
			log.Debug().Str("name", *repo.Name).Str("url", *repo.HTMLURL).Msg("Scan")
			iterateWorkflowRuns(client, repo)
		}
		if resp.NextPage == 0 {
			break
		}
		opt.Page = resp.NextPage
	}
}

func iterateWorkflowRuns(client *github.Client, repo *github.Repository) {
	opt := github.ListWorkflowRunsOptions{
		ListOptions: github.ListOptions{PerPage: 100},
	}
	wfCount := 0
	for {
		workflowRuns, resp, err := client.Actions.ListRepositoryWorkflowRuns(options.Context, *repo.Owner.Login, *repo.Name, &opt)

		if resp == nil {
			log.Trace().Msg("Empty response due to rate limit, resume now<")
			continue
		}

		if resp.StatusCode == 404 {
			return
		}

		if err != nil {
			log.Error().Stack().Err(err).Msg("Failed fetching workflow runs")
			return
		}

		for _, workflowRun := range workflowRuns.WorkflowRuns {
			log.Debug().Str("name", *workflowRun.DisplayTitle).Str("url", *workflowRun.HTMLURL).Msg("Workflow run")
			downloadWorkflowRunLog(client, repo, workflowRun)

			if options.Artifacts {
				listArtifacts(client, workflowRun)
			}

			wfCount = wfCount + 1
			if wfCount >= options.MaxWorkflows && options.MaxWorkflows > 0 {
				log.Debug().Str("name", *workflowRun.DisplayTitle).Str("url", *workflowRun.HTMLURL).Msg("Reached MaxWorkflow runs, skip remaining")
				return
			}
		}

		if resp.NextPage == 0 {
			break
		}
		opt.Page = resp.NextPage
	}
}

func downloadWorkflowRunLog(client *github.Client, repo *github.Repository, workflowRun *github.WorkflowRun) {
	logURL, resp, err := client.Actions.GetWorkflowRunLogs(options.Context, *repo.Owner.Login, *repo.Name, *workflowRun.ID, 5)

	if resp == nil {
		log.Trace().Msg("downloadWorkflowRunLog Empty response")
		return
	}

	// already deleted, skip
	switch resp.StatusCode {
	case 410:
		log.Debug().Str("workflowRunName", *workflowRun.Name).Msg("Skipped expired")
		return
	case 404:
		return
	}

	if err != nil {
		log.Error().Stack().Err(err).Msg("Failed getting workflow run log URL")
		return
	}

	log.Trace().Msg("Downloading run log")
	logs := downloadRunLogZIP(logURL.String())
	log.Trace().Msg("Finished downloading run log")

	logResult, err := logline.ProcessLogs(logs, logline.ProcessOptions{
		MaxGoRoutines:     options.MaxScanGoRoutines,
		VerifyCredentials: options.TruffleHogVerification,
		BuildURL:          *workflowRun.HTMLURL,
	})
	if err != nil {
		log.Debug().Err(err).Str("workflowRun", *workflowRun.HTMLURL).Msg("Failed detecting secrets")
		return
	}

	result.ReportFindings(logResult.Findings, result.ReportOptions{
		LocationURL: *workflowRun.HTMLURL,
	})
	log.Trace().Msg("Finished scannig run log")
}

func downloadRunLogZIP(url string) []byte {
	res, err := options.HttpClient.Get(url)
	logLines := make([]byte, 0)

	if err != nil {
		return logLines
	}

	if res.StatusCode == 200 {
		body, err := io.ReadAll(res.Body)
		if err != nil {
			log.Err(err).Msg("Failed reading response log body")
			return logLines
		}

		zipResult, err := logline.ExtractLogsFromZip(body)
		if err != nil {
			log.Err(err).Msg("Failed extracting logs from zip")
			return logLines
		}

		return zipResult.ExtractedLogs
	}

	return logLines
}

func identifyNewestPublicProjectId(client *github.Client) int64 {
	for {
		listOpts := github.ListOptions{PerPage: 1000}
		events, resp, err := client.Activity.ListEvents(options.Context, &listOpts)
		if err != nil {
			log.Fatal().Stack().Err(err).Msg("Failed fetching activity")
		}
		for _, event := range events {
			eventType := *event.Type
			log.Trace().Str("type", eventType).Msg("Event")
			if eventType == "CreateEvent" {
				repo, _, err := client.Repositories.GetByID(options.Context, *event.Repo.ID)
				if err != nil {
					log.Fatal().Stack().Err(err).Msg("Failed fetching Web URL of latest repo")
				}
				log.Info().Int64("Id", *repo.ID).Str("url", *repo.HTMLURL).Msg("Identified latest public repository")
				return *event.Repo.ID
			}
		}

		if resp.NextPage == 0 {
			break
		}
		listOpts.Page = resp.NextPage
	}

	log.Fatal().Msg("Failed finding a CreateEvent and thus no rerpository id")
	return -1
}

func listArtifacts(client *github.Client, workflowRun *github.WorkflowRun) {
	listOpt := github.ListOptions{PerPage: 100}
	for {
		artifactList, resp, err := client.Actions.ListWorkflowRunArtifacts(options.Context, *workflowRun.Repository.Owner.Login, *workflowRun.Repository.Name, *workflowRun.ID, &listOpt)
		if resp == nil {
			return
		}

		if resp.StatusCode == 404 {
			return
		}

		if err != nil {
			log.Error().Stack().Err(err).Msg("Failed fetching artifacts list")
			return
		}

		for _, artifact := range artifactList.Artifacts {
			log.Debug().Str("name", *artifact.Name).Str("url", *artifact.ArchiveDownloadURL).Msg("Scan")
			analyzeArtifact(client, workflowRun, artifact)
		}

		if resp.NextPage == 0 {
			break
		}
		listOpt.Page = resp.NextPage
	}
}

func analyzeArtifact(client *github.Client, workflowRun *github.WorkflowRun, artifact *github.Artifact) {
	if artifact.SizeInBytes != nil && *artifact.SizeInBytes > options.MaxArtifactSize {
		log.Debug().
			Int64("bytes", *artifact.SizeInBytes).
			Int64("maxBytes", options.MaxArtifactSize).
			Str("name", *artifact.Name).
			Str("url", *workflowRun.HTMLURL).
			Msg("Skipped large artifact")
		return
	}

	url, resp, err := client.Actions.DownloadArtifact(options.Context, *workflowRun.Repository.Owner.Login, *workflowRun.Repository.Name, *artifact.ID, 5)

	if resp == nil {
		log.Trace().Msg("analyzeArtifact Empty response")
		return
	}

	// already deleted, skip
	if resp.StatusCode == 410 {
		log.Debug().Str("workflowRunName", *workflowRun.Name).Msg("Skipped expired artifact")
		return
	}

	if err != nil {
		log.Err(err).Msg("Failed getting artifact download URL")
		return
	}

	res, err := options.HttpClient.Get(url.String())

	if err != nil {
		log.Err(err).Str("workflow", url.String()).Msg("Failed downloading artifacts zip")
		return
	}

	if res.StatusCode == 200 {
		body, err := io.ReadAll(res.Body)
		if err != nil {
			log.Err(err).Msg("Failed reading response log body")
			return
		}

		_, err = artifactproc.ProcessZipArtifact(body, artifactproc.ProcessOptions{
			MaxGoRoutines:     options.MaxScanGoRoutines,
			VerifyCredentials: options.TruffleHogVerification,
			BuildURL:          *workflowRun.HTMLURL,
			ArtifactName:      *workflowRun.Name,
		})
		if err != nil {
			log.Err(err).Str("url", url.String()).Msg("Failed processing artifact zip")
			return
		}
	}
}

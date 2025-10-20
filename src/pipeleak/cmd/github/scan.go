package github

import (
	"archive/zip"
	"bytes"
	"context"
	"io"
	"net/http"
	"sort"
	"time"

	"github.com/CompassSecurity/pipeleak/helper"
	"github.com/CompassSecurity/pipeleak/scanner"
	"github.com/gofri/go-github-ratelimit/v2/github_ratelimit"
	"github.com/gofri/go-github-ratelimit/v2/github_ratelimit/github_primary_ratelimit"
	"github.com/gofri/go-github-ratelimit/v2/github_ratelimit/github_secondary_ratelimit"
	"github.com/google/go-github/v69/github"
	"github.com/h2non/filetype"
	"github.com/hashicorp/go-retryablehttp"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"github.com/wandb/parallel"
)

type GitHubScanOptions struct {
	AccessToken            string
	Verbose                bool
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
	Context                context.Context
	Client                 *github.Client
	HttpClient             *retryablehttp.Client
}

var options = GitHubScanOptions{}

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
		`,
		Run: Scan,
	}
	scanCmd.Flags().StringVarP(&options.AccessToken, "token", "t", "", "GitHub Personal Access Token")
	err := scanCmd.MarkFlagRequired("token")
	if err != nil {
		log.Fatal().Msg("Unable to require token flag")
	}

	scanCmd.Flags().StringSliceVarP(&options.ConfidenceFilter, "confidence", "", []string{}, "Filter for confidence level, separate by comma if multiple. See readme for more info.")
	scanCmd.PersistentFlags().IntVarP(&options.MaxScanGoRoutines, "threads", "", 4, "Nr of threads used to scan")
	scanCmd.PersistentFlags().BoolVarP(&options.TruffleHogVerification, "truffleHogVerification", "", true, "Enable the TruffleHog credential verification, will actively test the found credentials and only report those. Disable with --truffleHogVerification=false")
	scanCmd.PersistentFlags().IntVarP(&options.MaxWorkflows, "maxWorkflows", "", -1, "Max. number of workflows to scan per repository")
	scanCmd.PersistentFlags().BoolVarP(&options.Artifacts, "artifacts", "a", false, "Scan workflow artifacts")
	scanCmd.Flags().StringVarP(&options.Organization, "org", "", "", "GitHub organization name to scan")
	scanCmd.Flags().StringVarP(&options.User, "user", "", "", "GitHub user name to scan")
	scanCmd.PersistentFlags().BoolVarP(&options.Owned, "owned", "", false, "Scan user onwed projects only")
	scanCmd.PersistentFlags().BoolVarP(&options.Public, "public", "p", false, "Scan all public repositories")
	scanCmd.Flags().StringVarP(&options.SearchQuery, "search", "s", "", "GitHub search query")
	scanCmd.MarkFlagsMutuallyExclusive("owned", "org", "user", "public", "search")

	scanCmd.PersistentFlags().BoolVarP(&options.Verbose, "verbose", "v", false, "Verbose logging")

	return scanCmd
}

func Scan(cmd *cobra.Command, args []string) {
	helper.SetLogLevel(options.Verbose)
	go helper.ShortcutListeners(scanStatus)

	options.Context = context.WithValue(context.Background(), github.BypassRateLimitCheck, true)
	options.Client = setupClient(options.AccessToken)
	options.HttpClient = helper.GetPipeleakHTTPClient("", nil, nil)
	scan(options.Client)
	log.Info().Msg("Scan Finished, Bye Bye 🏳️‍🌈🔥")
}

func setupClient(accessToken string) *github.Client {
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

	return github.NewClient(&http.Client{Transport: rateLimiter}).WithAuthToken(accessToken)
}

func scan(client *github.Client) {
	if options.Owned {
		log.Info().Msg("Scanning authenticated user's owned repositories actions")
	} else if options.User != "" {
		log.Info().Str("users", options.User).Msg("Scanning user's repositories actions")
	} else if options.SearchQuery != "" {
		log.Info().Str("query", options.SearchQuery).Msg("Searching repositories")
	} else if options.Public {
		log.Info().Msg("Scanning most recent public repositories")
	} else {
		log.Info().Str("organization", options.Organization).Msg("Scanning organization repositories actions")
	}

	scanner.InitRules(options.ConfidenceFilter)
	if options.Public {
		id := identifyNewestPublicProjectId(client)
		scanAllPublicRepositories(client, id)
	} else if options.SearchQuery != "" {
		searchRepositories(client, options.SearchQuery)
	} else {
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

func listRepositories(client *github.Client, listOpt github.ListOptions, organization string, user string, owned bool) ([]*github.Repository, *github.Response, github.ListOptions) {
	if organization != "" {
		opt := &github.RepositoryListByOrgOptions{
			Sort:        "updated",
			ListOptions: listOpt,
		}
		repos, resp, err := client.Repositories.ListByOrg(options.Context, organization, opt)
		if err != nil {
			log.Fatal().Stack().Err(err).Msg("Failed fetching organization repos")
		}
		return repos, resp, opt.ListOptions

	} else if user != "" {
		opt := &github.RepositoryListByUserOptions{
			Sort:        "updated",
			ListOptions: listOpt,
		}
		repos, resp, err := client.Repositories.ListByUser(options.Context, user, opt)
		if err != nil {
			log.Fatal().Stack().Err(err).Msg("Failed fetching user repos")
		}
		return repos, resp, opt.ListOptions
	} else {
		affiliation := "owner,collaborator,organization_member"
		if owned {
			affiliation = "owner"
		}
		opt := &github.RepositoryListByAuthenticatedUserOptions{
			ListOptions: listOpt,
			Affiliation: affiliation,
		}

		repos, resp, err := client.Repositories.ListByAuthenticatedUser(options.Context, opt)
		if err != nil {
			log.Fatal().Stack().Err(err).Msg("Failed fetching authenticated user repos")
		}

		return repos, resp, opt.ListOptions
	}
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
	listOpt := github.ListOptions{PerPage: 100}
	for {
		repos, resp, listOpt := listRepositories(client, listOpt, options.Organization, options.User, options.Owned)
		for _, repo := range repos {
			log.Debug().Str("name", *repo.Name).Str("url", *repo.HTMLURL).Msg("Scan")
			iterateWorkflowRuns(client, repo)
		}

		if resp.NextPage == 0 {
			break
		}
		listOpt.Page = resp.NextPage
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
	findings, err := scanner.DetectHits(logs, options.MaxScanGoRoutines, options.TruffleHogVerification)
	if err != nil {
		log.Debug().Err(err).Str("workflowRun", *workflowRun.HTMLURL).Msg("Failed detecting secrets")
		return
	}

	for _, finding := range findings {
		log.Warn().Str("confidence", finding.Pattern.Pattern.Confidence).Str("ruleName", finding.Pattern.Pattern.Name).Str("value", finding.Text).Str("workflowRun", *workflowRun.HTMLURL).Msg("HIT")
	}
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

		zipReader, err := zip.NewReader(bytes.NewReader(body), int64(len(body)))
		if err != nil {
			log.Err(err).Msg("Failed creating zip reader")
			return logLines
		}

		for _, zipFile := range zipReader.File {
			log.Trace().Str("zipFile", zipFile.Name).Msg("Zip file")
			unzippedFileBytes, err := readZipFile(zipFile)
			if err != nil {
				log.Err(err).Msg("Failed reading zip file")
				continue
			}

			logLines = append(logLines, unzippedFileBytes...)
		}
	}

	return logLines
}

func readZipFile(zf *zip.File) ([]byte, error) {
	f, err := zf.Open()
	if err != nil {
		return nil, err
	}
	defer func() { _ = f.Close() }()
	return io.ReadAll(f)
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
		zipListing, err := zip.NewReader(bytes.NewReader(body), int64(len(body)))
		if err != nil {
			log.Err(err).Str("url", url.String()).Msg("Failed creating zip reader")
			return
		}

		ctx := options.Context
		group := parallel.Limited(ctx, options.MaxScanGoRoutines)
		for _, file := range zipListing.File {
			group.Go(func(ctx context.Context) {
				fc, err := file.Open()
				if err != nil {
					log.Error().Stack().Err(err).Msg("Unable to open raw artifact zip file")
					return
				}

				content, err := io.ReadAll(fc)
				if err != nil {
					log.Error().Stack().Err(err).Msg("Unable to readAll artifact zip file")
					return
				}

				kind, _ := filetype.Match(content)
				// do not scan https://pkg.go.dev/github.com/h2non/filetype#readme-supported-types
				if kind == filetype.Unknown {
					scanner.DetectFileHits(content, *workflowRun.HTMLURL, *workflowRun.Name, file.Name, "", options.TruffleHogVerification)
				} else if filetype.IsArchive(content) {
					scanner.HandleArchiveArtifact(file.Name, content, *workflowRun.HTMLURL, *workflowRun.Name, options.TruffleHogVerification)
				}
				_ = fc.Close()
			})
		}

		group.Wait()
	}
}

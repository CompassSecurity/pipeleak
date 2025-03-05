package github

import (
	"archive/zip"
	"bytes"
	"context"
	"io"

	"github.com/CompassSecurity/pipeleak/helper"
	"github.com/CompassSecurity/pipeleak/scanner"
	"github.com/google/go-github/v69/github"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
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
}

var options = GitHubScanOptions{}

func NewScanCmd() *cobra.Command {
	scanCmd := &cobra.Command{
		Use:   "scan [no options!]",
		Short: "Scan GitHub Actions",
		Run:   Scan,
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
	scanCmd.Flags().StringVarP(&options.Organization, "org", "", "", "GitHub organization name to scan")
	scanCmd.Flags().StringVarP(&options.User, "user", "", "", "GitHub user name to scan")
	scanCmd.PersistentFlags().BoolVarP(&options.Owned, "owned", "", false, "Scan user onwed projects only")
	scanCmd.MarkFlagsMutuallyExclusive("owned", "org", "user")
	scanCmd.PersistentFlags().BoolVarP(&options.Verbose, "verbose", "v", false, "Verbose logging")

	return scanCmd
}

func Scan(cmd *cobra.Command, args []string) {
	helper.SetLogLevel(options.Verbose)
	// @todo this is buggy, does not refresh
	go helper.ShortcutListeners(0, 0)

	if options.Owned {
		log.Info().Msg("Scanning authenticated user's owned repositories actions")
	} else if options.User != "" {
		log.Info().Str("users", options.User).Msg("Scanning user's repositories actions")
	} else {
		log.Info().Str("organization", options.Organization).Msg("Scanning current authenticated user's repositories actions")
	}

	client := github.NewClient(nil).WithAuthToken(options.AccessToken)
	scanner.InitRules(options.ConfidenceFilter)
	id := identifyNewestPublicProjectId(client)
	scanAllPublicRepositories(client, id)
	log.Info().Msg("Scan Finished, Bye Bye 🏳️‍🌈🔥")
}

func listRepositories(client *github.Client, listOpt github.ListOptions, organization string, user string, owned bool) ([]*github.Repository, *github.Response, github.ListOptions) {
	if organization != "" {
		opt := &github.RepositoryListByOrgOptions{
			Sort:        "updated",
			ListOptions: listOpt,
		}
		repos, resp, err := client.Repositories.ListByOrg(context.Background(), organization, opt)
		if err != nil {
			log.Fatal().Stack().Err(err).Msg("Failed fetching organization repos")
		}
		return repos, resp, opt.ListOptions

	} else if user != "" {
		opt := &github.RepositoryListByUserOptions{
			Sort:        "updated",
			ListOptions: listOpt,
		}
		repos, resp, err := client.Repositories.ListByUser(context.Background(), user, opt)
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

		repos, resp, err := client.Repositories.ListByAuthenticatedUser(context.Background(), opt)
		if err != nil {
			log.Fatal().Stack().Err(err).Msg("Failed fetching authenticated user repos")
		}

		return repos, resp, opt.ListOptions
	}
}

func scanAllPublicRepositories(client *github.Client, latestProjectId int64) {
	opt := &github.RepositoryListAllOptions{
		// 100 = page size
		Since: latestProjectId - 100,
	}

	for {
		if opt.Since < 0 {
			break
		}

		repos, _, err := client.Repositories.ListAll(context.Background(), opt)
		if err != nil {
			log.Fatal().Stack().Err(err).Msg("Failed fetching authenticated user repos")
		}

		for _, repo := range repos {
			log.Info().Int64("id", *repo.ID).Str("owner", *repo.Owner.Login).Str("name", *repo.Name).Str("url", *repo.HTMLURL).Msg("Scan")
			iterateWorkflowRuns(client, repo)
		}

		opt.Since = opt.Since - 100
		log.Error().Int64("page", opt.Since).Msg("hacker")
	}
}

func scanGithubActions(client *github.Client) {
	listOpt := github.ListOptions{PerPage: 100}
	for {
		repos, resp, listOpt := listRepositories(client, listOpt, options.Organization, options.User, options.Owned)
		for _, repo := range repos {
			log.Info().Str("name", *repo.Name).Msg("Scanning Repository")
			iterateWorkflowRuns(client, repo)
		}

		if resp.NextPage == 0 {
			break
		}
		listOpt.Page = resp.NextPage
	}
}

func iterateWorkflowRuns(client *github.Client, repo *github.Repository) {
	opt := &github.ListWorkflowRunsOptions{
		ListOptions: github.ListOptions{PerPage: 1000},
	}
	wfCount := 0
	for {
		workflowRuns, resp, err := client.Actions.ListRepositoryWorkflowRuns(context.Background(), *repo.Owner.Login, *repo.Name, opt)
		if err != nil {
			log.Error().Stack().Err(err).Msg("Failed Fetching Workflow Runs")
			return
		}

		for _, workflowRun := range workflowRuns.WorkflowRuns {
			log.Debug().Str("name", *workflowRun.DisplayTitle).Str("repo", *repo.HTMLURL).Msg("Workflow Run")
			downloadWorkflowRunLog(client, repo, workflowRun)

			wfCount = wfCount + 1
			if wfCount > options.MaxWorkflows && options.MaxWorkflows > 0 {
				log.Debug().Str("name", *workflowRun.DisplayTitle).Str("repo", *repo.HTMLURL).Msg("Reached MaxWorkflow runs, skip remaining")
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
	logURL, resp, err := client.Actions.GetWorkflowRunLogs(context.Background(), *repo.Owner.Login, *repo.Name, *workflowRun.ID, 5)

	// already deleted, skip
	if resp.StatusCode == 410 {
		log.Debug().Str("workflowRunName", *workflowRun.Name).Msg("Skipped expired")
		return
	}

	if err != nil {
		log.Error().Stack().Err(err).Msg("Failed Getting Workflow Run Log URL")
		return
	}

	logs := downloadRunLogZIP(logURL.String())
	findings := scanner.DetectHits(logs, options.MaxScanGoRoutines, options.TruffleHogVerification)
	for _, finding := range findings {
		log.Warn().Str("confidence", finding.Pattern.Pattern.Confidence).Str("ruleName", finding.Pattern.Pattern.Name).Str("value", finding.Text).Str("workflowRun", *workflowRun.HTMLURL).Msg("HIT")
	}
}

func downloadRunLogZIP(url string) []byte {
	client := helper.GetNonVerifyingHTTPClient()
	res, err := client.Get(url)
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
	defer f.Close()
	return io.ReadAll(f)
}

func identifyNewestPublicProjectId(client *github.Client) int64 {
	for {
		listOpts := github.ListOptions{PerPage: 1000}
		events, resp, err := client.Activity.ListEvents(context.Background(), &listOpts)
		if err != nil {
			log.Fatal().Stack().Err(err).Msg("Failed fetching activity")
		}
		for _, event := range events {
			eventType := *event.Type
			log.Trace().Str("type", eventType).Msg("Event")
			if eventType == "CreateEvent" {
				repo, _, err := client.Repositories.GetByID(context.Background(), *event.Repo.ID)
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

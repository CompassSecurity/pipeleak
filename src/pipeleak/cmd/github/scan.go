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
}

var options = GitHubScanOptions{}

func NewScanCmd() *cobra.Command {
	scanCmd := &cobra.Command{
		Use:   "scan [no options!]",
		Short: "Scan GitHub Actions",
		Run:   Scan,
	}
	scanCmd.Flags().StringVarP(&options.AccessToken, "token", "t", "", "GitHub Peronsal Access Token")
	err := scanCmd.MarkFlagRequired("token")
	if err != nil {
		log.Fatal().Msg("Unable to require token flag")
	}

	scanCmd.Flags().StringSliceVarP(&options.ConfidenceFilter, "confidence", "", []string{}, "Filter for confidence level, separate by comma if multiple. See readme for more info.")
	scanCmd.PersistentFlags().IntVarP(&options.MaxScanGoRoutines, "threads", "", 4, "Nr of threads used to scan")
	scanCmd.PersistentFlags().BoolVarP(&options.TruffleHogVerification, "truffleHogVerification", "", true, "Enable the TruffleHog credential verification, will actively test the found credentials and only report those. Disable with --truffleHogVerification=false")
	scanCmd.PersistentFlags().BoolVarP(&options.Verbose, "verbose", "v", false, "Verbose logging")

	return scanCmd
}

func Scan(cmd *cobra.Command, args []string) {
	helper.SetLogLevel(options.Verbose)
	// @todo this is buggy, does not refresh
	go helper.ShortcutListeners(0, 0)

	client := github.NewClient(nil).WithAuthToken(options.AccessToken)
	scanner.InitRules(options.ConfidenceFilter)
	ScanGithubActions(client)
	log.Info().Msg("Scan Finished, Bye Bye 🏳️‍🌈🔥")
}

func ScanGithubActions(client *github.Client) {
	log.Info().Msg("Scanning GitHub Actions")
	opt := &github.RepositoryListByAuthenticatedUserOptions{
		ListOptions: github.ListOptions{PerPage: 100},
		Affiliation: "owner",
	}
	for {
		repos, resp, err := client.Repositories.ListByAuthenticatedUser(context.Background(), opt)
		if err != nil {
			log.Fatal().Stack().Err(err).Msg("Failed Fetching Repos")
		}

		log.Info().Int("len", len(repos)).Msg("test")

		for _, repo := range repos {
			log.Info().Str("name", *repo.Name).Msg("Scanning Repository")
			iterateWorkflowRuns(client, repo)
		}

		if resp.NextPage == 0 {
			break
		}
		opt.Page = resp.NextPage
	}
}

func iterateWorkflowRuns(client *github.Client, repo *github.Repository) {
	opt := &github.ListWorkflowRunsOptions{
		ListOptions: github.ListOptions{PerPage: 1000},
	}
	for {
		workflowRuns, resp, err := client.Actions.ListRepositoryWorkflowRuns(context.Background(), *repo.Owner.Login, *repo.Name, opt)
		if err != nil {
			log.Fatal().Stack().Err(err).Msg("Failed Fetching Workflow Runs")
		}

		for _, workflowRun := range workflowRuns.WorkflowRuns {
			log.Info().Str("name", *workflowRun.DisplayTitle).Msg("Workflow Run")
			downloadWorkflowRunLog(client, repo, workflowRun)
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
		log.Debug().Str("workflowRunId", *workflowRun.Name).Msg("Skipped expired")
		return
	}

	if err != nil {
		log.Fatal().Stack().Err(err).Msg("Failed Getting Workflow Run Log URL")
	}

	logs := downloadRunLogZIP(logURL.String())
	findings := scanner.DetectHits(logs, options.MaxScanGoRoutines, options.TruffleHogVerification)
	for _, finding := range findings {
		log.Warn().Str("confidence", finding.Pattern.Pattern.Confidence).Str("ruleName", finding.Pattern.Pattern.Name).Str("value", finding.Text).Str("workflowRun", *workflowRun.WorkflowURL).Msg("HIT")
	}
}

func downloadRunLogZIP(url string) []byte {
	client := helper.GetNonVerifyingHTTPClient()
	res, err := client.Get(url)
	logLines := make([]byte, 0, 0)

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

		// Read all the files from zip archive
		for _, zipFile := range zipReader.File {
			log.Debug().Str("zipFile", zipFile.Name).Msg("Zip file")
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

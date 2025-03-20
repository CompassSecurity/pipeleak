package bitbucket

import (
	"context"

	"github.com/CompassSecurity/pipeleak/helper"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
)

type GitHubScanOptions struct {
	Username               string
	AccessToken            string
	Verbose                bool
	ConfidenceFilter       []string
	MaxScanGoRoutines      int
	TruffleHogVerification bool
	MaxWorkflows           int
	Workspace              string
	Owned                  bool
	Public                 bool
	SearchQuery            string
	Artifacts              bool
	Context                context.Context
	Client                 BitBucketApiClient
}

var options = GitHubScanOptions{}

func NewScanCmd() *cobra.Command {
	scanCmd := &cobra.Command{
		Use:   "scan [no options!]",
		Short: "Scan GitHub Actions",
		Run:   Scan,
	}
	scanCmd.Flags().StringVarP(&options.AccessToken, "token", "t", "", "Bitbucket Application Password")
	scanCmd.Flags().StringVarP(&options.Username, "username", "u", "", "Bitbucket Username")
	scanCmd.MarkFlagsRequiredTogether("token", "username")

	scanCmd.Flags().StringSliceVarP(&options.ConfidenceFilter, "confidence", "", []string{}, "Filter for confidence level, separate by comma if multiple. See readme for more info.")
	scanCmd.PersistentFlags().IntVarP(&options.MaxScanGoRoutines, "threads", "", 4, "Nr of threads used to scan")
	scanCmd.PersistentFlags().BoolVarP(&options.TruffleHogVerification, "truffleHogVerification", "", true, "Enable the TruffleHog credential verification, will actively test the found credentials and only report those. Disable with --truffleHogVerification=false")
	scanCmd.PersistentFlags().IntVarP(&options.MaxWorkflows, "maxWorkflows", "", -1, "Max. number of workflows to scan per repository")
	scanCmd.PersistentFlags().BoolVarP(&options.Artifacts, "artifacts", "a", false, "Scan workflow artifacts")
	scanCmd.Flags().StringVarP(&options.Workspace, "workspace", "w", "", "Workspace name to scan")
	scanCmd.PersistentFlags().BoolVarP(&options.Owned, "owned", "", false, "Scan user onwed projects only")
	scanCmd.PersistentFlags().BoolVarP(&options.Public, "public", "p", false, "Scan all public repositories")
	scanCmd.Flags().StringVarP(&options.SearchQuery, "search", "s", "", "GitHub search query")

	scanCmd.PersistentFlags().BoolVarP(&options.Verbose, "verbose", "v", false, "Verbose logging")

	return scanCmd
}

func Scan(cmd *cobra.Command, args []string) {
	helper.SetLogLevel(options.Verbose)
	go helper.ShortcutListeners(scanStatus)

	options.Context = context.Background()
	options.Client = NewClient(options.Username, options.AccessToken)

	if options.Public {
		log.Info().Msg("Scanning public repos")
		scanPublic(options.Client)
	} else if options.Owned {
		log.Info().Msg("Scanning owned workspaces")
		scanOwned(options.Client, options.Workspace)

	}
	log.Info().Msg("Scan Finished, Bye Bye 🏳️‍🌈🔥")
}

func scanOwned(client BitBucketApiClient, owner string) {
	workspaces, _, err := client.ListOwnedWorkspaces()
	if err != nil {
		log.Error().Err(err).Msg("Failed fetching workspaces")
	}

	for _, workspace := range workspaces {
		log.Info().Str("name", workspace.Name).Msg("Workspace")
		listWorkspaceRepos(client, workspace.Slug)
	}
}

func listWorkspaceRepos(client BitBucketApiClient, workspaceSlug string) {
	repos, _, err := client.ListWorkspaceRepositoires(workspaceSlug)
	if err != nil {
		log.Error().Err(err).Msg("Failed fetching workspace repos")
	}

	for _, repo := range repos {
		log.Info().Str("name", repo.Name).Msg("Repo")
		listRepoPipelines(client, workspaceSlug, repo.Name)
	}
}

func listRepoPipelines(client BitBucketApiClient, workspaceSlug string, repoSlug string) {
	pipelines, _, err := client.ListRepositoryPipelines(workspaceSlug, repoSlug)
	if err != nil {
		log.Error().Err(err).Msg("Failed fetching repo pipelines")
	}

	for _, pipeline := range pipelines {
		log.Info().Int("buildNr", pipeline.BuildNumber).Msg("Pipeline")
		listPipelineSteps(client, workspaceSlug, repoSlug, pipeline.UUID)
	}
}

func listPipelineSteps(client BitBucketApiClient, workspaceSlug string, repoSlug string, pipelineUuid string) {
	steps, _, err := client.ListPipelineSteps(workspaceSlug, repoSlug, pipelineUuid)
	if err != nil {
		log.Error().Err(err).Msg("Failed fetching pipeline steps")
	}

	for _, step := range steps {
		log.Info().Str("step", step.UUID).Msg("Step")
		getSteplog(client, workspaceSlug, repoSlug, pipelineUuid, step.UUID)
	}
}

func getSteplog(client BitBucketApiClient, workspaceSlug string, repoSlug string, pipelineUuid string, stepUUID string) {
	logBytes, _, err := client.GetStepLog(workspaceSlug, repoSlug, pipelineUuid, stepUUID)
	if err != nil {
		log.Error().Err(err).Msg("Failed fetching pipeline steps")
	}

	log.Info().Bytes("by", logBytes).Msg("data")
}

func scanPublic(client BitBucketApiClient) {
	log.Info().Msg("public")
	return
}

func scanStatus() *zerolog.Event {
	return log.Info().Str("debug", "test")
}

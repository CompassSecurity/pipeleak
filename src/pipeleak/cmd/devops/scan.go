package devops

import (
	"context"

	"github.com/CompassSecurity/pipeleak/helper"
	"github.com/CompassSecurity/pipeleak/scanner"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
)

type DevOpsScanOptions struct {
	Username               string
	AccessToken            string
	Verbose                bool
	ConfidenceFilter       []string
	MaxScanGoRoutines      int
	TruffleHogVerification bool
	MaxPipelines           int
	Organization           string
	//Owned                  bool
	//Public                 bool
	//After                  string
	//SearchQuery            string
	Artifacts bool
	Context   context.Context
	Client    AzureDevOpsApiClient
}

var options = DevOpsScanOptions{}

func NewScanCmd() *cobra.Command {
	scanCmd := &cobra.Command{
		Use:   "scan [no options!]",
		Short: "Scan DevOps Actions",
		Run:   Scan,
	}
	scanCmd.Flags().StringVarP(&options.AccessToken, "token", "t", "", "Azure DevOps Personal Access Token - https://dev.azure.com/{yourUsername}/_usersSettings/tokens")
	scanCmd.MarkFlagRequired("token")
	scanCmd.Flags().StringVarP(&options.Username, "username", "u", "", "Username")
	scanCmd.MarkFlagRequired("username")
	scanCmd.MarkFlagsRequiredTogether("token", "username")

	scanCmd.Flags().StringSliceVarP(&options.ConfidenceFilter, "confidence", "", []string{}, "Filter for confidence level, separate by comma if multiple. See readme for more info.")
	scanCmd.PersistentFlags().IntVarP(&options.MaxScanGoRoutines, "threads", "", 4, "Nr of threads used to scan")
	scanCmd.PersistentFlags().BoolVarP(&options.TruffleHogVerification, "truffleHogVerification", "", true, "Enable the TruffleHog credential verification, will actively test the found credentials and only report those. Disable with --truffleHogVerification=false")
	scanCmd.PersistentFlags().IntVarP(&options.MaxPipelines, "maxPipelines", "", -1, "Max. number of pipelines to scan per repository")
	scanCmd.PersistentFlags().BoolVarP(&options.Artifacts, "artifacts", "a", false, "Scan workflow artifacts")
	scanCmd.Flags().StringVarP(&options.Organization, "organization", "o", "", "Organization name to scan")
	//scanCmd.PersistentFlags().BoolVarP(&options.Owned, "owned", "", false, "Scan user onwed projects only")
	//scanCmd.PersistentFlags().BoolVarP(&options.Public, "public", "p", false, "Scan all public repositories")
	//scanCmd.PersistentFlags().StringVarP(&options.After, "after", "", "", "Filter public repos by a given date in ISO 8601 format: 2025-04-02T15:00:00+02:00 ")
	//scanCmd.Flags().StringVarP(&options.SearchQuery, "search", "s", "", "DevOps search query")

	scanCmd.PersistentFlags().BoolVarP(&options.Verbose, "verbose", "v", false, "Verbose logging")

	return scanCmd
}

func Scan(cmd *cobra.Command, args []string) {
	helper.SetLogLevel(options.Verbose)
	go helper.ShortcutListeners(scanStatus)

	scanner.InitRules(options.ConfidenceFilter)

	options.Context = context.Background()
	options.Client = NewClient(options.Username, options.AccessToken)

	scanOrganization(options.Client, options.Organization)
	log.Info().Msg("Scan Finished, Bye Bye 🏳️‍🌈🔥")
}

func scanOrganization(client AzureDevOpsApiClient, organization string) {

	user, _, err := client.GetAuthenticatedUser()
	if err != nil {
		log.Error().Err(err).Msg("Failed fetching authenticated user")
	}

	log.Info().Str("displayName", user.DisplayName).Msg("Authenticated User")

	// @todo paging
	accounts, _, err := client.ListAccounts(user.ID)
	if err != nil {
		log.Fatal().Err(err).Str("userId", user.ID).Msg("Failed fetching accounts")
	}

	for _, account := range accounts {
		log.Debug().Str("name", account.AccountName).Msg("Scanning Account")

		projects, _, err := client.ListProjects(account.AccountName)
		if err != nil {
			log.Error().Err(err).Str("account", account.AccountName).Msg("Failed fetching projects")
		}

		for _, project := range projects {
			pipelines, _, err := client.ListPipelines(account.AccountName, project.Name)
			if err != nil {
				log.Error().Err(err).Str("account", account.AccountName).Str("project", project.Name).Msg("Failed fetching pipelines")
			}

			for _, pipeline := range pipelines {
				log.Debug().Str("url", pipeline.Links.Web.Href).Msg("Pipeline")

				runs, _, err := client.ListPipelineRuns(account.AccountName, project.Name, pipeline.ID)
				if err != nil {
					log.Error().Err(err).Str("account", account.AccountName).Str("project", project.Name).Int("pipeline", pipeline.ID).Msg("Failed fetching pipeline runs")
				}

				for _, run := range runs {
					log.Debug().Str("url", run.Links.Web.Href).Msg("Pipeline run")

					logs, _, err := client.ListRunLogs(account.AccountName, project.Name, pipeline.ID, run.ID)
					if err != nil {
						log.Error().Err(err).Str("account", account.AccountName).Str("project", project.Name).Int("pipeline", pipeline.ID).Int("run", run.ID).Msg("Failed fetching pipeline run log")
					}

					for _, lg := range logs {
						log.Debug().Str("url", run.Links.Web.Href).Any("sadf", lg).Msg("Pipeline run log")

						logLines, _, err := client.GetLog(account.AccountName, project.Name, pipeline.ID, run.ID, lg.ID)
						if err != nil {
							log.Error().Err(err).Str("account", account.AccountName).Str("project", project.Name).Int("pipeline", pipeline.ID).Int("run", run.ID).Msg("Failed fetching pipeline run log")
						}
						log.Warn().Str("logs", string(logLines)).Any("sadf", lg).Msg("LLOG")
					}
				}
			}
		}
	}
}

func getRepoWebUrl(organization string, repo string) string {
	return "https://dev.azure.com/" + organization + "/" + repo
}

func scanStatus() *zerolog.Event {
	return log.Info().Str("debug", "nothing to show ✔️")
}

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
	scanCmd.MarkFlagRequired("organization")
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

// notes: https://dev.azure.com/PowerShell/PowerShell/_apis/pipelines
func scanOrganization(client AzureDevOpsApiClient, organization string) {
	for {
		repos, _, err := client.ListRepositories(organization)
		if err != nil {
			log.Error().Err(err).Str("organization", organization).Msg("Failed fetching repositories")
		}

		for _, repo := range repos {
			log.Debug().Str("url", getRepoWebUrl(organization, repo.Name)).Msg("Repository")
		}
	}
}

func getRepoWebUrl(organization string, repo string) string {
	return "https://dev.azure.com/" + organization + "/" + repo
}

func scanStatus() *zerolog.Event {
	return log.Info().Str("debug", "nothing to show ✔️")
}

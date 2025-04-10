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

	scanAuthenticatedUser(options.Client, options.Organization)
	log.Info().Msg("Scan Finished, Bye Bye 🏳️‍🌈🔥")
}

func scanAuthenticatedUser(client AzureDevOpsApiClient, organization string) {

	user, _, err := client.GetAuthenticatedUser()
	if err != nil {
		log.Error().Err(err).Msg("Failed fetching authenticated user")
	}

	log.Info().Str("displayName", user.DisplayName).Msg("Authenticated User")
	listAccounts(client, user.ID)
}

func listAccounts(client AzureDevOpsApiClient, userId string) {
	accounts, _, err := client.ListAccounts(userId)
	if err != nil {
		log.Fatal().Err(err).Str("userId", userId).Msg("Failed fetching accounts")
	}

	for _, account := range accounts {
		log.Debug().Str("name", account.AccountName).Msg("Scanning Account")
		listProjects(client, account.AccountName)
	}
}

func listProjects(client AzureDevOpsApiClient, organization string) {
	projects, _, err := client.ListProjects(organization)
	if err != nil {
		log.Fatal().Err(err).Str("organization", organization).Msg("Failed fetching projects")
	}

	for _, project := range projects {
		listBuilds(client, organization, project.Name)
	}
}

func listBuilds(client AzureDevOpsApiClient, organization string, project string) {
	builds, _, err := client.ListBuilds(organization, project)
	if err != nil {
		log.Error().Err(err).Str("organization", organization).Str("project", project).Msg("Failed fetching builds")
	}

	for _, build := range builds {
		log.Trace().Str("url", build.Links.Web.Href).Msg("Build")
		listLogs(client, organization, project, build.ID, build.Links.Web.Href)
	}
}

func listLogs(client AzureDevOpsApiClient, organization string, project string, buildId int, buildWebUrl string) {
	logs, _, err := client.ListBuildLogs(organization, project, buildId)
	if err != nil {
		log.Error().Err(err).Str("organization", organization).Str("project", project).Int("build", buildId).Msg("Failed fetching build logs")
	}

	for _, logEntry := range logs {
		logLines, _, err := client.GetLog(organization, project, buildId, logEntry.ID)
		if err != nil {
			log.Error().Err(err).Str("organization", organization).Str("project", project).Int("build", buildId).Int("logId", logEntry.ID).Msg("Failed fetching build log lines")
		}

		scanLogLines(logLines, buildWebUrl)
	}
}

func scanLogLines(logs []byte, buildWebUrl string) {
	findings := scanner.DetectHits(logs, options.MaxScanGoRoutines, options.TruffleHogVerification)
	for _, finding := range findings {
		log.Warn().Str("confidence", finding.Pattern.Pattern.Confidence).Str("ruleName", finding.Pattern.Pattern.Name).Str("value", finding.Text).Str("url", buildWebUrl).Msg("HIT")
	}
}

func scanStatus() *zerolog.Event {
	return log.Info().Str("debug", "nothing to show ✔️")
}

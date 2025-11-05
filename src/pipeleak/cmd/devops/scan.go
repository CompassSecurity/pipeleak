package devops

import (
	"context"

	"github.com/CompassSecurity/pipeleak/pkg/logging"
	artifactproc "github.com/CompassSecurity/pipeleak/pkg/scan/artifact"
	"github.com/CompassSecurity/pipeleak/pkg/scan/logline"
	"github.com/CompassSecurity/pipeleak/pkg/scan/result"
	"github.com/CompassSecurity/pipeleak/pkg/scan/runner"
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
	MaxBuilds              int
	Organization           string
	Project                string
	Artifacts              bool
	DevOpsURL              string
	Context                context.Context
	Client                 AzureDevOpsApiClient
}

var options = DevOpsScanOptions{}

func NewScanCmd() *cobra.Command {
	scanCmd := &cobra.Command{
		Use:   "scan [no options!]",
		Short: "Scan Azure DevOps Actions",
		Long: `Scan Azure DevOps pipelines for secrets in logs and artifacts.

### Authentication
Create your personal access token here: https://dev.azure.com/{yourproject}/_usersSettings/tokens

> In the top right corner you can choose the scope (Global, Project etc.). 
> Global in that case means per tenant. If you have access to multiple tentants you need to run a scan per tenant.
> Create a read-only token with all scopes (click show all scopes), select the correct organization(s) and then generate the token.
> Get you username from an HTTPS git clone url from the UI.
		`,
		Example: `
# Scan all pipelines the current user has access to
pipeleak ad scan --token xxxxxxxxxxx --username auser --artifacts

# Scan all pipelines of an organization
pipeleak ad scan --token xxxxxxxxxxx --username auser --artifacts --organization myOrganization

# Scan all pipelines of a project e.g. https://dev.azure.com/PowerShell/PowerShell
pipeleak ad scan --token xxxxxxxxxxx --username auser --artifacts --organization powershell --project PowerShell
		`,
		Run: Scan,
	}
	scanCmd.Flags().StringVarP(&options.AccessToken, "token", "t", "", "Azure DevOps Personal Access Token - https://dev.azure.com/{yourUsername}/_usersSettings/tokens")
	err := scanCmd.MarkFlagRequired("token")
	if err != nil {
		log.Fatal().Err(err).Msg("Failed marking token required")
	}
	scanCmd.Flags().StringVarP(&options.Username, "username", "u", "", "Username")
	err = scanCmd.MarkFlagRequired("username")
	if err != nil {
		log.Fatal().Err(err).Msg("Failed marking username required")
	}
	scanCmd.MarkFlagsRequiredTogether("token", "username")

	scanCmd.Flags().StringSliceVarP(&options.ConfidenceFilter, "confidence", "", []string{}, "Filter for confidence level, separate by comma if multiple. See readme for more info.")
	scanCmd.PersistentFlags().IntVarP(&options.MaxScanGoRoutines, "threads", "", 4, "Nr of threads used to scan")
	scanCmd.PersistentFlags().BoolVarP(&options.TruffleHogVerification, "truffleHogVerification", "", true, "Enable the TruffleHog credential verification, will actively test the found credentials and only report those. Disable with --truffleHogVerification=false")
	scanCmd.PersistentFlags().IntVarP(&options.MaxBuilds, "maxBuilds", "", -1, "Max. number of builds to scan per project")
	scanCmd.PersistentFlags().BoolVarP(&options.Artifacts, "artifacts", "a", false, "Scan workflow artifacts")
	scanCmd.Flags().StringVarP(&options.Organization, "organization", "o", "", "Organization name to scan")
	scanCmd.Flags().StringVarP(&options.Project, "project", "p", "", "Project name to scan - can be combined with organization")
	scanCmd.Flags().StringVarP(&options.DevOpsURL, "devops", "d", "https://dev.azure.com", "Azure DevOps base URL")

	scanCmd.PersistentFlags().BoolVarP(&options.Verbose, "verbose", "v", false, "Verbose logging")

	return scanCmd
}

func Scan(cmd *cobra.Command, args []string) {
	logging.SetLogLevel(options.Verbose)
	go logging.ShortcutListeners(scanStatus)

	runner.InitScanner(options.ConfidenceFilter)

	options.Context = context.Background()
	options.Client = NewClient(options.Username, options.AccessToken, options.DevOpsURL)

	if options.Organization == "" && options.Project == "" {
		scanAuthenticatedUser(options.Client)
	} else if options.Organization != "" && options.Project == "" {
		scanOrganization(options.Client, options.Organization)
	} else if options.Organization != "" && options.Project != "" {
		scanProject(options.Client, options.Organization, options.Project)
	}

	log.Info().Msg("Scan Finished, Bye Bye 🏳️‍🌈🔥")
}

func scanAuthenticatedUser(client AzureDevOpsApiClient) {
	log.Info().Msg("Scanning authenticated user")

	user, _, err := client.GetAuthenticatedUser()
	if err != nil {
		log.Error().Err(err).Msg("Failed fetching authenticated user")
	}

	log.Info().Str("displayName", user.DisplayName).Msg("Authenticated User")
	listAccounts(client, user.ID)
}

func scanOrganization(client AzureDevOpsApiClient, organization string) {
	log.Info().Str("organization", organization).Msg("Scanning organization")
	listProjects(client, organization)
}

func scanProject(client AzureDevOpsApiClient, organization string, project string) {
	log.Info().Str("organization", organization).Str("project", project).Msg("Scanning project")
	listBuilds(client, organization, project)
}

func listAccounts(client AzureDevOpsApiClient, userId string) {
	accounts, _, err := client.ListAccounts(userId)
	if err != nil {
		log.Fatal().Err(err).Str("userId", userId).Msg("Failed fetching accounts")
	}

	if len(accounts) == 0 {
		log.Info().Msg("No accounts found, check your token access scope!")
		return
	}

	for _, account := range accounts {
		log.Debug().Str("name", account.AccountName).Msg("Scanning Account")
		listProjects(client, account.AccountName)
	}
}

func listProjects(client AzureDevOpsApiClient, organization string) {
	continuationToken := ""
	for {
		projects, _, ctoken, err := client.ListProjects(continuationToken, organization)

		if err != nil {
			log.Fatal().Err(err).Str("organization", organization).Msg("Failed fetching projects")
		}

		for _, project := range projects {
			listBuilds(client, organization, project.Name)
		}

		if ctoken == "" {
			break
		}
		continuationToken = ctoken
	}
}

func listBuilds(client AzureDevOpsApiClient, organization string, project string) {
	buildsCount := 0
	continuationToken := ""
	for {
		builds, _, ctoken, err := client.ListBuilds(continuationToken, organization, project)
		if err != nil {
			log.Error().Err(err).Str("organization", organization).Str("project", project).Msg("Failed fetching builds")
		}

		for _, build := range builds {
			log.Debug().Str("url", build.Links.Web.Href).Msg("Build")
			listLogs(client, organization, project, build.ID, build.Links.Web.Href)

			if options.Artifacts {
				listArtifacts(client, organization, project, build.ID, build.Links.Web.Href)
			}

			buildsCount = buildsCount + 1
			if buildsCount >= options.MaxBuilds && options.MaxBuilds > 0 {
				log.Trace().Str("organization", organization).Str("project", project).Msg("Reached MaxBuild runs, skip remaining")
				return
			}
		}

		if ctoken == "" {
			break
		}
		continuationToken = ctoken
	}
}

func listLogs(client AzureDevOpsApiClient, organization string, project string, buildId int, buildWebUrl string) {
	logs, _, err := client.ListBuildLogs(organization, project, buildId)
	if err != nil {
		log.Error().Err(err).Str("organization", organization).Str("project", project).Int("build", buildId).Msg("Failed fetching build logs")
	}

	for _, logEntry := range logs {
		log.Trace().Str("url", logEntry.URL).Msg("Download log")
		logLines, _, err := client.GetLog(organization, project, buildId, logEntry.ID)
		if err != nil {
			log.Error().Err(err).Str("organization", organization).Str("project", project).Int("build", buildId).Int("logId", logEntry.ID).Msg("Failed fetching build log lines")
		}

		scanLogLines(logLines, buildWebUrl)
	}
}

func scanLogLines(logs []byte, buildWebUrl string) {
	logResult, err := logline.ProcessLogs(logs, logline.ProcessOptions{
		MaxGoRoutines:     options.MaxScanGoRoutines,
		VerifyCredentials: options.TruffleHogVerification,
	})
	if err != nil {
		log.Debug().Err(err).Str("build", buildWebUrl).Msg("Failed detecting secrets of a single log line")
		return
	}

	result.ReportFindings(logResult.Findings, result.ReportOptions{
		LocationURL: buildWebUrl,
	})
}

func listArtifacts(client AzureDevOpsApiClient, organization string, project string, buildId int, buildWebUrl string) {
	continuationToken := ""
	for {
		artifacts, _, ctoken, err := client.ListBuildArtifacts(continuationToken, organization, project, buildId)
		if err != nil {
			log.Error().Err(err).Str("organization", organization).Str("project", project).Int("build", buildId).Msg("Failed fetching build artifacts")
		}

		for _, artifact := range artifacts {
			log.Trace().Str("name", artifact.Name).Msg("Analyze artifact")
			analyzeArtifact(client, artifact, buildWebUrl)
		}

		if ctoken == "" {
			break
		}
		continuationToken = ctoken
	}
}

func analyzeArtifact(client AzureDevOpsApiClient, art Artifact, buildWebUrl string) {
	zipBytes, _, err := client.DownloadArtifactZip(art.Resource.DownloadURL)
	if err != nil {
		log.Err(err).Msg("Failed downloading artifact")
		return
	}

	_, err = artifactproc.ProcessZipArtifact(zipBytes, artifactproc.ProcessOptions{
		MaxGoRoutines:     options.MaxScanGoRoutines,
		VerifyCredentials: options.TruffleHogVerification,
		BuildURL:          buildWebUrl,
		ArtifactName:      art.Name,
	})
	if err != nil {
		log.Err(err).Msg("Failed processing artifact")
		return
	}
}

func scanStatus() *zerolog.Event {
	return log.Info().Str("debug", "nothing to show ✔️")
}

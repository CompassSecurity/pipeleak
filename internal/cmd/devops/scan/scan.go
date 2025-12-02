package scan

import (
	"github.com/CompassSecurity/pipeleek/internal/cmd/flags"
	"github.com/CompassSecurity/pipeleek/pkg/config"
	pkgscan "github.com/CompassSecurity/pipeleek/pkg/devops/scan"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
)

type DevOpsScanOptions struct {
	config.CommonScanOptions
	Username     string
	AccessToken  string
	MaxBuilds    int
	Organization string
	Project      string
	DevOpsURL    string
}

var options = DevOpsScanOptions{
	CommonScanOptions: config.DefaultCommonScanOptions(),
}
var maxArtifactSize string

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
pipeleek ad scan --token xxxxxxxxxxx --username auser --artifacts

# Scan all pipelines of an organization
pipeleek ad scan --token xxxxxxxxxxx --username auser --artifacts --organization myOrganization

# Scan all pipelines of a project e.g. https://dev.azure.com/PowerShell/PowerShell
pipeleek ad scan --token xxxxxxxxxxx --username auser --artifacts --organization powershell --project PowerShell
		`,
		Run: Scan,
	}
	flags.AddCommonScanFlags(scanCmd, &options.CommonScanOptions, &maxArtifactSize)

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

	scanCmd.Flags().IntVarP(&options.MaxBuilds, "max-builds", "", -1, "Max. number of builds to scan per project")
	scanCmd.Flags().StringVarP(&options.Organization, "organization", "", "", "Organization name to scan")
	scanCmd.Flags().StringVarP(&options.Project, "project", "p", "", "Project name to scan - can be combined with organization")
	scanCmd.Flags().StringVarP(&options.DevOpsURL, "devops", "d", "https://dev.azure.com", "Azure DevOps base URL")

	return scanCmd
}

func Scan(cmd *cobra.Command, args []string) {
	if err := config.ValidateURL(options.DevOpsURL, "Azure DevOps URL"); err != nil {
		log.Fatal().Err(err).Msg("Invalid Azure DevOps URL")
	}
	if err := config.ValidateToken(options.AccessToken, "Azure DevOps Access Token"); err != nil {
		log.Fatal().Err(err).Msg("Invalid Azure DevOps Access Token")
	}
	if err := config.ValidateThreadCount(options.MaxScanGoRoutines); err != nil {
		log.Fatal().Err(err).Msg("Invalid thread count")
	}

	scanOpts, err := pkgscan.InitializeOptions(
		options.Username,
		options.AccessToken,
		options.DevOpsURL,
		options.Organization,
		options.Project,
		maxArtifactSize,
		options.Artifacts,
		options.TruffleHogVerification,
		options.MaxBuilds,
		options.MaxScanGoRoutines,
		options.ConfidenceFilter,
		options.HitTimeout,
	)
	if err != nil {
		log.Fatal().Err(err).Str("size", maxArtifactSize).Msg("Failed parsing max-artifact-size flag")
	}

	scanner := pkgscan.NewScanner(scanOpts)
	if err := scanner.Scan(); err != nil {
		log.Fatal().Err(err).Msg("Scan failed")
	}
}

package scan

import (
	giteascan "github.com/CompassSecurity/pipeleak/pkg/gitea/scan"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
)

type GiteaScanOptions struct {
	Token                  string
	GiteaURL               string
	Artifacts              bool
	ConfidenceFilter       []string
	MaxScanGoRoutines      int
	TruffleHogVerification bool
	Owned                  bool
	Organization           string
	Repository             string
	Cookie                 string
	RunsLimit              int
	StartRunID             int64
}

var scanOptions = GiteaScanOptions{}
var maxArtifactSize string

func NewScanCmd() *cobra.Command {
	scanCmd := &cobra.Command{
		Use:   "scan",
		Short: "Scan Gitea Actions",
		Long: `Scan Gitea Actions workflow runs and artifacts for secrets
### Token Authentication

You can create a personal access token in Gitea by navigating to your user settings, selecting "Applications", and then "Generate New Token". 

### Cookie Authentication

Due to differences between Gitea Actions API and UI access rights validation, a session cookie may be required in some cases.
The Actions API and UI are not yet fully in sync, causing some repositories to return 403 errors via API even when accessible through the UI.

To obtain the cookie:
1. Open your Gitea instance in a web browser
2. Open Developer Tools (F12)
3. Navigate to Application/Storage > Cookies
4. Find and copy the value of the 'i_like_gitea' cookie
5. Use it with the --cookie flag
`,
		Example: `
# Scan all accessible repositories (including public) and their artifacts
pipeleak gitea scan --token gitea_token_xxxxx --gitea https://gitea.example.com --artifacts --cookie your_cookie_value

# Scan without downloading artifacts
pipeleak gitea scan --token gitea_token_xxxxx --gitea https://gitea.example.com --cookie your_cookie_value

# Scan only repositories owned by the user
pipeleak gitea scan --token gitea_token_xxxxx --gitea https://gitea.example.com --owned --cookie your_cookie_value

# Scan all repositories of a specific organization
pipeleak gitea scan --token gitea_token_xxxxx --gitea https://gitea.example.com --organization my-org --cookie your_cookie_value

# Scan a specific repository
pipeleak gitea scan --token gitea_token_xxxxx --gitea https://gitea.example.com --repository owner/repo-name --cookie your_cookie_value

# Scan a specific repository but limit the number of workflow runs to scan
pipeleak gitea scan --token gitea_token_xxxxx --gitea https://gitea.example.com --repository owner/repo-name --runs-limit 20 --cookie your_cookie_value
		`,
		Run: Scan,
	}

	scanCmd.Flags().StringVarP(&scanOptions.Token, "token", "t", "", "Gitea personal access token")
	err := scanCmd.MarkFlagRequired("token")
	if err != nil {
		log.Fatal().Msg("Unable to require token flag")
	}

	scanCmd.Flags().StringVarP(&scanOptions.GiteaURL, "gitea", "g", "https://gitea.com", "Base Gitea URL (e.g. https://gitea.example.com)")

	scanCmd.Flags().BoolVarP(&scanOptions.Artifacts, "artifacts", "a", false, "Download and scan workflow artifacts")
	scanCmd.PersistentFlags().StringVarP(&maxArtifactSize, "max-artifact-size", "", "500Mb", "Max file size of an artifact to be included in scanning. Larger files are skipped. Format: https://pkg.go.dev/github.com/docker/go-units#FromHumanSize")
	scanCmd.Flags().BoolVarP(&scanOptions.Owned, "owned", "o", false, "Scan only repositories owned by the user")
	scanCmd.Flags().StringVarP(&scanOptions.Organization, "organization", "", "", "Scan all repositories of a specific organization")
	scanCmd.Flags().StringVarP(&scanOptions.Repository, "repository", "r", "", "Scan a specific repository (format: owner/repo)")
	scanCmd.Flags().StringVarP(&scanOptions.Cookie, "cookie", "c", "", "Gitea session cookie (i_like_gitea). Needed when scanning where you are NOT the owner of the repository")
	scanCmd.Flags().IntVarP(&scanOptions.RunsLimit, "runs-limit", "", 0, "Limit the number of workflow runs to scan per repository (0 = unlimited)")
	scanCmd.Flags().Int64VarP(&scanOptions.StartRunID, "start-run-id", "", 0, "Start scanning from a specific run ID (only valid with --repository flag, 0 = start from latest)")
	scanCmd.Flags().StringSliceVarP(&scanOptions.ConfidenceFilter, "confidence", "", []string{}, "Filter for confidence level, separate by comma if multiple. See documentation for more info.")
	scanCmd.PersistentFlags().IntVarP(&scanOptions.MaxScanGoRoutines, "threads", "", 4, "Nr of threads used to scan")
	scanCmd.PersistentFlags().BoolVarP(&scanOptions.TruffleHogVerification, "truffle-hog-verification", "", true, "Enable TruffleHog credential verification to actively test found credentials and only report verified ones (enabled by default, disable with --truffle-hog-verification=false)")

	return scanCmd
}

func Scan(cmd *cobra.Command, args []string) {
	if scanOptions.StartRunID > 0 && scanOptions.Repository == "" {
		log.Fatal().Msg("--start-run-id can only be used with --repository flag")
	}

	scanOpts, err := giteascan.InitializeOptions(
		scanOptions.Token,
		scanOptions.GiteaURL,
		scanOptions.Repository,
		scanOptions.Organization,
		scanOptions.Cookie,
		maxArtifactSize,
		scanOptions.Owned,
		scanOptions.Artifacts,
		scanOptions.TruffleHogVerification,
		scanOptions.RunsLimit,
		scanOptions.StartRunID,
		scanOptions.MaxScanGoRoutines,
		scanOptions.ConfidenceFilter,
	)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed initializing scan options")
	}

	// Validate cookie if provided
	if scanOptions.Cookie != "" {
		if err := giteascan.ValidateCookie(scanOpts); err != nil {
			log.Fatal().Err(err).Msg("Cookie validation failed")
		}
	}

	scanner := giteascan.NewScanner(scanOpts)
	if err := scanner.Scan(); err != nil {
		log.Fatal().Err(err).Msg("Scan failed")
	}
}

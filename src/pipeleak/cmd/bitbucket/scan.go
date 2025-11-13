package bitbucket

import (
	"github.com/CompassSecurity/pipeleak/pkg/bitbucket/scan"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
)

type BitBucketScanOptions struct {
	Email                  string
	AccessToken            string
	ConfidenceFilter       []string
	MaxScanGoRoutines      int
	TruffleHogVerification bool
	MaxPipelines           int
	Workspace              string
	Owned                  bool
	Public                 bool
	After                  string
	Artifacts              bool
	BitBucketURL           string
	BitBucketCookie        string
}

var options = BitBucketScanOptions{}
var maxArtifactSize string

func NewScanCmd() *cobra.Command {
	scanCmd := &cobra.Command{
		Use:   "scan",
		Short: "Scan BitBucket Pipelines",
		Long: `Create a BitBucket scoped API token [here](https://id.atlassian.com/manage-profile/security/api-tokens) and pass it to the <code>--token</code> flag.
The <code>--email</code> flag expects your account's email address.
To scan artifacts (uses internal APIs) you need to extract the session cookie value <code>cloud.session.token</code> from [bitbucket.org](https://bitbucket.org) using your browser and supply it in the <code>--cookie</code> flag.
A note on artifacts: Bitbucket artifacts are only stored for a limited time and only for paid accounts. Free accounts might not have artifacts available at all.
		  `,
		Example: `
# Scan a workspace (find public ones here: https://bitbucket.org/repo/all/) without artifacts
pipeleak bb scan --token ATATTxxxxxx --email auser@example.com --workspace bitbucketpipelines

# Scan your owned repositories and their artifacts
pipeleak bb scan -t ATATTxxxxxx -c eyJxxxxxxxxxxx --artifacts -e auser@example.com --owned

# Scan all public repositories without their artifacts
> If using --after, the API becomes quite unreliable 👀
pipeleak bb scan --token ATATTxxxxxx --email auser@example.com --public --maxPipelines 5 --after 2025-03-01T15:00:00+00:00
		`,
		Run: Scan,
	}
	scanCmd.Flags().StringVarP(&options.AccessToken, "token", "t", "", "Bitbucket API token - https://id.atlassian.com/manage-profile/security/api-tokens")
	scanCmd.Flags().StringVarP(&options.Email, "email", "e", "", "Bitbucket Email")
	scanCmd.Flags().StringVarP(&options.BitBucketCookie, "cookie", "c", "", "Bitbucket Cookie [value of cloud.session.token on https://bitbucket.org]")
	scanCmd.Flags().StringVarP(&options.BitBucketURL, "bitbucket", "b", "https://api.bitbucket.org/2.0", "BitBucket API base URL")
	scanCmd.PersistentFlags().BoolVarP(&options.Artifacts, "artifacts", "a", false, "Scan workflow artifacts")
	scanCmd.PersistentFlags().StringVarP(&maxArtifactSize, "max-artifact-size", "", "500Mb", "Max file size of an artifact to be included in scanning. Larger files are skipped. Format: https://pkg.go.dev/github.com/docker/go-units#FromHumanSize")
	scanCmd.MarkFlagsRequiredTogether("cookie", "artifacts")

	scanCmd.Flags().StringSliceVarP(&options.ConfidenceFilter, "confidence", "", []string{}, "Filter for confidence level, separate by comma if multiple. See readme for more info.")
	scanCmd.PersistentFlags().IntVarP(&options.MaxScanGoRoutines, "threads", "", 4, "Nr of threads used to scan")
	scanCmd.PersistentFlags().BoolVarP(&options.TruffleHogVerification, "truffle-hog-verification", "", true, "Enable the TruffleHog credential verification, will actively test the found credentials and only report those. Disable with --truffle-hog-verification=false")
	scanCmd.PersistentFlags().IntVarP(&options.MaxPipelines, "max-pipelines", "", -1, "Max. number of pipelines to scan per repository")

	scanCmd.Flags().StringVarP(&options.Workspace, "workspace", "w", "", "Workspace name to scan")
	scanCmd.PersistentFlags().BoolVarP(&options.Owned, "owned", "o", false, "Scan user onwed projects only")
	scanCmd.PersistentFlags().BoolVarP(&options.Public, "public", "p", false, "Scan all public repositories")
	scanCmd.PersistentFlags().StringVarP(&options.After, "after", "", "", "Filter public repos by a given date in ISO 8601 format: 2025-04-02T15:00:00+02:00 ")

	return scanCmd
}

func Scan(cmd *cobra.Command, args []string) {
	if options.AccessToken != "" && options.Email == "" {
		log.Fatal().Msg("When using --token you must also provide --email")
	}

	scanOpts, err := scan.InitializeOptions(
		options.Email,
		options.AccessToken,
		options.BitBucketCookie,
		options.BitBucketURL,
		options.Workspace,
		options.After,
		maxArtifactSize,
		options.Owned,
		options.Public,
		options.Artifacts,
		options.TruffleHogVerification,
		options.MaxPipelines,
		options.MaxScanGoRoutines,
		options.ConfidenceFilter,
	)
	if err != nil {
		log.Fatal().Err(err).Str("size", maxArtifactSize).Msg("Failed parsing max-artifact-size flag")
	}

	scanner := scan.NewScanner(scanOpts)
	if err := scanner.Scan(); err != nil {
		log.Fatal().Err(err).Msg("Scan failed")
	}
}

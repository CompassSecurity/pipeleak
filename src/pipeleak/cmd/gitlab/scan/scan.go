package scan

import (
	"net/url"
	"os"

	"github.com/CompassSecurity/pipeleak/cmd/gitlab/util"
	"github.com/CompassSecurity/pipeleak/pkg/logging"
	gounits "github.com/docker/go-units"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
)

var options = ScanOptions{}
var maxArtifactSize string

func NewScanCmd() *cobra.Command {
	scanCmd := &cobra.Command{
		Use:   "scan",
		Short: "Scan a GitLab instance",
		Long: `Scan a GitLab instance for secrets in pipeline jobs and optionally artifacts
### Dotenv
[Dotenv artifacts](https://docs.gitlab.com/ee/ci/yaml/artifacts_reports.html#artifactsreportsdotenv) are not accessible through the GitLab API. To scan these, you need to manually provide your session cookie after logging in via a web browser. The session cookie name is _gitlab_session. The cookie should be valid for [two weeks](https://gitlab.com/gitlab-org/gitlab/-/issues/395038).

### Memory Usage

As the scanner processes a lot of resources (especially when using  --artifacts) memory, CPU and disk usage can become hard to manage.
You can tweak --threads, --max-artifact-size and --job-limit to obtain a customized performance and achieve stable processing.
`,
		Example: `
# Scan all accessible projects pipelines and their artifacts and dotenv artifacts on gitlab.com
pipeleak gl scan --token glpat-xxxxxxxxxxx --gitlab https://gitlab.example.com -a -c [value-of-valid-_gitlab_session]

# Scan all projects matching the search query kubernetes
pipeleak gl scan --token glpat-xxxxxxxxxxx --gitlab https://gitlab.example.com --search kubernetes

# Scan all pipelines of projects you own
pipeleak gl scan --token glpat-xxxxxxxxxxx --gitlab https://gitlab.example.com --owned

# Scan all pipelines of projects you are a member of
pipeleak gl scan --token glpat-xxxxxxxxxxx --gitlab https://gitlab.example.com --member

# Scan all accessible projects pipelines but limit the number of jobs scanned per project to 10, only scan artifacts smaller than 200MB and use 8 threads
pipeleak gl scan --token glpat-xxxxxxxxxxx --gitlab https://gitlab.example.com --job-limit 10 -a --max-artifact-size 200Mb --threads 8

# Scan a single repository
pipeleak gl scan --token glpat-xxxxxxxxxxx --gitlab https://gitlab.example.com --repo mygroup/myproject

# Scan all repositories in a namespace
pipeleak gl scan --token glpat-xxxxxxxxxxx --gitlab https://gitlab.example.com --namespace mygroup
		`,
		Run: Scan,
	}

	scanCmd.Flags().StringVarP(&options.GitlabUrl, "gitlab", "g", "", "GitLab instance URL")
	err := scanCmd.MarkFlagRequired("gitlab")
	if err != nil {
		log.Fatal().Stack().Err(err).Msg("Unable to require gitlab flag")
	}

	scanCmd.Flags().StringVarP(&options.GitlabApiToken, "token", "t", "", "GitLab API Token")
	err = scanCmd.MarkFlagRequired("token")
	if err != nil {
		log.Fatal().Msg("Unable to require token flag")
	}
	scanCmd.MarkFlagsRequiredTogether("gitlab", "token")

	scanCmd.Flags().StringVarP(&options.GitlabCookie, "cookie", "c", "", "GitLab Cookie _gitlab_session (must be extracted from your browser, use remember me)")
	scanCmd.Flags().StringVarP(&options.ProjectSearchQuery, "search", "s", "", "Query string for searching projects")
	scanCmd.Flags().StringSliceVarP(&options.ConfidenceFilter, "confidence", "", []string{}, "Filter for confidence level, separate by comma if multiple. See readme for more info.")

	scanCmd.PersistentFlags().BoolVarP(&options.Artifacts, "artifacts", "a", false, "Scan job artifacts")
	scanCmd.PersistentFlags().BoolVarP(&options.Owned, "owned", "o", false, "Scan user onwed projects only")
	scanCmd.PersistentFlags().BoolVarP(&options.Member, "member", "m", false, "Scan projects the user is member of")
	scanCmd.PersistentFlags().StringVarP(&options.Repository, "repo", "r", "", "Single repository to scan, format: namespace/repo")
	scanCmd.PersistentFlags().StringVarP(&options.Namespace, "namespace", "n", "", "Namespace to scan (all repos in the namespace will be scanned)")
	scanCmd.PersistentFlags().IntVarP(&options.JobLimit, "job-limit", "j", 0, "Scan a max number of pipeline jobs - trade speed vs coverage. 0 scans all and is the default.")
	scanCmd.PersistentFlags().StringVarP(&maxArtifactSize, "max-artifact-size", "", "500Mb", "Max file size of an artifact to be included in scanning. Larger files are skipped. Format: https://pkg.go.dev/github.com/docker/go-units#FromHumanSize")
	scanCmd.PersistentFlags().IntVarP(&options.MaxScanGoRoutines, "threads", "", 4, "Nr of threads used to scan")
	scanCmd.PersistentFlags().StringVarP(&options.QueueFolder, "queue", "q", "", "Relative or absolute folderpath where the queue files will be stored. Defaults to system tmp. Non-existing folders will be created.")
	scanCmd.PersistentFlags().BoolVarP(&options.TruffleHogVerification, "truffleHogVerification", "", true, "Enable the TruffleHog credential verification, will actively test the found credentials and only report those. Disable with --truffleHogVerification=false")

	scanCmd.PersistentFlags().BoolVarP(&options.Verbose, "verbose", "v", false, "Verbose logging")

	return scanCmd
}

func Scan(cmd *cobra.Command, args []string) {
	logging.SetLogLevel(options.Verbose)
	go logging.ShortcutListeners(scanStatus)

	_, err := url.ParseRequestURI(options.GitlabUrl)
	if err != nil {
		log.Fatal().Msg("The provided GitLab URL is not a valid URL")
		os.Exit(1)
	}

	options.MaxArtifactSize = parseFileSize(maxArtifactSize)

	version := util.DetermineVersion(options.GitlabUrl, options.GitlabApiToken)
	log.Info().Str("version", version.Version).Str("revision", version.Revision).Msg("Gitlab Version Check")
	ScanGitLabPipelines(&options)
	log.Info().Msg("Scan Finished, Bye Bye 🏳️‍🌈🔥")
}

func parseFileSize(size string) int64 {
	byteSize, err := gounits.FromHumanSize(size)
	if err != nil {
		log.Fatal().Err(err).Str("size", size).Msg("Failed parsing flag")
	}

	return byteSize
}

func scanStatus() *zerolog.Event {
	queueLength := GetQueueStatus()
	return log.Info().Int("pendingjobs", queueLength)
}

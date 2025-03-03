package gitlab

import (
	"net/url"
	"os"

	"atomicgo.dev/keyboard"
	"atomicgo.dev/keyboard/keys"
	"github.com/CompassSecurity/pipeleak/helper"
	"github.com/CompassSecurity/pipeleak/scanner"
	gounits "github.com/docker/go-units"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
)

var options = scanner.ScanOptions{}
var maxArtifactSize string

func NewScanCmd() *cobra.Command {
	scanCmd := &cobra.Command{
		Use:   "scan [no options!]",
		Short: "Scan a GitLab instance",
		Run:   Scan,
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
	scanCmd.PersistentFlags().IntVarP(&options.JobLimit, "job-limit", "j", 0, "Scan a max number of pipeline jobs - trade speed vs coverage. 0 scans all and is the default.")
	scanCmd.PersistentFlags().StringVarP(&maxArtifactSize, "max-artifact-size", "", "500Mb", "Max file size of an artifact to be included in scanning. Larger files are skipped. Format: https://pkg.go.dev/github.com/docker/go-units#FromHumanSize")
	scanCmd.PersistentFlags().IntVarP(&options.MaxScanGoRoutines, "threads", "", 4, "Nr of threads used to scan")
	scanCmd.PersistentFlags().StringVarP(&options.QueueFolder, "queue", "q", "", "Relative folderpath where the queue files will be stored. Defaults to system tmp")
	scanCmd.PersistentFlags().BoolVarP(&options.TruffleHogVerification, "truffleHogVerification", "", true, "Enable the TruffleHog credential verification, will actively test the found credentials and only report those. Disable with --truffleHogVerification=false")

	scanCmd.PersistentFlags().BoolVarP(&options.Verbose, "verbose", "v", false, "Verbose logging")

	return scanCmd
}

func Scan(cmd *cobra.Command, args []string) {
	setLogLevel()
	go shortcutListeners()

	_, err := url.ParseRequestURI(options.GitlabUrl)
	if err != nil {
		log.Fatal().Msg("The provided GitLab URL is not a valid URL")
		os.Exit(1)
	}

	options.MaxArtifactSize = parseFileSize(maxArtifactSize)

	version := helper.DetermineVersion(options.GitlabUrl, options.GitlabApiToken)
	log.Info().Str("version", version.Version).Str("revision", version.Revision).Msg("Gitlab Version Check")
	scanner.ScanGitLabPipelines(&options)
	log.Info().Msg("Scan Finished, Bye Bye 🏳️‍🌈🔥")
}

func parseFileSize(size string) int64 {
	byteSize, err := gounits.FromHumanSize(size)
	if err != nil {
		log.Fatal().Err(err).Str("size", size).Msg("Failed parsing flag")
	}

	return byteSize
}

func setLogLevel() {
	if options.Verbose {
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
		log.Debug().Msg("Verbose log output enabled")
	}
}

func shortcutListeners() {
	err := keyboard.Listen(func(key keys.Key) (stop bool, err error) {
		switch key.Code {
		case keys.CtrlC, keys.Escape:
			return true, nil
		case keys.RuneKey:
			if key.String() == "t" {
				zerolog.SetGlobalLevel(zerolog.TraceLevel)
				log.Info().Msg("Loglevel Trace")
			}

			if key.String() == "d" {
				zerolog.SetGlobalLevel(zerolog.DebugLevel)
				log.Info().Msg("Loglevel Debug")
			}

			if key.String() == "i" {
				zerolog.SetGlobalLevel(zerolog.InfoLevel)
				log.Info().Msg("Loglevel Info")
			}

			if key.String() == "w" {
				zerolog.SetGlobalLevel(zerolog.WarnLevel)
				log.Warn().Msg("Loglevel Warn")
			}

			if key.String() == "e" {
				zerolog.SetGlobalLevel(zerolog.ErrorLevel)
				log.Error().Msg("Loglevel Error")
			}

			if key.String() == "s" {
				received, queueLength := scanner.GetQueueStatus()
				log.Info().Int("runningJobs", received).Int("pendingjobs", queueLength).Msg("Queue status")
			}
		}

		return false, nil
	})

	if err != nil {
		log.Error().Err(err).Msg("Failed hooking keyboard bindings")
	}
}

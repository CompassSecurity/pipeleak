package cmd

import (
	"net/url"
	"os"

	"github.com/CompassSecurity/pipeleak/scanner"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
)

var (
	gitlabUrl      string
	gitlabApiToken string
	gitlabCookie   string
	artifacts      bool
	owned          bool
	verbose        bool
)

func NewScanCmd() *cobra.Command {
	scanCmd := &cobra.Command{
		Use:   "scan [no options!]",
		Short: "Scan a GitLab instance",
		Run:   Scan,
	}
	//
	scanCmd.Flags().StringVarP(&gitlabUrl, "gitlab", "g", "", "GitLab instance URL")
	err := scanCmd.MarkFlagRequired("gitlab")
	if err != nil {
		log.Error().Msg("Unable to require gitlab flag: " + err.Error())
	}

	scanCmd.Flags().StringVarP(&gitlabApiToken, "token", "t", "", "GitLab API Token")
	err = scanCmd.MarkFlagRequired("token")
	if err != nil {
		log.Error().Msg("Unable to require token flag: " + err.Error())
	}
	scanCmd.MarkFlagsRequiredTogether("gitlab", "token")

	scanCmd.Flags().StringVarP(&gitlabCookie, "cookie", "c", "", "GitLab Cookie _gitlab_session (must be extracted from your browser, use remember me)")

	scanCmd.PersistentFlags().BoolVarP(&artifacts, "artifacts", "a", false, "Scan Job Artifacts")
	scanCmd.PersistentFlags().BoolVarP(&owned, "owned", "o", false, "Scan Onwed Projects Only")

	scanCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "Verbose Logging")

	return scanCmd
}

func Scan(cmd *cobra.Command, args []string) {
	setLogLevel()

	_, err := url.ParseRequestURI(gitlabUrl)
	if err != nil {
		log.Fatal().Msg("The provided GitLab URL is not a valid URL")
		os.Exit(1)
	}

	scanner.ScanGitLabPipelines(gitlabUrl, gitlabApiToken, gitlabCookie, artifacts, owned)
}

func setLogLevel() {
	if verbose {
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
		log.Debug().Msg("Verbose log output enabled")
	}
}

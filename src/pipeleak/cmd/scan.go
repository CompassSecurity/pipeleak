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
	gitlabUrl          string
	gitlabApiToken     string
	gitlabCookie       string
	projectSearchQuery string
	artifacts          bool
	owned              bool
	member             bool
	jobLimit           int
	verbose            bool
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
		log.Fatal().Stack().Err(err).Msg("Unable to require gitlab flag")
	}

	scanCmd.Flags().StringVarP(&gitlabApiToken, "token", "t", "", "GitLab API Token")
	err = scanCmd.MarkFlagRequired("token")
	if err != nil {
		log.Fatal().Msg("Unable to require token flag")
	}
	scanCmd.MarkFlagsRequiredTogether("gitlab", "token")

	scanCmd.Flags().StringVarP(&gitlabCookie, "cookie", "c", "", "GitLab Cookie _gitlab_session (must be extracted from your browser, use remember me)")
	scanCmd.Flags().StringVarP(&projectSearchQuery, "search", "s", "", "Query string for searching projects")

	scanCmd.PersistentFlags().BoolVarP(&artifacts, "artifacts", "a", true, "Scan job artifacts")
	scanCmd.PersistentFlags().BoolVarP(&owned, "owned", "o", false, "Scan user onwed projects only")
	scanCmd.PersistentFlags().BoolVarP(&member, "member", "m", false, "Scan projects the user is member of")
	scanCmd.PersistentFlags().IntVarP(&jobLimit, "job-limit", "j", 0, "Scan a max number of pipeline jobs - trade speed vs coverage. 0 scans all and is the default.")

	scanCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "Verbose logging")

	return scanCmd
}

func Scan(cmd *cobra.Command, args []string) {
	setLogLevel()

	_, err := url.ParseRequestURI(gitlabUrl)
	if err != nil {
		log.Fatal().Msg("The provided GitLab URL is not a valid URL")
		os.Exit(1)
	}

	scanner.ScanGitLabPipelines(gitlabUrl, gitlabApiToken, gitlabCookie, artifacts, owned, projectSearchQuery, jobLimit, member)
	log.Info().Msg("Scan Finished, Bye Bye 🏳️‍🌈🔥")
}

func setLogLevel() {
	if verbose {
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
		log.Debug().Msg("Verbose log output enabled")
	}
}

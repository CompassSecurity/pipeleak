package cmd

import (
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
)

var (
	gitlabApiToken string
	gitlabUrl      string
	verbose        bool
)

func NewRunnersRootCmd() *cobra.Command {
	runnersCmd := &cobra.Command{
		Use:   "runners [no options!]",
		Short: "runner related commands",
	}

	runnersCmd.PersistentFlags().StringVarP(&gitlabUrl, "gitlab", "g", "", "GitLab instance URL")
	err := runnersCmd.MarkPersistentFlagRequired("gitlab")
	if err != nil {
		log.Fatal().Stack().Err(err).Msg("Unable to require gitlab flag")
	}

	runnersCmd.PersistentFlags().StringVarP(&gitlabApiToken, "token", "t", "", "GitLab API Token")
	err = runnersCmd.MarkPersistentFlagRequired("token")
	if err != nil {
		log.Error().Stack().Err(err).Msg("Unable to require token flag")
	}
	runnersCmd.MarkFlagsRequiredTogether("gitlab", "token")

	runnersCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "Verbose logging")

	runnersCmd.AddCommand(NewRunnersListCmd())
	runnersCmd.AddCommand(NewRunnersExploitCmd())

	return runnersCmd
}

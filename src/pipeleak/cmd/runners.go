package cmd

import (
	"github.com/CompassSecurity/pipeleak/scanner"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
)

type runnersResult struct {
	Hostnames []string `json:"hostnames"`
	Port      int      `json:"port"`
}

func NewRunnersCmd() *cobra.Command {
	runnersCmd := &cobra.Command{
		Use:   "runners [no options!]",
		Short: "List available runners",
		Run:   Runners,
	}
	runnersCmd.Flags().StringVarP(&gitlabUrl, "gitlab", "g", "", "GitLab instance URL")
	err := runnersCmd.MarkFlagRequired("gitlab")
	if err != nil {
		log.Error().Msg("Unable to require gitlab flag: " + err.Error())
	}

	runnersCmd.Flags().StringVarP(&gitlabApiToken, "token", "t", "", "GitLab API Token")
	err = runnersCmd.MarkFlagRequired("token")
	if err != nil {
		log.Error().Msg("Unable to require token flag: " + err.Error())
	}
	runnersCmd.MarkFlagsRequiredTogether("gitlab", "token")

	runnersCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "Verbose logging")
	return runnersCmd
}

func Runners(cmd *cobra.Command, args []string) {
	setLogLevel()
	scanner.ListAllAvailableRunners(gitlabUrl, gitlabApiToken)
	log.Info().Msg("Done, Bye Bye 🏳️‍🌈🔥")
}

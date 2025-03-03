package runners

import (
	"github.com/CompassSecurity/pipeleak/gitlab"
	"github.com/CompassSecurity/pipeleak/helper"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
)

func NewRunnersListCmd() *cobra.Command {
	runnersCmd := &cobra.Command{
		Use:   "list [no options!]",
		Short: "List available runners",
		Run:   ListRunners,
	}

	return runnersCmd
}

func ListRunners(cmd *cobra.Command, args []string) {
	helper.SetLogLevel(verbose)
	gitlab.ListAllAvailableRunners(gitlabUrl, gitlabApiToken)
	log.Info().Msg("Done, Bye Bye 🏳️‍🌈🔥")
}

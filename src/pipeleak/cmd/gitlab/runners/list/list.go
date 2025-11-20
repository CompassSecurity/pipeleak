package list

import (
	pkgrunners "github.com/CompassSecurity/pipeleak/pkg/gitlab/runners/list"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
)

func NewRunnersListCmd(gitlabUrl, gitlabApiToken string) *cobra.Command {
	runnersCmd := &cobra.Command{
		Use:     "list",
		Short:   "List available runners",
		Long:    "List all available runners for projects and groups your token has access to.",
		Example: `pipeleak gl runners list --token glpat-xxxxxxxxxxx --gitlab https://gitlab.mydomain.com`,
		Run: func(cmd *cobra.Command, args []string) {
			pkgrunners.ListAllAvailableRunners(gitlabUrl, gitlabApiToken)
			log.Info().Msg("Done, Bye Bye 🏳️‍🌈🔥")
		},
	}

	return runnersCmd
}

package list

import (
	"github.com/CompassSecurity/pipeleak/pkg/gitea/runners/list"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
)

func NewListCmd(giteaUrl *string, giteaApiToken *string) *cobra.Command {
	listCmd := &cobra.Command{
		Use:     "list",
		Short:   "List available runners",
		Long:    "List all available Gitea Actions runners for repositories and organizations your token has access to.",
		Example: `pipeleak gitea runners list --token xxxxx --gitea https://gitea.mydomain.com`,
		Run: func(cmd *cobra.Command, args []string) {
			list.ListAllAvailableRunners(*giteaUrl, *giteaApiToken)
			log.Info().Msg("Done, Bye Bye 🏳️‍🌈🔥")
		},
	}

	return listCmd
}

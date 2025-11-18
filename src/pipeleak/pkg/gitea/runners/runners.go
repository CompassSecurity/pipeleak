package runners

import (
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
)

var (
	giteaApiToken string
	giteaUrl      string
)

func NewRunnersRootCmd() *cobra.Command {
	runnersCmd := &cobra.Command{
		Use:   "runners",
		Short: "runner related commands",
		Long:  "Commands to enumerate and exploit Gitea Actions runners.",
	}

	runnersCmd.PersistentFlags().StringVarP(&giteaUrl, "gitea", "g", "", "Gitea instance URL")
	err := runnersCmd.MarkPersistentFlagRequired("gitea")
	if err != nil {
		log.Fatal().Stack().Err(err).Msg("Unable to require gitea flag")
	}

	runnersCmd.PersistentFlags().StringVarP(&giteaApiToken, "token", "t", "", "Gitea API Token")
	err = runnersCmd.MarkPersistentFlagRequired("token")
	if err != nil {
		log.Error().Stack().Err(err).Msg("Unable to require token flag")
	}
	runnersCmd.MarkFlagsRequiredTogether("gitea", "token")

	runnersCmd.AddCommand(NewRunnersListCmd())
	runnersCmd.AddCommand(NewRunnersExploitCmd())

	return runnersCmd
}

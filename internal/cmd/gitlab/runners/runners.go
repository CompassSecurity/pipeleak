package runners

import (
	"github.com/CompassSecurity/pipeleek/internal/cmd/gitlab/runners/exploit"
	"github.com/CompassSecurity/pipeleek/internal/cmd/gitlab/runners/list"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
)

var (
	gitlabApiToken string
	gitlabUrl      string
)

func NewRunnersRootCmd() *cobra.Command {
	runnersCmd := &cobra.Command{
		Use:   "runners",
		Short: "runner related commands",
		Long:  "Commands to enumerate and exploit GitLab runners.",
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

	runnersCmd.AddCommand(list.NewRunnersListCmd())
	runnersCmd.AddCommand(exploit.NewRunnersExploitCmd())

	return runnersCmd
}

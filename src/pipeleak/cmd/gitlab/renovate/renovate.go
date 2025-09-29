package renovate

import (
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
)

var (
	gitlabApiToken string
	gitlabUrl      string
	verbose        bool
)

func NewRenovateRootCmd() *cobra.Command {
	renovateCmd := &cobra.Command{
		Use:   "renovate",
		Short: "Renovate related commands",
		Long:  "Commands to enumerate and exploit GitLab Renovate bot configurations.",
	}

	renovateCmd.PersistentFlags().StringVarP(&gitlabUrl, "gitlab", "g", "", "GitLab instance URL")
	err := renovateCmd.MarkPersistentFlagRequired("gitlab")
	if err != nil {
		log.Fatal().Stack().Err(err).Msg("Unable to require gitlab flag")
	}

	renovateCmd.PersistentFlags().StringVarP(&gitlabApiToken, "token", "t", "", "GitLab API Token")
	err = renovateCmd.MarkPersistentFlagRequired("token")
	if err != nil {
		log.Fatal().Stack().Err(err).Msg("Unable to require token flag")
	}
	renovateCmd.MarkFlagsRequiredTogether("gitlab", "token")

	renovateCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "Verbose logging")

	renovateCmd.AddCommand(NewEnumCmd())
	renovateCmd.AddCommand(NewAutodiscoveryCmd())
	renovateCmd.AddCommand(NewPrivescCmd())

	return renovateCmd
}

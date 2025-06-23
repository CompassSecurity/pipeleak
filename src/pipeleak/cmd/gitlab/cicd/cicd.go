package cicd

import (
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
)

var (
	gitlabApiToken string
	gitlabUrl      string
	verbose        bool
)

func NewCiCdCmd() *cobra.Command {
	ciCdCmd := &cobra.Command{
		Use:   "cicd -r mygroup/myrepo",
		Short: "CI/CD related commands",
	}

	ciCdCmd.PersistentFlags().StringVarP(&gitlabUrl, "gitlab", "g", "", "GitLab instance URL")
	err := ciCdCmd.MarkPersistentFlagRequired("gitlab")
	if err != nil {
		log.Fatal().Stack().Err(err).Msg("Unable to require gitlab flag")
	}

	ciCdCmd.PersistentFlags().StringVarP(&gitlabApiToken, "token", "t", "", "GitLab API Token")
	err = ciCdCmd.MarkPersistentFlagRequired("token")
	if err != nil {
		log.Error().Stack().Err(err).Msg("Unable to require token flag")
	}
	ciCdCmd.MarkFlagsRequiredTogether("gitlab", "token")

	ciCdCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "Verbose logging")

	ciCdCmd.AddCommand(NewYamlCmd())

	return ciCdCmd
}

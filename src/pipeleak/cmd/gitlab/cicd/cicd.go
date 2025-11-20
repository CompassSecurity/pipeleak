package cicd

import (
	"github.com/CompassSecurity/pipeleak/cmd/gitlab/cicd/yaml"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
)

var (
	gitlabApiToken string
	gitlabUrl      string
)

func NewCiCdCmd() *cobra.Command {
	ciCdCmd := &cobra.Command{
		Use:   "cicd",
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

	ciCdCmd.AddCommand(yaml.NewYamlCmd(gitlabUrl, gitlabApiToken))

	return ciCdCmd
}

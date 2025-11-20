package variables

import (
	pkgvariables "github.com/CompassSecurity/pipeleak/pkg/gitlab/variables"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
)

var (
	gitlabApiToken string
	gitlabUrl      string
)

func NewVariablesCmd() *cobra.Command {
	variablesCmd := &cobra.Command{
		Use:     "variables",
		Short:   "Print configured CI/CD variables",
		Long:    "Fetch and print all configured CI/CD variables for projects, groups and instance (if admin) your token has access to.",
		Example: `pipeleak gl variables --token glpat-xxxxxxxxxxx --gitlab https://gitlab.mydomain.com`,
		Run:     FetchVariables,
	}
	variablesCmd.Flags().StringVarP(&gitlabUrl, "gitlab", "g", "", "GitLab instance URL")
	err := variablesCmd.MarkFlagRequired("gitlab")
	if err != nil {
		log.Fatal().Stack().Err(err).Msg("Unable to require gitlab flag")
	}

	variablesCmd.Flags().StringVarP(&gitlabApiToken, "token", "t", "", "GitLab API Token")
	err = variablesCmd.MarkFlagRequired("token")
	if err != nil {
		log.Fatal().Msg("Unable to require token flag")
	}
	variablesCmd.MarkFlagsRequiredTogether("gitlab", "token")

	return variablesCmd
}

func FetchVariables(cmd *cobra.Command, args []string) {
	pkgvariables.RunFetchVariables(gitlabUrl, gitlabApiToken)
}

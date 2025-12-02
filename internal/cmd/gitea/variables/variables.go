package variables

import (
	"github.com/CompassSecurity/pipeleek/pkg/gitea/variables"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
)

func NewVariablesCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "variables",
		Short: "List all Gitea Actions variables from groups and repositories",
		Long:  `Fetches and logs all Actions variables from organizations and their repositories in Gitea.`,
		Run: func(cmd *cobra.Command, args []string) {
			token, _ := cmd.Flags().GetString("token")
			url, _ := cmd.Flags().GetString("gitea")

			config := variables.Config{
				URL:   url,
				Token: token,
			}

			if err := variables.ListAllVariables(config); err != nil {
				log.Fatal().Err(err).Msg("Failed to list variables")
			}
		},
	}

	return cmd
}

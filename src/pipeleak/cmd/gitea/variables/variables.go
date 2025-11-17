package variables

import (
	"github.com/CompassSecurity/pipeleak/pkg/gitea/variables"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
)

func NewVariablesCommand() *cobra.Command {
	var (
		url      string
		token    string
		insecure bool
	)

	cmd := &cobra.Command{
		Use:   "variables",
		Short: "List all Gitea Actions variables from groups and repositories",
		Long:  `Fetches and logs all Actions variables from organizations and their repositories in Gitea.`,
		Run: func(cmd *cobra.Command, args []string) {
			config := variables.Config{
				URL:      url,
				Token:    token,
				Insecure: insecure,
			}

			if err := variables.ListAllVariables(config); err != nil {
				log.Fatal().Err(err).Msg("Failed to list variables")
			}
		},
	}

	cmd.Flags().StringVarP(&url, "url", "u", "https://gitea.com", "Gitea server URL (required)")
	cmd.Flags().StringVarP(&token, "token", "t", "", "Gitea access token (required)")

	cmd.MarkFlagRequired("token")

	return cmd
}

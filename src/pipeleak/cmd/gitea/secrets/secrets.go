package secrets

import (
	"github.com/CompassSecurity/pipeleak/pkg/gitea/secrets"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
)

func NewSecretsCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "secrets",
		Short: "List all Gitea Actions secrets from groups and repositories",
		Long:  `Fetches and logs all Actions secrets from organizations and their repositories in Gitea.`,
		Run: func(cmd *cobra.Command, args []string) {
			// Get token and gitea from parent persistent flags
			token, _ := cmd.Flags().GetString("token")
			url, _ := cmd.Flags().GetString("gitea")

			config := secrets.Config{
				URL:   url,
				Token: token,
			}

			if err := secrets.ListAllSecrets(config); err != nil {
				log.Fatal().Err(err).Msg("Failed to list secrets")
			}
		},
	}
	// --token and --gitea flags are inherited from parent command

	return cmd
}

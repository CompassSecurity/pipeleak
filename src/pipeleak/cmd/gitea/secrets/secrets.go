package secrets

import (
	"github.com/CompassSecurity/pipeleak/pkg/gitea/secrets"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
)

func NewSecretsCommand() *cobra.Command {
	var (
		url   string
		token string
	)

	cmd := &cobra.Command{
		Use:   "secrets",
		Short: "List all Gitea Actions secrets from groups and repositories",
		Long:  `Fetches and logs all Actions secrets from organizations and their repositories in Gitea.`,
		Run: func(cmd *cobra.Command, args []string) {
			config := secrets.Config{
				URL:   url,
				Token: token,
			}

			if err := secrets.ListAllSecrets(config); err != nil {
				log.Fatal().Err(err).Msg("Failed to list secrets")
			}
		},
	}

	cmd.Flags().StringVarP(&url, "url", "u", "https://gitea.com", "Gitea server URL")
	cmd.Flags().StringVarP(&token, "token", "t", "", "Gitea access token (required)")

	_ = cmd.MarkFlagRequired("token")

	return cmd
}

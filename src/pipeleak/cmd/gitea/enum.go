package gitea

import (
	"code.gitea.io/sdk/gitea"
	"github.com/CompassSecurity/pipeleak/helper"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
)

func NewEnumCmd() *cobra.Command {
	enumCmd := &cobra.Command{
		Use:     "enum",
		Short:   "Enumerate access rights of a Gitea access token",
		Long:    "Enumerate access rights of a Gitea access token by retrieving the authenticated user's information.",
		Example: `pipeleak gitea enum --token $GITEA_TOKEN --gitea https://gitea.mycompany.com`,
		Run:     Enum,
	}
	enumCmd.Flags().StringVarP(&giteaUrl, "gitea", "g", "https://gitea.com", "Gitea instance URL")
	enumCmd.Flags().StringVarP(&giteaApiToken, "token", "t", "", "Gitea API Token")

	return enumCmd
}

func Enum(cmd *cobra.Command, args []string) {
	helper.SetLogLevel(verbose)

	// Initialize Gitea client
	client, err := gitea.NewClient(giteaUrl, gitea.SetToken(giteaApiToken))
	if err != nil {
		log.Fatal().Stack().Err(err).Msg("Failed creating gitea client")
		return
	}

	// Fetch user info
	log.Info().Msg("Enumerating User")
	user, _, err := client.GetMyUserInfo()
	if err != nil {
		log.Fatal().Stack().Err(err).Msg("Failed fetching current user")
		return
	}

	// Log user data structure for debug visibility
	log.Debug().Interface("user", user).Msg("Full user data structure")

	// Also log with structured logging
	log.Warn().
		Int64("id", user.ID).
		Str("username", user.UserName).
		Str("fullName", user.FullName).
		Str("email", user.Email).
		Str("description", user.Description).
		Bool("isAdmin", user.IsAdmin).
		Bool("isActive", user.IsActive).
		Bool("restricted", user.Restricted).
		Msg("Current user")

	log.Info().Msg("Done")
}

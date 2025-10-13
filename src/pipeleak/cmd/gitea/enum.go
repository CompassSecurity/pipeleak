package gitea

import (
	"fmt"
	"os"

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

	enumCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "Verbose logging")
	return enumCmd
}

func Enum(cmd *cobra.Command, args []string) {
	helper.SetLogLevel(verbose)

	// Check if token is provided via flag or environment variable
	if giteaApiToken == "" {
		giteaApiToken = os.Getenv("GITEA_TOKEN")
	}

	if giteaApiToken == "" {
		log.Fatal().Msg("error: missing --token flag or GITEA_TOKEN environment variable")
		return
	}

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

	// Output all user data fields in plain text
	fmt.Printf("\nAuthenticated User Information:\n")
	fmt.Printf("================================\n")
	fmt.Printf("ID:               %d\n", user.ID)
	fmt.Printf("Username:         %s\n", user.UserName)
	fmt.Printf("Login Name:       %s\n", user.LoginName)
	fmt.Printf("Source ID:        %d\n", user.SourceID)
	fmt.Printf("Full Name:        %s\n", user.FullName)
	fmt.Printf("Email:            %s\n", user.Email)
	fmt.Printf("Avatar URL:       %s\n", user.AvatarURL)
	fmt.Printf("Language:         %s\n", user.Language)
	fmt.Printf("Is Admin:         %t\n", user.IsAdmin)
	fmt.Printf("Last Login:       %s\n", user.LastLogin)
	fmt.Printf("Created:          %s\n", user.Created)
	fmt.Printf("Restricted:       %t\n", user.Restricted)
	fmt.Printf("Is Active:        %t\n", user.IsActive)
	fmt.Printf("Prohibit Login:   %t\n", user.ProhibitLogin)
	fmt.Printf("Location:         %s\n", user.Location)
	fmt.Printf("Website:          %s\n", user.Website)
	fmt.Printf("Description:      %s\n", user.Description)
	fmt.Printf("Visibility:       %s\n", user.Visibility)
	fmt.Printf("Followers:        %d\n", user.FollowerCount)
	fmt.Printf("Following:        %d\n", user.FollowingCount)
	fmt.Printf("Starred Repos:    %d\n", user.StarredRepoCount)

	// Also log with structured logging
	log.Warn().
		Int64("id", user.ID).
		Str("username", user.UserName).
		Str("fullName", user.FullName).
		Str("email", user.Email).
		Bool("isAdmin", user.IsAdmin).
		Bool("isActive", user.IsActive).
		Bool("restricted", user.Restricted).
		Msg("Current user")

	log.Info().Msg("Done")
}

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
		Long:    "Enumerate access rights of a Gitea access token by retrieving the authenticated user's information, organizations with access levels, and all accessible repositories with permissions.",
		Example: `pipeleak gitea enum --token $GITEA_TOKEN --gitea https://gitea.mycompany.com`,
		Run:     Enum,
	}
	enumCmd.Flags().StringVarP(&giteaUrl, "gitea", "g", "https://gitea.com", "Gitea instance URL")
	enumCmd.Flags().StringVarP(&giteaApiToken, "token", "t", "", "Gitea API Token")

	return enumCmd
}

func Enum(cmd *cobra.Command, args []string) {
	helper.SetLogLevel(verbose)

	client, err := gitea.NewClient(giteaUrl, gitea.SetToken(giteaApiToken))
	if err != nil {
		log.Fatal().Stack().Err(err).Msg("Failed creating gitea client")
		return
	}

	log.Info().Msg("Enumerating User")
	user, _, err := client.GetMyUserInfo()
	if err != nil {
		log.Fatal().Stack().Err(err).Msg("Failed fetching current user")
		return
	}

	log.Debug().Interface("user", user).Msg("Full user data structure")

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

	log.Info().Msg("Enumerating Organizations")
	orgs, _, err := client.ListMyOrgs(gitea.ListOrgsOptions{})
	if err != nil {
		log.Error().Stack().Err(err).Msg("Failed fetching organizations")
	} else {
		for _, org := range orgs {
			orgPerms, _, err := client.GetOrgPermissions(org.UserName, user.UserName)
			if err != nil {
				log.Debug().Str("org", org.UserName).Err(err).Msg("Failed to get org permissions")
			}

			logEvent := log.Warn().
				Int64("id", org.ID).
				Str("name", org.UserName).
				Str("fullName", org.FullName).
				Str("description", org.Description).
				Str("visibility", org.Visibility)

			if orgPerms != nil {
				logEvent = logEvent.
					Bool("isOwner", orgPerms.IsOwner).
					Bool("isAdmin", orgPerms.IsAdmin).
					Bool("canWrite", orgPerms.CanWrite).
					Bool("canRead", orgPerms.CanRead).
					Bool("canCreateRepo", orgPerms.CanCreateRepository)
			}

			logEvent.Msg("Organization")

			orgRepos, _, err := client.ListOrgRepos(org.UserName, gitea.ListOrgReposOptions{})
			if err != nil {
				log.Debug().Str("org", org.UserName).Err(err).Msg("Failed to list org repositories")
				continue
			}

			for _, repo := range orgRepos {
				logRepo := log.Warn().
					Int64("id", repo.ID).
					Str("name", repo.Name).
					Str("fullName", repo.FullName).
					Str("owner", repo.Owner.UserName).
					Str("description", repo.Description).
					Bool("private", repo.Private).
					Bool("archived", repo.Archived).
					Str("url", repo.HTMLURL)

				if repo.Permissions != nil {
					logRepo = logRepo.
						Bool("admin", repo.Permissions.Admin).
						Bool("push", repo.Permissions.Push).
						Bool("pull", repo.Permissions.Pull)
				}

				logRepo.Msg("Organization Repository")
			}
		}
	}

	log.Info().Msg("Enumerating User Repositories")
	repos, _, err := client.ListMyRepos(gitea.ListReposOptions{})
	if err != nil {
		log.Error().Stack().Err(err).Msg("Failed fetching user repositories")
	} else {
		for _, repo := range repos {
			logRepo := log.Warn().
				Int64("id", repo.ID).
				Str("name", repo.Name).
				Str("fullName", repo.FullName).
				Str("owner", repo.Owner.UserName).
				Str("description", repo.Description).
				Bool("private", repo.Private).
				Bool("archived", repo.Archived).
				Str("url", repo.HTMLURL)

			if repo.Permissions != nil {
				logRepo = logRepo.
					Bool("admin", repo.Permissions.Admin).
					Bool("push", repo.Permissions.Push).
					Bool("pull", repo.Permissions.Pull)
			}

			logRepo.Msg("User Repository")
		}
	}
}

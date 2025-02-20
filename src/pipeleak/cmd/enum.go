package cmd

import (
	"github.com/CompassSecurity/pipeleak/helper"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	gitlab "gitlab.com/gitlab-org/api/client-go"
)

var minAccessLevel int

func NewEnumCmd() *cobra.Command {
	enumCmd := &cobra.Command{
		Use:   "enum [no options!]",
		Short: "Enumerate access rights of a Gitlab access token",
		Run:   Enum,
	}
	enumCmd.Flags().StringVarP(&gitlabUrl, "gitlab", "g", "", "GitLab instance URL")
	err := enumCmd.MarkFlagRequired("gitlab")
	if err != nil {
		log.Fatal().Stack().Err(err).Msg("Unable to require gitlab flag")
	}

	enumCmd.Flags().StringVarP(&gitlabApiToken, "token", "t", "", "GitLab API Token")
	err = enumCmd.MarkFlagRequired("token")
	if err != nil {
		log.Fatal().Msg("Unable to require token flag")
	}
	enumCmd.MarkFlagsRequiredTogether("gitlab", "token")

	enumCmd.PersistentFlags().IntVarP(&minAccessLevel, "level", "l", int(gitlab.GuestPermissions), "Minimum repo access level. See https://docs.gitlab.com/api/access_requests/#valid-access-levels for integer values")

	enumCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "Verbose logging")
	return enumCmd
}

func Enum(cmd *cobra.Command, args []string) {
	setLogLevel()
	git, err := helper.GetGitlabClient(gitlabApiToken, gitlabUrl)
	if err != nil {
		log.Fatal().Stack().Err(err).Msg("failed creating gitlab client")
	}

	user, _, err := git.Users.CurrentUser()

	if err != nil {
		log.Fatal().Stack().Err(err).Msg("failed fetching current usert")
	}

	log.Info().Msg("Enumerating User")
	log.Warn().Str("username", user.Username).Str("name", user.Name).Str("email", user.Email).Bool("admin", user.IsAdmin).Bool("bot", user.Bot).Msg("Current user")

	log.Info().Int("minAccessLevel", minAccessLevel).Msg("Enumerating repositories with minimum access level")
	projectOpts := &gitlab.ListProjectsOptions{
		ListOptions: gitlab.ListOptions{
			PerPage: 100,
			Page:    1,
		},
		MinAccessLevel: gitlab.Ptr(gitlab.AccessLevelValue(minAccessLevel)),
		OrderBy:        gitlab.Ptr("last_activity_at"),
	}

	for {
		projects, resp, err := git.Projects.ListProjects(projectOpts)
		if err != nil {
			log.Error().Stack().Err(err).Msg("Failed fetching projects")
		}

		for _, project := range projects {
			logLine := log.Warn().Str("project", project.WebURL).Str("name", project.NameWithNamespace).Bool("publicJobs", project.PublicJobs)

			if len(project.Description) > 0 {
				logLine.Str("description", project.Description)
			}

			if project.Permissions.ProjectAccess != nil {
				logLine.Int("projectAccessLevel", int(project.Permissions.ProjectAccess.AccessLevel))
			}

			if project.Permissions.GroupAccess != nil {
				logLine.Int("groupAcessLevel", int(project.Permissions.GroupAccess.AccessLevel))
			}

			logLine.Msg("Repo")
		}

		if resp.NextPage == 0 {
			break
		}

		projectOpts.Page = resp.NextPage
	}

	log.Info().Int("minAccessLevel", minAccessLevel).Msg("Enumerating groups with minimum access level")
	listGroupsOpts := &gitlab.ListGroupsOptions{
		ListOptions: gitlab.ListOptions{
			PerPage: 100,
			Page:    1,
		},
		MinAccessLevel: gitlab.Ptr(gitlab.AccessLevelValue(minAccessLevel)),
		TopLevelOnly:   gitlab.Ptr(false),
	}

	for {
		groups, resp, err := git.Groups.ListGroups(listGroupsOpts)
		if err != nil {
			log.Error().Stack().Err(err).Msg("failed listing groups")
		}

		for _, group := range groups {
			logLine := log.Warn().Str("group", group.WebURL).Str("fullName", group.FullName).Str("name", group.Name).Str("visibility", string(group.Visibility))

			if len(group.Description) > 0 {
				logLine.Str("description", group.Description)
			}

			logLine.Msg("Group")
		}

		if resp.NextPage == 0 {
			break
		}
		listGroupsOpts.Page = resp.NextPage
	}
}

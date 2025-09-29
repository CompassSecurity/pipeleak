package gitlab

import (
	"github.com/CompassSecurity/pipeleak/cmd/gitlab/util"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	gitlab "gitlab.com/gitlab-org/api/client-go"
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

	variablesCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "Verbose logging")
	return variablesCmd
}

func FetchVariables(cmd *cobra.Command, args []string) {
	if verbose {
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
		log.Debug().Msg("Verbose log output enabled")
	}

	log.Info().Msg("Fetching project variables")

	git, err := util.GetGitlabClient(gitlabApiToken, gitlabUrl)
	if err != nil {
		log.Fatal().Stack().Err(err).Msg("Failed creating gitlab client")
	}

	projectOpts := &gitlab.ListProjectsOptions{
		ListOptions: gitlab.ListOptions{
			PerPage: 100,
			Page:    1,
		},
		Membership:     gitlab.Ptr(true),
		MinAccessLevel: gitlab.Ptr(gitlab.MaintainerPermissions),
		OrderBy:        gitlab.Ptr("last_activity_at"),
	}

	for {
		projects, resp, err := git.Projects.ListProjects(projectOpts)
		if err != nil {
			log.Error().Stack().Err(err).Msg("Failed fetching projects")
			break
		}

		for _, project := range projects {
			log.Debug().Str("project", project.WebURL).Msg("Fetch project variables")
			pvs, _, err := git.ProjectVariables.ListVariables(project.ID, nil, nil)
			if err != nil {
				log.Error().Stack().Err(err).Msg("Failed fetching project variables")
				continue
			}
			if len(pvs) > 0 {
				log.Warn().Str("project", project.WebURL).Any("variables", pvs).Msg("Project variables")
			}
		}

		if resp.NextPage == 0 {
			break
		}
		projectOpts.Page = resp.NextPage
	}

	log.Info().Msg("Fetching group variables")
	log.Warn().Msg("Group inherited variables cannot really be enumerated through the API. If you have inherited access to variables from groups, you do not have owner access to, check manually in the GitLab UI!")

	listGroupsOpts := &gitlab.ListGroupsOptions{
		ListOptions: gitlab.ListOptions{
			PerPage: 100,
			Page:    1,
		},
		AllAvailable: gitlab.Ptr(true),
		// one can have group guest access and thus inherit variables.
		// However these are visible in the GUI but not on API level, only if the user has owner access to the group
		MinAccessLevel: gitlab.Ptr(gitlab.MaintainerPermissions),
		TopLevelOnly:   gitlab.Ptr(false),
	}

	for {
		groups, resp, err := git.Groups.ListGroups(listGroupsOpts)
		if err != nil {
			log.Error().Stack().Err(err).Msg("failed listing groups")
		}

		for _, group := range groups {
			log.Debug().Str("Group", group.WebURL).Msg("Fetch group variables")
			gvs, _, err := git.GroupVariables.ListVariables(group.ID, nil, nil)
			if err != nil {
				log.Debug().Stack().Err(err).Msg("Failed fetching group variables")
			}
			if len(gvs) > 0 {
				log.Warn().Str("Group", group.WebURL).Any("variables", gvs).Msg("Group variables")
			}
		}

		if resp.NextPage == 0 {
			break
		}
		listGroupsOpts.Page = resp.NextPage
	}

	log.Info().Msg("Fetching instance variables, only allowed for admins")
	ivs, _, err := git.InstanceVariables.ListVariables(nil)
	if err != nil {
		log.Debug().Stack().Err(err).Msg("Failed fetching instance variables")
	} else {
		log.Warn().Any("variables", ivs).Msg("Instance variables")
	}

	log.Info().Msg("Fetched all variables")
}

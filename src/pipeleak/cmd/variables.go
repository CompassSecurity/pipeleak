package cmd

import (
	"github.com/CompassSecurity/pipeleak/helper"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"github.com/xanzy/go-gitlab"
)

func NewVariablesCmd() *cobra.Command {
	vulnCmd := &cobra.Command{
		Use:   "variables [no options!]",
		Short: "Print configured CI/CD variables",
		Run:   FetchVariables,
	}
	vulnCmd.Flags().StringVarP(&gitlabUrl, "gitlab", "g", "", "GitLab instance URL")
	err := vulnCmd.MarkFlagRequired("gitlab")
	if err != nil {
		log.Fatal().Stack().Err(err).Msg("Unable to require gitlab flag")
	}

	vulnCmd.Flags().StringVarP(&gitlabApiToken, "token", "t", "", "GitLab API Token")
	err = vulnCmd.MarkFlagRequired("token")
	if err != nil {
		log.Fatal().Msg("Unable to require token flag")
	}
	vulnCmd.MarkFlagsRequiredTogether("gitlab", "token")

	vulnCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "Verbose logging")
	return vulnCmd
}

func FetchVariables(cmd *cobra.Command, args []string) {
	if verbose {
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
		log.Debug().Msg("Verbose log output enabled")
	}

	log.Info().Msg("Fetching project variables")

	git, err := helper.GetGitlabClient(gitlabApiToken, gitlabUrl)
	if err != nil {
		log.Fatal().Stack().Err(err).Msg("failed creating gitlab client")
	}

	projectOpts := &gitlab.ListProjectsOptions{
		ListOptions: gitlab.ListOptions{
			PerPage: 100,
			Page:    1,
		},
		Membership:     gitlab.Ptr(true),
		MinAccessLevel: gitlab.Ptr(gitlab.OwnerPermissions),
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

	listGroupsOpts := &gitlab.ListGroupsOptions{
		ListOptions: gitlab.ListOptions{
			PerPage: 100,
			Page:    1,
		},
		AllAvailable:   gitlab.Ptr(true),
		MinAccessLevel: gitlab.Ptr(gitlab.OwnerPermissions),
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
				log.Error().Stack().Err(err).Msg("Failed fetching group variables")
				continue
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
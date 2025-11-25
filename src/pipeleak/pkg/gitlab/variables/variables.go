package variables

import (
	"github.com/CompassSecurity/pipeleak/pkg/gitlab/util"
	"github.com/rs/zerolog/log"
	gitlab "gitlab.com/gitlab-org/api/client-go"
)

// RunFetchVariables fetches and prints all CI/CD variables
func RunFetchVariables(gitlabUrl, gitlabApiToken string) {
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

	err = util.IterateProjects(git, projectOpts, func(project *gitlab.Project) error {
		log.Debug().Str("project", project.WebURL).Msg("Fetch project variables")
		pvs, _, err := git.ProjectVariables.ListVariables(project.ID, nil, nil)
		if err != nil {
			log.Error().Stack().Err(err).Msg("Failed fetching project variables")
			return nil // Continue to next project
		}
		if len(pvs) > 0 {
			log.Warn().Str("project", project.WebURL).Any("variables", pvs).Msg("Project variables")
		}

		fetchPipelineScheduleVariables(git, project)
		return nil
	})
	if err != nil {
		log.Fatal().Stack().Err(err).Msg("Failed iterating projects")
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

func fetchPipelineScheduleVariables(git *gitlab.Client, project *gitlab.Project) {
	scheduleOpts := &gitlab.ListPipelineSchedulesOptions{
		ListOptions: gitlab.ListOptions{
			PerPage: 100,
			Page:    1,
		},
	}

	for {
		schedules, resp, err := git.PipelineSchedules.ListPipelineSchedules(project.ID, scheduleOpts)

		if resp == nil {
			return
		}

		// If we get a 404, the project has no schedules
		if resp.StatusCode == 404 {
			return
		}

		if err != nil {
			log.Error().Stack().Err(err).Int64("project", project.ID).Msg("Failed fetching pipeline schedules")
			break
		}

		for _, schedule := range schedules {
			detailedSchedule, _, err := git.PipelineSchedules.GetPipelineSchedule(project.ID, schedule.ID)
				if err != nil {
				log.Error().Stack().Err(err).Int64("scheduleID", schedule.ID).Msg("Failed fetching pipeline schedule details")
				continue
			}

			if len(detailedSchedule.Variables) > 0 {
				log.Warn().
					Str("project", project.WebURL).
					Str("schedule", detailedSchedule.Description).
					Any("variables", detailedSchedule.Variables).
					Msg("Pipeline schedule variables")
			}
		}

		if resp.NextPage == 0 {
			break
		}
		scheduleOpts.Page = resp.NextPage
	}
}

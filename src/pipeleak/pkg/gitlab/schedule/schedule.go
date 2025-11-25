package schedule

import (
	"github.com/CompassSecurity/pipeleak/pkg/gitlab/util"
	"github.com/rs/zerolog/log"
	gitlab "gitlab.com/gitlab-org/api/client-go"
)

// RunFetchSchedules fetches and prints all scheduled pipelines and their variables
func RunFetchSchedules(gitlabUrl, gitlabApiToken string) {

	log.Info().Msg("Fetching schedules and their variables")

	git, err := util.GetGitlabClient(gitlabApiToken, gitlabUrl)
	if err != nil {
		log.Fatal().Stack().Err(err).Msg("Failed creating gitlab client")
	}

	projectOpts := &gitlab.ListProjectsOptions{
		ListOptions: gitlab.ListOptions{
			PerPage: 100,
			Page:    1,
		},
		// min level to create/edit schedules thus also view their variables
		MinAccessLevel: gitlab.Ptr(gitlab.MaintainerPermissions),
		OrderBy:        gitlab.Ptr("last_activity_at"),
	}

	err = util.IterateProjects(git, projectOpts, func(project *gitlab.Project) error {
		log.Debug().Str("project", project.WebURL).Msg("Fetch project schedules")
		ListPipelineSchedules(git, project)
		return nil
	})
	if err != nil {
		log.Fatal().Stack().Err(err).Msg("Failed iterating projects")
	}

	log.Info().Msg("Fetched all schedules")
}

func ListPipelineSchedules(git *gitlab.Client, project *gitlab.Project) {
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

		// If we get a 404, the project probably has no schedules
		if resp.StatusCode == 404 {
			return
		}

		if err != nil {
			log.Error().Stack().Err(err).Int64("project", project.ID).Msg("Failed fetching project scheduled pipelines")
			break
		}

		for _, schedule := range schedules {
			scheduleWithVars, _, err := git.PipelineSchedules.GetPipelineSchedule(project.ID, schedule.ID)
			if err != nil {
				log.Error().Stack().Err(err).Int64("project", project.ID).Msg("Failed fetching schedule variables")
				continue
			}
			for _, variable := range scheduleWithVars.Variables {
				log.Debug().Str("project", project.WebURL).
					Str("description", schedule.Description).
					Str("cron", schedule.Cron).
					Str("owner", schedule.Owner.Name).
					Str("ownerEmail", schedule.Owner.Email).
					Bool("active", schedule.Active).
					Str("nextRunAt", schedule.NextRunAt.String()).
					Str("createdAt", schedule.CreatedAt.String()).
					Str("updatedAt", schedule.UpdatedAt.String()).
					Msg("Fetch schedule variables")

				log.Warn().Str("project", project.WebURL).Str("description", schedule.Description).Str("key", variable.Key).Str("value", variable.Value).Msg("Schedule variable")
			}
		}

		if resp.NextPage == 0 {
			break
		}
		scheduleOpts.Page = resp.NextPage
	}
}

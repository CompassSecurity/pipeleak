package schedule

import (
	"github.com/CompassSecurity/pipeleak/cmd/gitlab/util"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	gitlab "gitlab.com/gitlab-org/api/client-go"
)

var (
	gitlabApiToken string
	gitlabUrl      string
	verbose        bool
)

func NewScheduleCmd() *cobra.Command {
	scheduleCmd := &cobra.Command{
		Use:   "schedule [no options!]",
		Short: "Enumerate scheduled pipelines and dump their variables",
		Run:   FetchSchedules,
	}
	scheduleCmd.Flags().StringVarP(&gitlabUrl, "gitlab", "g", "", "GitLab instance URL")
	err := scheduleCmd.MarkFlagRequired("gitlab")
	if err != nil {
		log.Fatal().Stack().Err(err).Msg("Unable to require gitlab flag")
	}

	scheduleCmd.Flags().StringVarP(&gitlabApiToken, "token", "t", "", "GitLab API Token")
	err = scheduleCmd.MarkFlagRequired("token")
	if err != nil {
		log.Fatal().Msg("Unable to require token flag")
	}
	scheduleCmd.MarkFlagsRequiredTogether("gitlab", "token")

	scheduleCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "Verbose logging")
	return scheduleCmd
}

func FetchSchedules(cmd *cobra.Command, args []string) {
	if verbose {
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
		log.Debug().Msg("Verbose log output enabled")
	}

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

	for {
		projects, resp, err := git.Projects.ListProjects(projectOpts)
		if err != nil {
			log.Error().Stack().Err(err).Msg("Failed fetching projects")
			break
		}

		for _, project := range projects {
			log.Debug().Str("project", project.WebURL).Msg("Fetch project schedules")
			ListPipelineSchedules(git, project)
		}

		if resp.NextPage == 0 {
			break
		}
		projectOpts.Page = resp.NextPage
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
			log.Error().Stack().Err(err).Int("project", project.ID).Msg("Failed fetching project scheduled pipelines")
			break
		}

		for _, schedule := range schedules {
			scheduleWithVars, _, err := git.PipelineSchedules.GetPipelineSchedule(project.ID, schedule.ID)
			if err != nil {
				log.Error().Stack().Err(err).Int("project", project.ID).Msg("Failed fetching schedule variables")
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

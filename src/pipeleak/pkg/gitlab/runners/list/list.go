package runners

import (
	"github.com/CompassSecurity/pipeleak/pkg/gitlab/util"
	"github.com/rs/zerolog/log"
	gitlab "gitlab.com/gitlab-org/api/client-go"
)

func ListAllAvailableRunners(gitlabUrl string, apiToken string) {
	git, err := util.GetGitlabClient(apiToken, gitlabUrl)
	if err != nil {
		log.Fatal().Stack().Err(err).Msg("Failed creating gitlab client")
	}

	projectRunners := listProjectRunners(git)
	groupRunners := listGroupRunners(git)
	runnerMap := MergeRunnerMaps(projectRunners, groupRunners)

	log.Info().Msg("Listing avaialable runenrs: Runners are only shown once, even when available by multiple source e,g, group or project")

	var runnerDetails []*gitlab.RunnerDetails
	for _, entry := range runnerMap {
		details, _, err := git.Runners.GetRunnerDetails(entry.Runner.ID)

		if err != nil {
			log.Error().Stack().Err(err).Msg("failed getting runner details")
			continue
		}

		runnerDetails = append(runnerDetails, details)
		info := FormatRunnerInfo(entry, details)

		if info.SourceType == "project" {
			log.Info().Str("project", info.SourceName).Str("runner", info.Name).Str("description", info.Description).Str("type", info.Type).Bool("paused", info.Paused).Str("tags", FormatTagsString(info.Tags)).Msg("project runner")
		} else if info.SourceType == "group" {
			log.Info().Str("name", info.SourceName).Str("runner", info.Name).Str("description", info.Description).Str("type", info.Type).Bool("paused", info.Paused).Str("tags", FormatTagsString(info.Tags)).Msg("group runner")
		}
	}

	uniqueTags := ExtractUniqueTags(runnerDetails)
	if len(uniqueTags) > 0 {
		log.Info().Str("tags", FormatTagsString(uniqueTags)).Msg("Unique runner tags")
	}
}

func listProjectRunners(git *gitlab.Client) map[int]RunnerResult {
	runnerMap := make(map[int]RunnerResult)
	projectOpts := &gitlab.ListProjectsOptions{
		ListOptions: gitlab.ListOptions{
			PerPage: 100,
			Page:    1,
		},
		MinAccessLevel: gitlab.Ptr(gitlab.MaintainerPermissions),
	}

	err := util.IterateProjects(git, projectOpts, func(project *gitlab.Project) error {
		log.Debug().Str("name", project.Name).Int("id", project.ID).Msg("List runners for")
		runnerOpts := &gitlab.ListProjectRunnersOptions{
			ListOptions: gitlab.ListOptions{
				PerPage: 100,
				Page:    1,
			},
		}
		runners, _, _ := git.Runners.ListProjectRunners(project.ID, runnerOpts)
		for _, runner := range runners {
			runnerMap[runner.ID] = RunnerResult{Runner: runner, Project: project, Group: nil}
		}
		return nil
	})
	if err != nil {
		log.Error().Stack().Err(err).Msg("Failed iterating projects")
	}

	return runnerMap
}

func listGroupRunners(git *gitlab.Client) map[int]RunnerResult {
	runnerMap := make(map[int]RunnerResult)
	log.Debug().Msg("Logging available groups with at least developer access")

	listGroupsOpts := &gitlab.ListGroupsOptions{
		ListOptions: gitlab.ListOptions{
			PerPage: 100,
			Page:    1,
		},
		AllAvailable:   gitlab.Ptr(true),
		MinAccessLevel: gitlab.Ptr(gitlab.DeveloperPermissions),
	}

	var availableGroups []*gitlab.Group

	for {
		groups, resp, err := git.Groups.ListGroups(listGroupsOpts)
		if err != nil {
			log.Error().Stack().Err(err).Msg("failed listing groups")
		}

		for _, group := range groups {
			log.Debug().Str("name", group.Name).Msg("List runners for")
			availableGroups = append(availableGroups, group)
		}

		if resp.NextPage == 0 {
			break
		}
		listGroupsOpts.Page = resp.NextPage
	}

	listRunnerOpts := &gitlab.ListGroupsRunnersOptions{
		ListOptions: gitlab.ListOptions{
			PerPage: 100,
			Page:    1,
		},
	}

	for _, group := range availableGroups {
		for {
			runners, resp, err := git.Runners.ListGroupsRunners(group.ID, listRunnerOpts)
			if err != nil {
				log.Error().Stack().Err(err).Msg("failed listing group runners")
			}
			for _, runner := range runners {
				runnerMap[runner.ID] = RunnerResult{Runner: runner, Project: nil, Group: group}
			}

			if resp.NextPage == 0 {
				break
			}
			listRunnerOpts.Page = resp.NextPage
		}
	}

	return runnerMap
}

package gitlab

import (
	"strings"

	"github.com/CompassSecurity/pipeleak/helper"
	"github.com/rs/zerolog/log"
	"github.com/xanzy/go-gitlab"
)

type runnerResult struct {
	runner  *gitlab.Runner
	project *gitlab.Project
	group   *gitlab.Group
}

func ListAllAvailableRunners(gitlabUrl string, apiToken string) {
	git, err := helper.GetGitlabClient(apiToken, gitlabUrl)
	if err != nil {
		log.Fatal().Stack().Err(err).Msg("failed creating gitlab client")
	}
	runnerMap := make(map[int]runnerResult)
	runnerMap = listProjectRunners(git, runnerMap)
	runnerMap = listGroupRunners(git, runnerMap)

	log.Info().Msg("Listing avaialable runenrs: Runners are only shown once, even when available by multiple source e,g, group or project")

	runnerTags := make(map[string]bool)
	for _, entry := range runnerMap {
		details, _, err := git.Runners.GetRunnerDetails(entry.runner.ID)

		if err != nil {
			log.Error().Stack().Err(err).Msg("failed getting runner details")
			continue
		}

		for _, tag := range details.TagList {
			runnerTags[tag] = true
		}

		if entry.project != nil {
			log.Info().Str("project", entry.project.Name).Str("runner", details.Name).Str("description", details.Description).Str("type", details.RunnerType).Bool("paused", details.Paused).Str("tags", strings.Join(details.TagList, ",")).Msg("project runner")
		}

		if entry.group != nil {
			log.Info().Str("name", entry.group.Name).Str("runner", details.Name).Str("description", details.Description).Str("type", details.RunnerType).Bool("paused", details.Paused).Str("tags", strings.Join(details.TagList, ",")).Msg("group runner")
		}
	}

	keys := make([]string, 0, len(runnerTags))
	for k := range runnerTags {
		keys = append(keys, k)
	}

	log.Info().Str("tags", strings.Join(keys, ",")).Msg("Unique runner tags")
}

func listProjectRunners(git *gitlab.Client, runnerMap map[int]runnerResult) map[int]runnerResult {
	projectOpts := &gitlab.ListProjectsOptions{
		ListOptions: gitlab.ListOptions{
			PerPage: 100,
			Page:    1,
		},
		MinAccessLevel: gitlab.Ptr(gitlab.MaintainerPermissions),
	}

	for {
		projects, resp, err := git.Projects.ListProjects(projectOpts)
		if err != nil {
			log.Error().Stack().Err(err).Msg("Failed fetching projects")
		}

		for _, project := range projects {
			log.Debug().Str("name", project.Name).Int("id", project.ID).Msg("List runners for")
			runnerOpts := &gitlab.ListProjectRunnersOptions{
				ListOptions: gitlab.ListOptions{
					PerPage: 100,
					Page:    1,
				},
			}
			runners, _, _ := git.Runners.ListProjectRunners(project.ID, runnerOpts)
			for _, runner := range runners {
				runnerMap[runner.ID] = runnerResult{runner: runner, project: project, group: nil}
			}
		}

		if resp.NextPage == 0 {
			break
		}
		projectOpts.Page = resp.NextPage
	}

	return runnerMap

}

func listGroupRunners(git *gitlab.Client, runnerMap map[int]runnerResult) map[int]runnerResult {
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
				runnerMap[runner.ID] = runnerResult{runner: runner, project: nil, group: group}
			}

			if resp.NextPage == 0 {
				break
			}
			listRunnerOpts.Page = resp.NextPage
		}
	}

	return runnerMap
}

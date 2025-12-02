package runners

import (
	"strings"

	"github.com/CompassSecurity/pipeleek/pkg/gitlab/util"
	"github.com/rs/zerolog/log"
	gitlab "gitlab.com/gitlab-org/api/client-go"
)

type RunnerResult struct {
	Runner  *gitlab.Runner
	Project *gitlab.Project
	Group   *gitlab.Group
}

type RunnerInfo struct {
	ID          int
	Name        string
	Description string
	Type        string
	Paused      bool
	Tags        []string
	SourceType  string
	SourceName  string
}

func ListAllAvailableRunners(gitlabUrl string, apiToken string) {
	git, err := util.GetGitlabClient(apiToken, gitlabUrl)
	if err != nil {
		log.Fatal().Stack().Err(err).Msg("Failed creating gitlab client")
	}

	projectRunners := listProjectRunners(git)
	groupRunners := listGroupRunners(git)
	runnerMap := MergeRunnerMaps(projectRunners, groupRunners)

	log.Info().Msg("Listing available runners: runners are only shown once, even when available from multiple sources (e.g., group or project)")

	var runnerDetails []*gitlab.RunnerDetails
	for _, entry := range runnerMap {
		details, _, err := git.Runners.GetRunnerDetails(entry.Runner.ID)

		if err != nil {
			log.Error().Stack().Err(err).Msg("failed getting runner details")
			continue
		}

		runnerDetails = append(runnerDetails, details)
		info := FormatRunnerInfo(entry, details)

		switch info.SourceType {
		case "project":
			log.Info().Str("project", info.SourceName).Str("runner", info.Name).Str("description", info.Description).Str("type", info.Type).Bool("paused", info.Paused).Str("tags", FormatTagsString(info.Tags)).Msg("project runner")
		case "group":
			log.Info().Str("name", info.SourceName).Str("runner", info.Name).Str("description", info.Description).Str("type", info.Type).Bool("paused", info.Paused).Str("tags", FormatTagsString(info.Tags)).Msg("group runner")
		}
	}

	uniqueTags := ExtractUniqueTags(runnerDetails)
	if len(uniqueTags) > 0 {
		log.Info().Str("tags", FormatTagsString(uniqueTags)).Msg("Unique runner tags")
	}
}

func listProjectRunners(git *gitlab.Client) map[int64]RunnerResult {
	runnerMap := make(map[int64]RunnerResult)
	projectOpts := &gitlab.ListProjectsOptions{
		ListOptions: gitlab.ListOptions{
			PerPage: 100,
			Page:    1,
		},
		MinAccessLevel: gitlab.Ptr(gitlab.MaintainerPermissions),
	}

	err := util.IterateProjects(git, projectOpts, func(project *gitlab.Project) error {
		log.Debug().Str("name", project.Name).Int64("id", project.ID).Msg("List runners for")
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

func listGroupRunners(git *gitlab.Client) map[int64]RunnerResult {
	runnerMap := make(map[int64]RunnerResult)
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

// MergeRunnerMaps merges project and group runner maps with deduplication.
// Project runners take precedence over group runners.
func MergeRunnerMaps(projectRunners, groupRunners map[int64]RunnerResult) map[int64]RunnerResult {
	merged := make(map[int64]RunnerResult)

	for id, runner := range projectRunners {
		merged[id] = runner
	}

	for id, runner := range groupRunners {
		if _, exists := merged[id]; !exists {
			merged[id] = runner
		}
	}

	return merged
}

// FormatRunnerInfo formats a RunnerResult and RunnerDetails into a RunnerInfo struct.
func FormatRunnerInfo(result RunnerResult, details *gitlab.RunnerDetails) *RunnerInfo {
	if details == nil {
		return nil
	}

	info := &RunnerInfo{
		ID:          int(details.ID),
		Name:        details.Name,
		Description: details.Description,
		Type:        details.RunnerType,
		Paused:      details.Paused,
		Tags:        details.TagList,
	}

	if result.Project != nil {
		info.SourceType = "project"
		info.SourceName = result.Project.Name
	} else if result.Group != nil {
		info.SourceType = "group"
		info.SourceName = result.Group.Name
	}

	return info
}

// ExtractUniqueTags extracts all unique tags from a list of runner details.
func ExtractUniqueTags(runners []*gitlab.RunnerDetails) []string {
	tagSet := make(map[string]bool)

	for _, runner := range runners {
		for _, tag := range runner.TagList {
			tagSet[tag] = true
		}
	}

	tags := make([]string, 0, len(tagSet))
	for tag := range tagSet {
		tags = append(tags, tag)
	}

	return tags
}

// FormatTagsString formats a slice of tags as a comma-separated string.
func FormatTagsString(tags []string) string {
	return strings.Join(tags, ",")
}

// CountRunnersBySource counts runners by their source type (project or group).
func CountRunnersBySource(runnerMap map[int64]RunnerResult) (projectCount, groupCount int) {
	for _, result := range runnerMap {
		if result.Project != nil {
			projectCount++
		} else if result.Group != nil {
			groupCount++
		}
	}
	return
}

package processor

import (
	"strings"

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

func MergeRunnerMaps(projectRunners, groupRunners map[int]RunnerResult) map[int]RunnerResult {
	merged := make(map[int]RunnerResult)

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

func FormatRunnerInfo(result RunnerResult, details *gitlab.RunnerDetails) *RunnerInfo {
	if details == nil {
		return nil
	}

	info := &RunnerInfo{
		ID:          details.ID,
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

func FormatTagsString(tags []string) string {
	return strings.Join(tags, ",")
}

func CountRunnersBySource(runnerMap map[int]RunnerResult) (projectCount, groupCount int) {
	for _, result := range runnerMap {
		if result.Project != nil {
			projectCount++
		} else if result.Group != nil {
			groupCount++
		}
	}
	return
}

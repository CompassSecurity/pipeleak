package processor

import (
	"strings"

	gitlab "gitlab.com/gitlab-org/api/client-go"
)

// RunnerResult represents a runner with its source (project or group)
type RunnerResult struct {
	Runner  *gitlab.Runner
	Project *gitlab.Project
	Group   *gitlab.Group
}

// RunnerInfo contains formatted information about a runner
type RunnerInfo struct {
	ID          int
	Name        string
	Description string
	Type        string
	Paused      bool
	Tags        []string
	SourceType  string // "project" or "group"
	SourceName  string
}

// MergeRunnerMaps combines project and group runner maps, deduplicating by runner ID
// Returns a merged map where each runner ID appears only once
func MergeRunnerMaps(projectRunners, groupRunners map[int]RunnerResult) map[int]RunnerResult {
	merged := make(map[int]RunnerResult)

	// Add all project runners first
	for id, runner := range projectRunners {
		merged[id] = runner
	}

	// Add group runners, skipping duplicates
	// Project runners take precedence over group runners
	for id, runner := range groupRunners {
		if _, exists := merged[id]; !exists {
			merged[id] = runner
		}
	}

	return merged
}

// FormatRunnerInfo extracts structured information from a RunnerResult
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

// ExtractUniqueTags collects all unique tags from a list of runner details
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

// FormatTagsString joins tags into a comma-separated string
func FormatTagsString(tags []string) string {
	return strings.Join(tags, ",")
}

// CountRunnersBySource counts how many runners come from projects vs groups
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

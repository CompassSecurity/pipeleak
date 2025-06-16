package renovate

import (
	"bytes"
	"strings"

	"github.com/CompassSecurity/pipeleak/cmd/gitlab/util"
	"github.com/CompassSecurity/pipeleak/helper"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	gitlab "gitlab.com/gitlab-org/api/client-go"
	"gopkg.in/yaml.v3"
)

var (
	gitlabApiToken     string
	gitlabUrl          string
	verbose            bool
	owned              bool
	member             bool
	projectSearchQuery string
	fast               bool
)

func NewRenovateCmd() *cobra.Command {
	renovateCmd := &cobra.Command{
		Use:   "renovate [no options!]",
		Short: "Enumerate renovate runner projects",
		Run:   Enumerate,
	}

	renovateCmd.PersistentFlags().StringVarP(&gitlabUrl, "gitlab", "g", "", "GitLab instance URL")
	err := renovateCmd.MarkPersistentFlagRequired("gitlab")
	if err != nil {
		log.Fatal().Stack().Err(err).Msg("Unable to require gitlab flag")
	}

	renovateCmd.PersistentFlags().StringVarP(&gitlabApiToken, "token", "t", "", "GitLab API Token")
	err = renovateCmd.MarkPersistentFlagRequired("token")
	if err != nil {
		log.Error().Stack().Err(err).Msg("Unable to require token flag")
	}
	renovateCmd.MarkFlagsRequiredTogether("gitlab", "token")

	renovateCmd.PersistentFlags().BoolVarP(&owned, "owned", "o", false, "Scan user onwed projects only")
	renovateCmd.PersistentFlags().BoolVarP(&member, "member", "m", false, "Scan projects the user is member of")
	renovateCmd.Flags().StringVarP(&projectSearchQuery, "search", "s", "", "Query string for searching projects")
	renovateCmd.Flags().BoolVarP(&fast, "fast", "f", false, "Fast mode (skip renovate config file detection, only check CIDC yml for renovate bot job)")

	renovateCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "Verbose logging")

	return renovateCmd
}

func Enumerate(cmd *cobra.Command, args []string) {
	helper.SetLogLevel(verbose)
	git, err := util.GetGitlabClient(gitlabApiToken, gitlabUrl)
	if err != nil {
		log.Fatal().Stack().Err(err).Msg("failed creating gitlab client")
	}

	fetchProjects(git)

	log.Info().Msg("Done, Bye Bye 🏳️‍🌈🔥")
}

func fetchProjects(git *gitlab.Client) {
	log.Info().Msg("Fetching projects")

	projectOpts := &gitlab.ListProjectsOptions{
		ListOptions: gitlab.ListOptions{
			PerPage: 100,
			Page:    1,
		},
		OrderBy:    gitlab.Ptr("last_activity_at"),
		Owned:      gitlab.Ptr(owned),
		Membership: gitlab.Ptr(member),
		Search:     gitlab.Ptr(projectSearchQuery),
	}

	for {
		projects, resp, err := git.Projects.ListProjects(projectOpts)
		if err != nil {
			log.Error().Stack().Err(err).Msg("Failed fetching projects")
			break
		}

		for _, project := range projects {
			log.Debug().Str("url", project.WebURL).Msg("Check project")
			identifyRenovateBotJob(git, project)
		}

		if resp.NextPage == 0 {
			break
		}

		projectOpts.Page = resp.NextPage
		log.Info().Int("currentPage", projectOpts.Page).Msg("Fetched projects page")
	}

	log.Info().Msg("Fetched all projects")
}

func identifyRenovateBotJob(git *gitlab.Client, project *gitlab.Project) {

	lintOpts := &gitlab.ProjectLintOptions{
		IncludeJobs: gitlab.Ptr(true),
	}
	res, response, err := git.Validate.ProjectLint(project.ID, lintOpts)

	if response.StatusCode == 404 || response.StatusCode == 403 {
		return // Project does not have a CI/CD configuration or is not accessible
	}

	if err != nil {
		log.Error().Stack().Err(err).Msg("Failed fetching project ci/cd yml")
		return
	}

	hasRenovateConfig, configFile := detectRenovateBotJob(res.MergedYaml, git, project)
	if hasRenovateConfig || configFile != nil {
		log.Warn().Str("pipelines", string(project.BuildsAccessLevel)).Str("url", project.WebURL).Msg("Identified potential self-hosted renovate bot configuration")

		if detectAutodiscover(res.MergedYaml) {
			log.Warn().Str("url", project.WebURL).Msg("Identified potential self-hosted renovate bot configuration with autodiscovery enabled")
		}

		if verbose && hasRenovateConfig {
			yml, err := prettyPrintYAML(res.MergedYaml)
			if err != nil {
				log.Error().Stack().Err(err).Msg("Failed pretty printing project ci/cd yml")
				return
			}
			// make windows compatible
			log.Info().Msg("\n" + yml)
		}
	}
}

func detectRenovateBotJob(cicdConf string, git *gitlab.Client, project *gitlab.Project) (bool, *gitlab.File) {
	// Check for common Renovate bot job identifiers
	hasRenovateConfig := strings.Contains(cicdConf, "renovate/renovate") ||
		strings.Contains(cicdConf, "renovatebot/renovate") ||
		strings.Contains(cicdConf, "renovate-bot/renovate-runner") ||
		strings.Contains(cicdConf, "RENOVATE_")

	if hasRenovateConfig {
		return true, nil
	}

	if !fast && !hasRenovateConfig {
		configFile := detectRenovateConfigFile(git, project)
		if configFile != nil {
			log.Info().Str("file", configFile.FilePath).Str("url", project.WebURL).Msg("Found renovate config file")
			return false, configFile
		}
	}

	return false, nil
}

func detectAutodiscover(cicdConf string) bool {
	// Check for autodiscover flag: https://docs.renovatebot.com/self-hosted-configuration/#autodiscover
	return strings.Contains(cicdConf, "--autodiscover=") ||
		strings.Contains(cicdConf, "RENOVATE_AUTODISCOVER=true")
}

func detectRenovateConfigFile(git *gitlab.Client, project *gitlab.Project) *gitlab.File {
	// https://docs.renovatebot.com/configuration-options/
	configFiles := []string{
		"renovate.json",
		"renovate.json5",
		".github/renovate.json",
		".github/renovate.json5",
		".gitlab/renovate.json",
		".gitlab/renovate.json5",
		".renovaterc",
		".renovaterc.json",
		".renovaterc.json5",
	}

	opts := gitlab.GetFileOptions{Ref: gitlab.Ptr(project.DefaultBranch)}
	for _, configFile := range configFiles {
		file, _, err := git.RepositoryFiles.GetFile(project.ID, configFile, &opts)
		if err != nil {
			continue
		}

		if file != nil {
			return file
		}
	}

	return nil
}

func prettyPrintYAML(yamlStr string) (string, error) {
	var node yaml.Node

	err := yaml.Unmarshal([]byte(yamlStr), &node)
	if err != nil {
		return "", err
	}

	var buf bytes.Buffer
	encoder := yaml.NewEncoder(&buf)
	encoder.SetIndent(2)

	err = encoder.Encode(&node)
	if err != nil {
		return "", err
	}

	return buf.String(), nil
}

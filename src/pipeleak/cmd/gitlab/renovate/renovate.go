package renovate

import (
	"bytes"
	"strings"

	"github.com/CompassSecurity/pipeleak/cmd/gitlab/util"
	"github.com/CompassSecurity/pipeleak/helper"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"gitlab.com/gitlab-org/api/client-go"
	"gopkg.in/yaml.v3"
)

var (
	gitlabApiToken     string
	gitlabUrl          string
	verbose            bool
	owned              bool
	member             bool
	projectSearchQuery string
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
			log.Debug().Str("url", project.WebURL).Msg("Fetch project jobs")
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

	if strings.Contains(res.MergedYaml, "renovate/renovate") || strings.Contains(res.MergedYaml, "--autodiscover=true") {
		log.Info().Str("project", project.Name).Str("url", project.WebURL).Msg("Found renovate bot job image")
		yml, err := prettyPrintYAML(res.MergedYaml)

		if err != nil {
			log.Error().Stack().Err(err).Msg("Failed pretty printing project ci/cd yml")
			return
		}

		// make windows compatible
		log.Info().Msg("\n" + yml)
	}
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

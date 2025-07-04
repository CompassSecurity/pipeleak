package cicd

import (
	"github.com/CompassSecurity/pipeleak/cmd/gitlab/util"
	"github.com/CompassSecurity/pipeleak/helper"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	gitlab "gitlab.com/gitlab-org/api/client-go"
)

var (
	projectName string
)

func NewYamlCmd() *cobra.Command {
	yamlCmd := &cobra.Command{
		Use:   "yaml [no options!]",
		Short: "Fetch full CI/CD yaml of project",
		Run:   Fetch,
	}

	yamlCmd.Flags().StringVarP(&projectName, "repo", "r", "", "Repository to fetch its fully compiled .gitlab-ci.yml")

	return yamlCmd
}

func Fetch(cmd *cobra.Command, args []string) {
	helper.SetLogLevel(verbose)
	git, err := util.GetGitlabClient(gitlabApiToken, gitlabUrl)
	if err != nil {
		log.Fatal().Stack().Err(err).Msg("Failed creating gitlab client")
	}

	project, resp, err := git.Projects.GetProject(projectName, &gitlab.GetProjectOptions{})
	if err != nil {
		log.Fatal().Stack().Err(err).Msg("Failed fetching project by repository name")
	}
	if resp.StatusCode == 404 {
		log.Fatal().Msg("Project not found")
	}

	ciCdYml, err := util.FetchCICDYml(git, project.ID)
	if err != nil {
		log.Fatal().Stack().Err(err).Msg("Failed fetching project CI/CD YML")
	}

	yml, err := helper.PrettyPrintYAML(ciCdYml)
	if err != nil {
		log.Fatal().Stack().Err(err).Msg("Failed pretty printing project CI/CD YML")
	}

	log.Info().Msg(helper.GetPlatformAgnosticNewline() + yml)
}

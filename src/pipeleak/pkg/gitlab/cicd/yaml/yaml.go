package cicd

import (
	"github.com/CompassSecurity/pipeleak/pkg/format"
	"github.com/CompassSecurity/pipeleak/pkg/gitlab/util"
	"github.com/rs/zerolog/log"
	gitlab "gitlab.com/gitlab-org/api/client-go"
)

// DumpCICDYaml fetches and prints the fully compiled CI/CD yaml of a given project
func DumpCICDYaml(gitlabUrl, gitlabApiToken, projectName string) {
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

	yml, err := format.PrettyPrintYAML(ciCdYml)
	if err != nil {
		log.Fatal().Stack().Err(err).Msg("Failed pretty printing project CI/CD YML")
	}

	log.Info().Msg(format.GetPlatformAgnosticNewline() + yml)
}

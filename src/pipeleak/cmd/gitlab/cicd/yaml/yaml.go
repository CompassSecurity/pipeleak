package yaml

import (
	pkgcicd "github.com/CompassSecurity/pipeleak/pkg/gitlab/cicd/yaml"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
)

func NewYamlCmd() *cobra.Command {
	var projectName string

	yamlCmd := &cobra.Command{
		Use:     "yaml",
		Short:   "Dump the CI/CD yaml configuration of a project",
		Long:    "Dump the CI/CD yaml configuration of a project, useful for analyzing the configuration and identifying potential security issues.",
		Example: `pipeleak gl cicd yaml --token glpat-xxxxxxxxxxx --gitlab https://gitlab.mydomain.com --project mygroup/myproject`,
		Run: func(cmd *cobra.Command, args []string) {
			gitlabUrl, _ := cmd.Flags().GetString("gitlab")
			gitlabApiToken, _ := cmd.Flags().GetString("token")
			pkgcicd.DumpCICDYaml(gitlabUrl, gitlabApiToken, projectName)
			log.Info().Msg("Done, Bye Bye 🏳️‍🌈🔥")
		},
	}

	yamlCmd.Flags().StringVarP(&projectName, "project", "p", "", "Project name")
	err := yamlCmd.MarkFlagRequired("project")
	if err != nil {
		log.Fatal().Stack().Err(err).Msg("Unable to require project flag")
	}

	return yamlCmd
}

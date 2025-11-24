package autodiscovery

import (
	pkgrenovate "github.com/CompassSecurity/pipeleak/pkg/gitlab/renovate/autodiscovery"
	"github.com/spf13/cobra"
)

var (
	autodiscoveryRepoName string
	autodiscoveryUsername string
)

func NewAutodiscoveryCmd() *cobra.Command {
	autodiscoveryCmd := &cobra.Command{
		Use:   "autodiscovery",
		Short: "Create a PoC for Renovate Autodiscovery misconfigurations exploitation",
		Long:  "Create a project with a Renovate Bot configuration that will be picked up by an existing Renovate Bot user. The Renovate Bot will then execute the 'prepare' script defined in package.json which you can customize in exploit.sh.",
		Example: `
# Create a project and invite the victim Renovate Bot user to it. Adds a malicious prepare script to package.json which is executed by the Renovate Bot during the renovation process.    
pipeleak gl renovate autodiscovery --token glpat-xxxxxxxxxxx --gitlab https://gitlab.mydomain.com --repo-name my-exploit-repo --username renovate-bot-user
    `,
		Run: func(cmd *cobra.Command, args []string) {
			parent := cmd.Parent()
			gitlabUrl, _ := parent.Flags().GetString("gitlab")
			gitlabApiToken, _ := parent.Flags().GetString("token")
			pkgrenovate.RunGenerate(gitlabUrl, gitlabApiToken, autodiscoveryRepoName, autodiscoveryUsername)
		},
	}
	autodiscoveryCmd.Flags().StringVarP(&autodiscoveryRepoName, "repo-name", "r", "", "The name for the created repository")
	autodiscoveryCmd.Flags().StringVarP(&autodiscoveryUsername, "username", "u", "", "The username of the victim Renovate Bot user to invite")

	return autodiscoveryCmd
}

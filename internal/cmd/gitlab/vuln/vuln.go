package vuln

import (
	pkgvuln "github.com/CompassSecurity/pipeleek/pkg/gitlab/vuln"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
)

var (
	gitlabApiToken string
	gitlabUrl      string
)

func NewVulnCmd() *cobra.Command {
	vulnCmd := &cobra.Command{
		Use:     "vuln",
		Short:   "Check if the installed GitLab version is vulnerable",
		Long:    "Check the installed GitLab instance version against the NIST vulnerability database to see if it is affected by any vulnerabilities.",
		Example: `pipeleek gl vuln --token glpat-xxxxxxxxxxx --gitlab https://gitlab.mydomain.com`,
		Run:     CheckVulns,
	}
	vulnCmd.Flags().StringVarP(&gitlabUrl, "gitlab", "g", "", "GitLab instance URL")
	err := vulnCmd.MarkFlagRequired("gitlab")
	if err != nil {
		log.Fatal().Stack().Err(err).Msg("Unable to require gitlab flag")
	}

	vulnCmd.Flags().StringVarP(&gitlabApiToken, "token", "t", "", "GitLab API Token")
	err = vulnCmd.MarkFlagRequired("token")
	if err != nil {
		log.Fatal().Msg("Unable to require token flag")
	}
	vulnCmd.MarkFlagsRequiredTogether("gitlab", "token")

	return vulnCmd
}

func CheckVulns(cmd *cobra.Command, args []string) {
	pkgvuln.RunCheckVulns(gitlabUrl, gitlabApiToken)
}

package vuln

import (
	"github.com/CompassSecurity/pipeleak/pkg/gitlab/nist"
	"github.com/CompassSecurity/pipeleak/pkg/gitlab/util"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"github.com/tidwall/gjson"
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
		Example: `pipeleak gl vuln --token glpat-xxxxxxxxxxx --gitlab https://gitlab.mydomain.com`,
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
	installedVersion := util.DetermineVersion(gitlabUrl, gitlabApiToken)
	log.Info().Str("version", installedVersion.Version).Msg("GitLab")

	log.Info().Str("version", installedVersion.Version).Msg("Fetching CVEs for this version")
	vulnsJsonStr, err := nist.FetchVulns(installedVersion.Version, installedVersion.Enterprise)
	if err != nil {
		log.Fatal().Msg("Unable fetch vulnerabilities from NIST")
	}

	result := gjson.Get(vulnsJsonStr, "vulnerabilities")
	result.ForEach(func(key, value gjson.Result) bool {
		cve := value.Get("cve.id").String()
		description := value.Get("cve.descriptions.0.value").String()
		log.Info().Str("cve", cve).Str("description", description).Msg("Vulnerable")
		return true
	})

	log.Info().Msg("Finished vuln scan")
}

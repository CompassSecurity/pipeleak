package vuln

import (
	"github.com/CompassSecurity/pipeleak/pkg/gitlab/nist"
	"github.com/CompassSecurity/pipeleak/pkg/gitlab/util"
	"github.com/rs/zerolog/log"
	"github.com/tidwall/gjson"
)

// RunCheckVulns checks the GitLab instance for vulnerabilities
func RunCheckVulns(gitlabUrl, gitlabApiToken string) {
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

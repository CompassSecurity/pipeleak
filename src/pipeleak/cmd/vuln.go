package cmd

import (
	"github.com/CompassSecurity/pipeleak/circl"
	"github.com/CompassSecurity/pipeleak/helper"
	"github.com/Masterminds/semver/v3"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"github.com/tidwall/gjson"
)

var (
	gitlabApiToken string
)

func NewVulnCmd() *cobra.Command {
	vulnCmd := &cobra.Command{
		Use:   "vuln [no options!]",
		Short: "Check if the installed GitLab version is vulnerable",
		Run:   CheckVulns,
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

	vulnCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "Verbose logging")
	return vulnCmd
}

func CheckVulns(cmd *cobra.Command, args []string) {
	if verbose {
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
		log.Debug().Msg("Verbose log output enabled")
	}

	installedVersion := helper.DetermineVersion(gitlabUrl, gitlabApiToken)
	log.Info().Str("version", installedVersion.Version).Msg("GitLab")
	installedVersionSemVer, err := semver.NewVersion(installedVersion.Version)
	if err != nil {
		log.Fatal().Str("version", installedVersion.Version).Msg("Cannot parse installed gitlab version")
	}

	log.Info().Msg("Fetching cve list")
	vulnsJsonStr, err := circl.FetchVulns()
	if err != nil {
		log.Fatal().Msg("Unable fetch vulnerabilities from circl")
	}

	result := gjson.Get(vulnsJsonStr, "cvelistv5")
	result.ForEach(func(key, value gjson.Result) bool {
		cve := value.Get("0").String()
		value.Get("1.containers.cna.affected").ForEach(func(key, affectedEntry gjson.Result) bool {
			affectedEntry.Get("versions").ForEach(func(key, versionEntry gjson.Result) bool {
				if versionEntry.Get("status").String() == "affected" {
					versionContstraint := versionEntry.Get("version").String()

					if "" == versionContstraint {
						log.Debug().Str("semver", versionContstraint).Msg("Empty version constraint, skip")
						return true
					}

					c, err := semver.NewConstraint(versionContstraint)
					if err != nil {
						log.Debug().Str("semver", versionContstraint).Msg("Unable to parse semver constraint")
						return true
					}

					if c.Check(installedVersionSemVer) {
						log.Info().Str("constraint", versionContstraint).Str("cve", cve).Msg("Vulnerable")
					}
				}
				return true
			})
			return true
		})
		return true
	})

	log.Info().Msg("Finished vuln scan")
}

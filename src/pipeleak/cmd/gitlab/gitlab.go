package gitlab

import (
	"github.com/CompassSecurity/pipeleak/cmd/gitlab/cicd"
	"github.com/CompassSecurity/pipeleak/cmd/gitlab/renovate"
	"github.com/CompassSecurity/pipeleak/cmd/gitlab/runners"
	"github.com/CompassSecurity/pipeleak/cmd/gitlab/scan"
	"github.com/CompassSecurity/pipeleak/cmd/gitlab/schedule"
	"github.com/CompassSecurity/pipeleak/cmd/gitlab/secureFiles"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
)

var (
	gitlabApiToken string
	gitlabUrl      string
	verbose        bool
)

func NewGitLabRootCmd() *cobra.Command {
	glCmd := &cobra.Command{
		Use:     "gl [command]",
		Short:   "GitLab related commands",
		GroupID: "GitLab",
	}

	glCmd.AddCommand(scan.NewScanCmd())
	glCmd.AddCommand(NewShodanCmd())
	glCmd.AddCommand(runners.NewRunnersRootCmd())
	glCmd.AddCommand(NewRegisterCmd())
	glCmd.AddCommand(NewVulnCmd())
	glCmd.AddCommand(NewVariablesCmd())
	glCmd.AddCommand(securefiles.NewSecureFilesCmd())
	glCmd.AddCommand(NewEnumCmd())
	glCmd.AddCommand(renovate.NewRenovateRootCmd())
	glCmd.AddCommand(cicd.NewCiCdCmd())
	glCmd.AddCommand(schedule.NewScheduleCmd())

	glCmd.PersistentFlags().StringVarP(&gitlabUrl, "gitlab", "g", "", "GitLab instance URL")
	err := glCmd.MarkPersistentFlagRequired("gitlab")
	if err != nil {
		log.Fatal().Stack().Err(err).Msg("Unable to require gitlab flag")
	}

	glCmd.PersistentFlags().StringVarP(&gitlabApiToken, "token", "t", "", "GitLab API Token")
	err = glCmd.MarkPersistentFlagRequired("token")
	if err != nil {
		log.Error().Stack().Err(err).Msg("Unable to require token flag")
	}
	glCmd.MarkFlagsRequiredTogether("gitlab", "token")

	glCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "Verbose logging")

	glCmd.AddCommand(runners.NewRunnersListCmd())
	glCmd.AddCommand(runners.NewRunnersExploitCmd())

	return glCmd
}

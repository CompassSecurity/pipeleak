package schedule

import (
	pkgschedule "github.com/CompassSecurity/pipeleek/pkg/gitlab/schedule"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
)

var (
	gitlabApiToken string
	gitlabUrl      string
)

func NewScheduleCmd() *cobra.Command {
	scheduleCmd := &cobra.Command{
		Use:     "schedule",
		Short:   "Enumerate scheduled pipelines and dump their variables",
		Long:    "Fetch and print all scheduled pipelines and their variables for projects your token has access to.",
		Example: `pipeleek gl schedule --token glpat-xxxxxxxxxxx --gitlab https://gitlab.mydomain.com`,
		Run:     FetchSchedules,
	}
	scheduleCmd.Flags().StringVarP(&gitlabUrl, "gitlab", "g", "", "GitLab instance URL")
	err := scheduleCmd.MarkFlagRequired("gitlab")
	if err != nil {
		log.Fatal().Stack().Err(err).Msg("Unable to require gitlab flag")
	}

	scheduleCmd.Flags().StringVarP(&gitlabApiToken, "token", "t", "", "GitLab API Token")
	err = scheduleCmd.MarkFlagRequired("token")
	if err != nil {
		log.Fatal().Msg("Unable to require token flag")
	}
	scheduleCmd.MarkFlagsRequiredTogether("gitlab", "token")

	return scheduleCmd
}

func FetchSchedules(cmd *cobra.Command, args []string) {
	pkgschedule.RunFetchSchedules(gitlabUrl, gitlabApiToken)
}

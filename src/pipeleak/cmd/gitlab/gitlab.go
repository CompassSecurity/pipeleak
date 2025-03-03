package gitlab

import (
	"github.com/CompassSecurity/pipeleak/cmd/gitlab/runners"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
)

var (
	gitlabApiToken string
	gitlabUrl      string
	verbose        bool
)

func NewGitLabRootCmd() *cobra.Command {
	runnersCmd := &cobra.Command{
		Use:   "gl [no options!]",
		Short: "GitLab related commands",
	}

	runnersCmd.AddCommand(NewScanCmd())
	runnersCmd.AddCommand(NewShodanCmd())
	runnersCmd.AddCommand(runners.NewRunnersRootCmd())
	runnersCmd.AddCommand(NewRegisterCmd())
	runnersCmd.AddCommand(NewVulnCmd())
	runnersCmd.AddCommand(NewVariablesCmd())
	runnersCmd.AddCommand(NewSecureFilesCmd())
	runnersCmd.AddCommand(NewEnumCmd())

	runnersCmd.PersistentFlags().StringVarP(&gitlabUrl, "gitlab", "g", "", "GitLab instance URL")
	err := runnersCmd.MarkPersistentFlagRequired("gitlab")
	if err != nil {
		log.Fatal().Stack().Err(err).Msg("Unable to require gitlab flag")
	}

	runnersCmd.PersistentFlags().StringVarP(&gitlabApiToken, "token", "t", "", "GitLab API Token")
	err = runnersCmd.MarkPersistentFlagRequired("token")
	if err != nil {
		log.Error().Stack().Err(err).Msg("Unable to require token flag")
	}
	runnersCmd.MarkFlagsRequiredTogether("gitlab", "token")

	runnersCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "Verbose logging")

	runnersCmd.AddCommand(runners.NewRunnersListCmd())
	runnersCmd.AddCommand(runners.NewRunnersExploitCmd())

	return runnersCmd
}

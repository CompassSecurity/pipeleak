package secureFiles

import (
	pkgsecurefiles "github.com/CompassSecurity/pipeleek/pkg/gitlab/secureFiles"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
)

var (
	gitlabApiToken string
	gitlabUrl      string
)

func NewSecureFilesCmd() *cobra.Command {
	secureFilesCmd := &cobra.Command{
		Use:     "secureFiles",
		Short:   "Print CI/CD secure files",
		Long:    "Fetch and print all CI/CD secure files for projects your token has access to.",
		Example: `pipeleek gl secureFiles --token glpat-xxxxxxxxxxx --gitlab https://gitlab.mydomain.com`,
		Run:     FetchSecureFiles,
	}
	secureFilesCmd.Flags().StringVarP(&gitlabUrl, "gitlab", "g", "", "GitLab instance URL")
	err := secureFilesCmd.MarkFlagRequired("gitlab")
	if err != nil {
		log.Fatal().Stack().Err(err).Msg("Unable to require gitlab flag")
	}

	secureFilesCmd.Flags().StringVarP(&gitlabApiToken, "token", "t", "", "GitLab API Token")
	err = secureFilesCmd.MarkFlagRequired("token")
	if err != nil {
		log.Fatal().Msg("Unable to require token flag")
	}
	secureFilesCmd.MarkFlagsRequiredTogether("gitlab", "token")

	return secureFilesCmd
}

func FetchSecureFiles(cmd *cobra.Command, args []string) {
	pkgsecurefiles.RunFetchSecureFiles(gitlabUrl, gitlabApiToken)
}

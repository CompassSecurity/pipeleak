package register

import (
	"github.com/CompassSecurity/pipeleak/cmd/gitlab/util"
	"github.com/CompassSecurity/pipeleak/pkg/logging"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
)

var (
	gitlabUrl string
	verbose   bool
	username  string
	password  string
	email     string
)

func NewRegisterCmd() *cobra.Command {
	registerCmd := &cobra.Command{
		Use:     "register",
		Short:   "Register a new user to a Gitlab instance",
		Long:    "Register a new user to a Gitlab instance that allows self-registration. This command is best effort and might not work.",
		Example: `pipeleak gl register --gitlab https://gitlab.mydomain.com --username newuser --password newpassword --email newuser@example.com`,
		Run:     Register,
	}
	registerCmd.Flags().StringVarP(&gitlabUrl, "gitlab", "g", "", "GitLab instance URL")
	err := registerCmd.MarkFlagRequired("gitlab")
	if err != nil {
		log.Fatal().Stack().Err(err).Msg("Unable to require gitlab flag")
	}

	registerCmd.Flags().StringVarP(&username, "username", "u", "", "Username")
	err = registerCmd.MarkFlagRequired("username")
	if err != nil {
		log.Fatal().Stack().Err(err).Msg("Unable to require username flag")
	}

	registerCmd.Flags().StringVarP(&password, "password", "p", "", "Password")
	err = registerCmd.MarkFlagRequired("password")
	if err != nil {
		log.Fatal().Stack().Err(err).Msg("Unable to require password flag")
	}

	registerCmd.Flags().StringVarP(&email, "email", "e", "", "Email Address")
	err = registerCmd.MarkFlagRequired("email")
	if err != nil {
		log.Fatal().Stack().Err(err).Msg("Unable to require email flag")
	}

	return registerCmd
}

func Register(cmd *cobra.Command, args []string) {
	logging.SetLogLevel(verbose)
	util.RegisterNewAccount(gitlabUrl, username, password, email)
}

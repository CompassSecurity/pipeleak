package gitlab

import (
	"github.com/CompassSecurity/pipeleak/helper"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"github.com/CompassSecurity/pipeleak/cmd/gitlab/util"
)

var (
	username string
	password string
	email    string
)

func NewRegisterCmd() *cobra.Command {
	registerCmd := &cobra.Command{
		Use:   "register [no options!]",
		Short: "Register a new user to a Gitlab instance",
		Run:   Register,
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

	registerCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "Verbose logging")
	return registerCmd
}

func Register(cmd *cobra.Command, args []string) {
	helper.SetLogLevel(verbose)
	util.RegisterNewAccount(gitlabUrl, username, password, email)
}

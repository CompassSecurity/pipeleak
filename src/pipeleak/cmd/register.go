package cmd

import (
	"github.com/CompassSecurity/pipeleak/helper"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
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
		log.Error().Msg("Unable to require gitlab flag: " + err.Error())
	}

	registerCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "Verbose logging")
	return registerCmd
}

func Register(cmd *cobra.Command, args []string) {
	setLogLevel()
	helper.RegisterNewAccount(gitlabUrl)
	log.Info().Msg("Registered, Bye Bye 🏳️‍🌈🔥")
}

package cmd

import (
	"os"
	"time"

	"github.com/CompassSecurity/pipeleak/cmd/bitbucket"
	"github.com/CompassSecurity/pipeleak/cmd/github"
	"github.com/CompassSecurity/pipeleak/cmd/gitlab"
	"github.com/mattn/go-colorable"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
)

var (
	rootCmd = &cobra.Command{
		Use:   "pipeleak",
		Short: "💎💎 Scan job logs and artifacts for secrets 💎💎",
		Long:  "Pipeleak is a tool designed to scan CI/CD job output logs and artifacts for potential secrets. 💎💎",
		PersistentPreRun: func(cmd *cobra.Command, args []string) {
			initLogger()
		},
	}
	JsonLogoutput bool
)

func Execute() error {
	return rootCmd.Execute()
}

func init() {
	rootCmd.AddCommand(github.NewGitHubRootCmd())
	rootCmd.AddCommand(gitlab.NewGitLabRootCmd())
	rootCmd.AddCommand(bitbucket.NewBitBucketRootCmd())
	rootCmd.PersistentFlags().BoolVarP(&JsonLogoutput, "json", "", false, "Use JSON as log output format")
}

func initLogger() {
	log.Logger = zerolog.New(colorable.NewColorableStdout()).With().Timestamp().Logger()
	if !JsonLogoutput {
		output := zerolog.ConsoleWriter{Out: os.Stdout, TimeFormat: time.RFC3339}
		log.Logger = zerolog.New(output).With().Timestamp().Logger()
	}

	zerolog.SetGlobalLevel(zerolog.InfoLevel)
}

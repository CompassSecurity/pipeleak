package cmd

import (
	"os"
	"time"

	"github.com/CompassSecurity/pipeleak/cmd/bitbucket"
	"github.com/CompassSecurity/pipeleak/cmd/devops"
	"github.com/CompassSecurity/pipeleak/cmd/github"
	"github.com/CompassSecurity/pipeleak/cmd/gitlab"
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
	LogFile       string
	LogColor      bool
)

func Execute() error {
	return rootCmd.Execute()
}

func init() {
	rootCmd.AddCommand(github.NewGitHubRootCmd())
	rootCmd.AddCommand(gitlab.NewGitLabRootCmd())
	rootCmd.AddCommand(bitbucket.NewBitBucketRootCmd())
	rootCmd.AddCommand(devops.NewAzureDevOpsRootCmd())
	rootCmd.PersistentFlags().BoolVarP(&JsonLogoutput, "json", "", false, "Use JSON as log output format")
	rootCmd.PersistentFlags().BoolVarP(&LogColor, "coloredLog", "", true, "Output the human-readable log in color")
	rootCmd.PersistentFlags().StringVarP(&LogFile, "logfile", "l", "", "Log output to a file")
}

func initLogger() {
	defaultOut := os.Stdout
	if LogFile != "" {
		runLogFile, err := os.OpenFile(
			LogFile,
			os.O_APPEND|os.O_CREATE|os.O_WRONLY,
			0664,
		)
		if err != nil {
			panic(err)
		}
		defaultOut = runLogFile
	}

	if JsonLogoutput {
		log.Logger = zerolog.New(defaultOut).With().Timestamp().Logger()
	} else {
		output := zerolog.ConsoleWriter{Out: defaultOut, TimeFormat: time.RFC3339, NoColor: !LogColor}
		log.Logger = zerolog.New(output).With().Timestamp().Logger()
	}

	zerolog.SetGlobalLevel(zerolog.InfoLevel)
}

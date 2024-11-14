package cmd

import (
	"os"
	"time"

	"github.com/CompassSecurity/pipeleak/cmd/runners"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
)

var (
	rootCmd = &cobra.Command{
		Use:   "pipeleak",
		Short: "💎💎 Scan GitLab job logs and artifacts for secrets 💎💎",
		Long:  "Pipeleak is a tool designed to scan GitLab job output logs and artifacts for potential secrets. 💎💎",
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
	rootCmd.AddCommand(NewScanCmd())
	rootCmd.AddCommand(NewShodanCmd())
	rootCmd.AddCommand(cmd.NewRunnersRootCmd())
	rootCmd.AddCommand(NewRegisterCmd())
	rootCmd.AddCommand(NewVulnCmd())
	rootCmd.AddCommand(NewVariablesCmd())
	rootCmd.PersistentFlags().BoolVarP(&JsonLogoutput, "json", "", false, "Use JSON as log output format")
}

func initLogger() {
	log.Logger = zerolog.New(os.Stdout).With().Timestamp().Logger()
	if !JsonLogoutput {
		output := zerolog.ConsoleWriter{Out: os.Stdout, TimeFormat: time.RFC3339}
		log.Logger = zerolog.New(output).With().Timestamp().Logger()
	}

	zerolog.SetGlobalLevel(zerolog.InfoLevel)
}

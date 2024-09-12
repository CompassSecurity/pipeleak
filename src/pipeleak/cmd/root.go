package cmd

import (
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"os"
	"time"
)

var (
	rootCmd = &cobra.Command{
		Use:   "pipeleak",
		Short: "💎💎 Scan GitLab job logs and artifacts for secrets 💎💎",
		Long:  "Pipeleak is a tool designed to scan GitLab job output logs and artifacts for potential secrets. 💎💎",
	}
)

// Execute executes the root command.
func Execute() error {
	return rootCmd.Execute()
}

func init() {
	rootCmd.AddCommand(NewScanCmd())
	rootCmd.AddCommand(NewShodanCmd())
	rootCmd.AddCommand(NewRunnersCmd())
	rootCmd.AddCommand(NewRegisterCmd())

	output := zerolog.ConsoleWriter{Out: os.Stdout, TimeFormat: time.RFC3339}
	log.Logger = zerolog.New(output).With().Timestamp().Logger()
	zerolog.SetGlobalLevel(zerolog.InfoLevel)
}

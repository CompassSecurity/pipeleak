package cmd

import (
	"os"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
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
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stdout})
	zerolog.SetGlobalLevel(zerolog.InfoLevel)
}

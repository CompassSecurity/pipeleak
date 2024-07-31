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
		Short: "💎💎 A GitLab Pipeline Output Secret Scanner 💎💎",
		Long:  "Pipeleak scan the output of GitLab pipelines for secrets. Thus detecting what treasures turn up on runtime 💎💎",
	}
)

// Execute executes the root command.
func Execute() error {
	return rootCmd.Execute()
}

func init() {
	rootCmd.AddCommand(NewScanCmd())
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})
	zerolog.SetGlobalLevel(zerolog.InfoLevel)
}

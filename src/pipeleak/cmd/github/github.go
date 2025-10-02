package github

import (
	"github.com/spf13/cobra"
)

func NewGitHubRootCmd() *cobra.Command {
	ghCmd := &cobra.Command{
		Use:     "gh [command]",
		Short:   "GitHub related commands",
		GroupID: "GitHub",
	}

	ghCmd.AddCommand(NewScanCmd())
	ghCmd.AddCommand(NewActionScanCmd())

	return ghCmd
}

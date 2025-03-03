package github

import (
	"github.com/spf13/cobra"
)


func NewGitHubRootCmd() *cobra.Command {
	runnersCmd := &cobra.Command{
		Use:   "gh [no options!]",
		Short: "GitHub related commands",
	}


	return runnersCmd
}

package gitlab

import (
	"github.com/spf13/cobra"
)

func NewGitLabRootUnauthenticatedCmd() *cobra.Command {
	glunaCmd := &cobra.Command{
		Use:     "gluna [command]",
		Short:   "GitLab related commands which do not require authentication",
		Long:    "These commands can be used without providing a GitLab API token, making them suitable for initial reconnaissance and information gathering on GitLab instances.",
		GroupID: "Helper",
	}

	glunaCmd.AddCommand(NewShodanCmd())
	glunaCmd.AddCommand(NewRegisterCmd())

	glunaCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "Verbose logging")

	return glunaCmd
}

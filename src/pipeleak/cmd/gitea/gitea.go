package gitea

import (
	"github.com/spf13/cobra"
)

var (
	giteaApiToken string
	giteaUrl      string
	verbose       bool
)

func NewGiteaRootCmd() *cobra.Command {
	giteaCmd := &cobra.Command{
		Use:     "gitea [command]",
		Short:   "Gitea related commands",
		Long:    "Commands to enumerate and exploit Gitea instances.",
		GroupID: "Gitea",
	}

	giteaCmd.AddCommand(NewEnumCmd())

	giteaCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "Verbose logging")

	return giteaCmd
}

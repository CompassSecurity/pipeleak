package gitea

import (
	"github.com/CompassSecurity/pipeleak/cmd/gitea/enum"
	"github.com/CompassSecurity/pipeleak/cmd/gitea/scan"
	"github.com/spf13/cobra"
)

var (
	verbose       bool
)

func NewGiteaRootCmd() *cobra.Command {
	giteaCmd := &cobra.Command{
		Use:     "gitea [command]",
		Short:   "Gitea related commands",
		Long:    "Commands to enumerate and exploit Gitea instances.",
		GroupID: "Gitea",
	}

	giteaCmd.AddCommand(enum.NewEnumCmd())
	giteaCmd.AddCommand(scan.NewScanCmd())

	giteaCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "Verbose logging")

	return giteaCmd
}

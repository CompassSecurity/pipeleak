package gitea

import (
	"github.com/CompassSecurity/pipeleak/cmd/gitea/enum"
	"github.com/CompassSecurity/pipeleak/cmd/gitea/scan"
	"github.com/spf13/cobra"
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

	return giteaCmd
}

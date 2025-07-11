package devops

import (
	"github.com/spf13/cobra"
)

func NewAzureDevOpsRootCmd() *cobra.Command {
	dvoCmd := &cobra.Command{
		Use:   "ad [command]",
		Short: "Azure DevOps related commands",
	}

	dvoCmd.AddCommand(NewScanCmd())

	return dvoCmd
}

package devops

import (
	"github.com/CompassSecurity/pipeleak/internal/cmd/devops/scan"
	"github.com/spf13/cobra"
)

func NewAzureDevOpsRootCmd() *cobra.Command {
	dvoCmd := &cobra.Command{
		Use:     "ad [command]",
		Short:   "Azure DevOps related commands",
		GroupID: "AzureDevOps",
	}

	dvoCmd.AddCommand(scan.NewScanCmd())

	return dvoCmd
}

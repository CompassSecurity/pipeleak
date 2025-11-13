package bitbucket

import (
	"github.com/CompassSecurity/pipeleak/cmd/bitbucket/scan"
	"github.com/spf13/cobra"
)

func NewBitBucketRootCmd() *cobra.Command {
	bbCmd := &cobra.Command{
		Use:     "bb [command]",
		Short:   "BitBucket related commands",
		GroupID: "BitBucket",
	}

	bbCmd.AddCommand(scan.NewScanCmd())

	return bbCmd
}

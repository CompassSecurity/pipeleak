package bitbucket

import (
	"github.com/spf13/cobra"
)

func NewBitBucketRootCmd() *cobra.Command {
	bbCmd := &cobra.Command{
		Use:     "bb [command]",
		Short:   "BitBucket related commands",
		GroupID: "BitBucket",
	}

	bbCmd.AddCommand(NewScanCmd())

	return bbCmd
}

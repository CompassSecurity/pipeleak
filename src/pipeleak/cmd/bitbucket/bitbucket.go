package bitbucket

import (
	"github.com/spf13/cobra"
)

func NewBitBucketRootCmd() *cobra.Command {
	bbCmd := &cobra.Command{
		Use:   "bb [command]",
		Short: "BitBucket related commands",
	}

	bbCmd.AddCommand(NewScanCmd())

	return bbCmd
}

package main

import (
	"github.com/CompassSecurity/pipeleak/internal/cmd/bitbucket"
	"github.com/CompassSecurity/pipeleak/internal/cmd/common"
	"github.com/spf13/cobra"
)

func main() {
	common.Run(newRootCmd())
}

func newRootCmd() *cobra.Command {
	bbCmd := bitbucket.NewBitBucketRootCmd()
	bbCmd.Use = "pipeleak-bitbucket"
	bbCmd.Short = "Scan BitBucket Pipelines logs and artifacts for secrets"
	bbCmd.Long = `Pipeleak-BitBucket scans CI/CD logs and artifacts to detect leaked secrets and pivot from them.`
	bbCmd.Version = common.Version
	bbCmd.GroupID = ""

	common.SetupPersistentPreRun(bbCmd)
	common.AddCommonFlags(bbCmd)

	bbCmd.SetVersionTemplate(`{{.Version}}
`)

	return bbCmd
}

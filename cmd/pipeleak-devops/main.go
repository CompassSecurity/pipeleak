package main

import (
	"github.com/CompassSecurity/pipeleak/internal/cmd/common"
	"github.com/CompassSecurity/pipeleak/internal/cmd/devops"
	"github.com/spf13/cobra"
)

func main() {
	common.Run(newRootCmd())
}

func newRootCmd() *cobra.Command {
	adCmd := devops.NewAzureDevOpsRootCmd()
	adCmd.Use = "pipeleak-devops"
	adCmd.Short = "Scan Azure DevOps Pipelines logs and artifacts for secrets"
	adCmd.Long = `Pipeleak-DevOps scans CI/CD logs and artifacts to detect leaked secrets and pivot from them.`
	adCmd.Version = common.Version
	adCmd.GroupID = ""

	common.SetupPersistentPreRun(adCmd)
	common.AddCommonFlags(adCmd)

	adCmd.SetVersionTemplate(`{{.Version}}
`)

	return adCmd
}

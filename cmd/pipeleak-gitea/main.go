package main

import (
	"github.com/CompassSecurity/pipeleak/internal/cmd/common"
	"github.com/CompassSecurity/pipeleak/internal/cmd/gitea"
	"github.com/spf13/cobra"
)

func main() {
	common.Run(newRootCmd())
}

func newRootCmd() *cobra.Command {
	giteaCmd := gitea.NewGiteaRootCmd()
	giteaCmd.Use = "pipeleak-gitea"
	giteaCmd.Short = "Scan Gitea Actions logs and artifacts for secrets"
	giteaCmd.Long = `Pipeleak-Gitea scans CI/CD logs and artifacts to detect leaked secrets and pivot from them.`
	giteaCmd.Version = common.Version
	giteaCmd.GroupID = ""

	common.SetupPersistentPreRun(giteaCmd)
	common.AddCommonFlags(giteaCmd)

	giteaCmd.SetVersionTemplate(`{{.Version}}
`)

	return giteaCmd
}

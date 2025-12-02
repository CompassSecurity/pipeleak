package main

import (
	"github.com/CompassSecurity/pipeleek/internal/cmd/common"
	"github.com/CompassSecurity/pipeleek/internal/cmd/github"
	"github.com/spf13/cobra"
)

func main() {
	common.Run(newRootCmd())
}

func newRootCmd() *cobra.Command {
	ghCmd := github.NewGitHubRootCmd()
	ghCmd.Use = "pipeleek-github"
	ghCmd.Short = "Scan GitHub Actions logs and artifacts for secrets"
	ghCmd.Long = `Pipeleek-GitHub scans CI/CD logs and artifacts to detect leaked secrets and pivot from them.`
	ghCmd.Version = common.Version
	ghCmd.GroupID = ""

	common.SetupPersistentPreRun(ghCmd)
	common.AddCommonFlags(ghCmd)

	ghCmd.SetVersionTemplate(`{{.Version}}
`)

	return ghCmd
}

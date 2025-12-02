package main

import (
	"github.com/CompassSecurity/pipeleek/internal/cmd/common"
	"github.com/CompassSecurity/pipeleek/internal/cmd/gitlab"
	"github.com/spf13/cobra"
)

func main() {
	common.Run(newRootCmd())
}

func newRootCmd() *cobra.Command {
	glCmd := gitlab.NewGitLabRootCmd()
	glCmd.Use = "pipeleek-gitlab"
	glCmd.Short = "Scan GitLab CI/CD logs and artifacts for secrets"
	glCmd.Long = `Pipeleek-GitLab scans CI/CD logs and artifacts to detect leaked secrets and pivot from them.`
	glCmd.Version = common.Version
	glCmd.GroupID = ""

	common.SetupPersistentPreRun(glCmd)
	common.AddCommonFlags(glCmd)

	glCmd.SetVersionTemplate(`{{.Version}}
`)

	return glCmd
}

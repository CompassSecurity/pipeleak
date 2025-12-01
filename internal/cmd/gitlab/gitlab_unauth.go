package gitlab

import (
	"github.com/CompassSecurity/pipeleak/internal/cmd/gitlab/register"
	"github.com/CompassSecurity/pipeleak/internal/cmd/gitlab/shodan"
	"github.com/spf13/cobra"
)

func NewGitLabRootUnauthenticatedCmd() *cobra.Command {
	glunaCmd := &cobra.Command{
		Use:     "gluna [command]",
		Short:   "GitLab related commands which do not require authentication",
		Long:    "These commands can be used without providing a GitLab API token, making them suitable for initial reconnaissance and information gathering on GitLab instances.",
		GroupID: "Helper",
	}

	glunaCmd.AddCommand(shodan.NewShodanCmd())
	glunaCmd.AddCommand(register.NewRegisterCmd())

	return glunaCmd
}

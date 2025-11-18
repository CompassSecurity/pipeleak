package runners

import (
	pkgrunners "github.com/CompassSecurity/pipeleak/pkg/gitea/runners"
	"github.com/spf13/cobra"
)

func NewRunnersRootCmd() *cobra.Command {
	return pkgrunners.NewRunnersRootCmd()
}

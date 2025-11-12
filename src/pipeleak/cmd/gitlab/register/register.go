package register

import (
	pkgregister "github.com/CompassSecurity/pipeleak/pkg/gitlab/register"
	"github.com/spf13/cobra"
)

// NewRegisterCmd creates the register command.
func NewRegisterCmd() *cobra.Command {
	return pkgregister.NewRegisterCmd()
}

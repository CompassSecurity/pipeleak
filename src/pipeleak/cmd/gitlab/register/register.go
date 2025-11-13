package register

import (
	pkgregister "github.com/CompassSecurity/pipeleak/pkg/gitlab/register"
	"github.com/spf13/cobra"
)

func NewRegisterCmd() *cobra.Command {
	return pkgregister.NewRegisterCmd()
}

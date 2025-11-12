package enum

import (
	pkgenum "github.com/CompassSecurity/pipeleak/pkg/gitea/enum"
	"github.com/spf13/cobra"
)

// NewEnumCmd creates the enum command.
func NewEnumCmd() *cobra.Command {
	return pkgenum.NewEnumCmd()
}

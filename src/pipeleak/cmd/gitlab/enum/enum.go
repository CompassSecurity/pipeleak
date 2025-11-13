package enum

import (
	pkgenum "github.com/CompassSecurity/pipeleak/pkg/gitlab/enum"
	"github.com/spf13/cobra"
)

func NewEnumCmd() *cobra.Command {
	return pkgenum.NewEnumCmd()
}

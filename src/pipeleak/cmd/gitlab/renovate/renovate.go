package renovate

import (
	pkgrenovate "github.com/CompassSecurity/pipeleak/pkg/gitlab/renovate"
	"github.com/spf13/cobra"
)

func NewRenovateRootCmd() *cobra.Command {
	return pkgrenovate.NewRenovateRootCmd()
}

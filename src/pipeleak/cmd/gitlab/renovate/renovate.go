package renovate

import (
	pkgrenovate "github.com/CompassSecurity/pipeleak/pkg/gitlab/renovate"
	"github.com/spf13/cobra"
)

// NewRenovateRootCmd creates the renovate root command.
func NewRenovateRootCmd() *cobra.Command {
	return pkgrenovate.NewRenovateRootCmd()
}

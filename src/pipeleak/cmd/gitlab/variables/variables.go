package variables

import (
	pkgvariables "github.com/CompassSecurity/pipeleak/pkg/gitlab/variables"
	"github.com/spf13/cobra"
)

// NewVariablesCmd creates the variables command.
func NewVariablesCmd() *cobra.Command {
	return pkgvariables.NewVariablesCmd()
}

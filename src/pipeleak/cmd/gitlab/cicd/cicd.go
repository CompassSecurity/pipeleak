package cicd

import (
	pkgcicd "github.com/CompassSecurity/pipeleak/pkg/gitlab/cicd"
	"github.com/spf13/cobra"
)

// NewCiCdCmd creates the cicd command.
func NewCiCdCmd() *cobra.Command {
	return pkgcicd.NewCiCdCmd()
}

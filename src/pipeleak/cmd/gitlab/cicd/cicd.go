package cicd

import (
	pkgcicd "github.com/CompassSecurity/pipeleak/pkg/gitlab/cicd"
	"github.com/spf13/cobra"
)

func NewCiCdCmd() *cobra.Command {
	return pkgcicd.NewCiCdCmd()
}

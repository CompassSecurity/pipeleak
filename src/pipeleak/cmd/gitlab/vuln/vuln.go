package vuln

import (
	pkgvuln "github.com/CompassSecurity/pipeleak/pkg/gitlab/vuln"
	"github.com/spf13/cobra"
)

func NewVulnCmd() *cobra.Command {
	return pkgvuln.NewVulnCmd()
}

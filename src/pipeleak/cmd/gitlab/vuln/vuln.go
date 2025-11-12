package vuln

import (
	pkgvuln "github.com/CompassSecurity/pipeleak/pkg/gitlab/vuln"
	"github.com/spf13/cobra"
)

// NewVulnCmd creates the vuln command.
func NewVulnCmd() *cobra.Command {
	return pkgvuln.NewVulnCmd()
}

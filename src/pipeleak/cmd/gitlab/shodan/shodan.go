package shodan

import (
	pkgshodan "github.com/CompassSecurity/pipeleak/pkg/gitlab/shodan"
	"github.com/spf13/cobra"
)

// NewShodanCmd creates the shodan command.
func NewShodanCmd() *cobra.Command {
	return pkgshodan.NewShodanCmd()
}

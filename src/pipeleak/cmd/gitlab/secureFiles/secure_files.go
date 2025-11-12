package secureFiles

import (
	pkgsecurefiles "github.com/CompassSecurity/pipeleak/pkg/gitlab/secureFiles"
	"github.com/spf13/cobra"
)

// NewSecureFilesCmd creates the secure files command.
func NewSecureFilesCmd() *cobra.Command {
	return pkgsecurefiles.NewSecureFilesCmd()
}

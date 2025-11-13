package schedule

import (
	pkgschedule "github.com/CompassSecurity/pipeleak/pkg/gitlab/schedule"
	"github.com/spf13/cobra"
)

func NewScheduleCmd() *cobra.Command {
	return pkgschedule.NewScheduleCmd()
}

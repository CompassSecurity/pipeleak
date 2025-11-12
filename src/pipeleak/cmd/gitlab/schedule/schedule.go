package schedule

import (
	pkgschedule "github.com/CompassSecurity/pipeleak/pkg/gitlab/schedule"
	"github.com/spf13/cobra"
)

// NewScheduleCmd creates the schedule command.
func NewScheduleCmd() *cobra.Command {
	return pkgschedule.NewScheduleCmd()
}

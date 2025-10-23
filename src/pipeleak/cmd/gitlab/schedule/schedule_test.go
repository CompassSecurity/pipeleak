package schedule

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewScheduleCmd(t *testing.T) {
	t.Run("creates schedule command", func(t *testing.T) {
		cmd := NewScheduleCmd()
		assert.NotNil(t, cmd)
		assert.Equal(t, "schedule", cmd.Use)
		assert.Equal(t, "Enumerate scheduled pipelines and dump their variables", cmd.Short)
	})

	t.Run("has required flags", func(t *testing.T) {
		cmd := NewScheduleCmd()
		gitlabFlag := cmd.Flags().Lookup("gitlab")
		assert.NotNil(t, gitlabFlag)
		
		tokenFlag := cmd.Flags().Lookup("token")
		assert.NotNil(t, tokenFlag)
	})
}

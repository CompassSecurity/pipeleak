package runners

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewRunnersRootCmd(t *testing.T) {
	t.Run("creates runners root command", func(t *testing.T) {
		cmd := NewRunnersRootCmd()
		assert.NotNil(t, cmd)
		assert.Equal(t, "runners", cmd.Use)
		assert.Equal(t, "runner related commands", cmd.Short)
	})

	t.Run("has persistent flags", func(t *testing.T) {
		cmd := NewRunnersRootCmd()
		gitlabFlag := cmd.PersistentFlags().Lookup("gitlab")
		assert.NotNil(t, gitlabFlag)
		
		tokenFlag := cmd.PersistentFlags().Lookup("token")
		assert.NotNil(t, tokenFlag)
	})
}

func TestNewRunnersListCmd(t *testing.T) {
	t.Run("creates runners list command", func(t *testing.T) {
		cmd := NewRunnersListCmd()
		assert.NotNil(t, cmd)
		assert.Equal(t, "list", cmd.Use)
		assert.Equal(t, "List available runners", cmd.Short)
	})
}

package renovate

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewRenovateRootCmd(t *testing.T) {
	t.Run("creates renovate root command", func(t *testing.T) {
		cmd := NewRenovateRootCmd()
		assert.NotNil(t, cmd)
		assert.Equal(t, "renovate", cmd.Use)
		assert.Equal(t, "Renovate related commands", cmd.Short)
	})

	t.Run("has persistent flags", func(t *testing.T) {
		cmd := NewRenovateRootCmd()
		gitlabFlag := cmd.PersistentFlags().Lookup("gitlab")
		assert.NotNil(t, gitlabFlag)
        
		tokenFlag := cmd.PersistentFlags().Lookup("token")
		assert.NotNil(t, tokenFlag)
	})
}

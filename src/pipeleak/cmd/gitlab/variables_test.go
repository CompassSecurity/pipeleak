package gitlab

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewVariablesCmd(t *testing.T) {
	t.Run("creates variables command", func(t *testing.T) {
		cmd := NewVariablesCmd()
		assert.NotNil(t, cmd)
		assert.Equal(t, "variables", cmd.Use)
		assert.Equal(t, "Print configured CI/CD variables", cmd.Short)
	})

	t.Run("has required flags", func(t *testing.T) {
		cmd := NewVariablesCmd()
		
		gitlabFlag := cmd.Flags().Lookup("gitlab")
		assert.NotNil(t, gitlabFlag)
		
		tokenFlag := cmd.Flags().Lookup("token")
		assert.NotNil(t, tokenFlag)
	})

	t.Run("has verbose flag", func(t *testing.T) {
		cmd := NewVariablesCmd()
		
		verboseFlag := cmd.PersistentFlags().Lookup("verbose")
		assert.NotNil(t, verboseFlag)
		assert.Equal(t, "false", verboseFlag.DefValue)
	})

	t.Run("has Run function assigned", func(t *testing.T) {
		cmd := NewVariablesCmd()
		assert.NotNil(t, cmd.Run)
	})
}

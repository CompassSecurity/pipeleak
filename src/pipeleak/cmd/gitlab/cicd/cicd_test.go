package cicd

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewCiCdCmd(t *testing.T) {
	t.Run("creates cicd command", func(t *testing.T) {
		cmd := NewCiCdCmd()
		assert.NotNil(t, cmd)
		assert.Equal(t, "cicd", cmd.Use)
		assert.Equal(t, "CI/CD related commands", cmd.Short)
	})

	t.Run("has persistent flags", func(t *testing.T) {
		cmd := NewCiCdCmd()
		gitlabFlag := cmd.PersistentFlags().Lookup("gitlab")
		assert.NotNil(t, gitlabFlag)
		
		tokenFlag := cmd.PersistentFlags().Lookup("token")
		assert.NotNil(t, tokenFlag)
		
		verboseFlag := cmd.PersistentFlags().Lookup("verbose")
		assert.NotNil(t, verboseFlag)
	})
}

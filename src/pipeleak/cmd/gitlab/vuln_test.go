package gitlab

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewVulnCmd(t *testing.T) {
	t.Run("creates vuln command", func(t *testing.T) {
		cmd := NewVulnCmd()
		assert.NotNil(t, cmd)
		assert.Equal(t, "vuln", cmd.Use)
		assert.Equal(t, "Check if the installed GitLab version is vulnerable", cmd.Short)
	})

	t.Run("has required flags", func(t *testing.T) {
		cmd := NewVulnCmd()
		
		gitlabFlag := cmd.Flags().Lookup("gitlab")
		assert.NotNil(t, gitlabFlag)
		
		tokenFlag := cmd.Flags().Lookup("token")
		assert.NotNil(t, tokenFlag)
	})

	t.Run("has verbose flag", func(t *testing.T) {
		cmd := NewVulnCmd()
		
		verboseFlag := cmd.PersistentFlags().Lookup("verbose")
		assert.NotNil(t, verboseFlag)
		assert.Equal(t, "false", verboseFlag.DefValue)
	})

	t.Run("has Run function assigned", func(t *testing.T) {
		cmd := NewVulnCmd()
		assert.NotNil(t, cmd.Run)
	})
}

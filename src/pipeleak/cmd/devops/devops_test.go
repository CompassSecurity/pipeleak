package devops

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewAzureDevOpsRootCmd(t *testing.T) {
	t.Run("creates root command", func(t *testing.T) {
		cmd := NewAzureDevOpsRootCmd()
		assert.NotNil(t, cmd)
		assert.Equal(t, "ad [command]", cmd.Use)
		assert.Equal(t, "Azure DevOps related commands", cmd.Short)
		assert.Equal(t, "AzureDevOps", cmd.GroupID)
	})

	t.Run("has scan subcommand", func(t *testing.T) {
		cmd := NewAzureDevOpsRootCmd()
		assert.True(t, cmd.HasSubCommands())
		
		scanCmd, _, err := cmd.Find([]string{"scan"})
		assert.NoError(t, err)
		assert.NotNil(t, scanCmd)
	})
}

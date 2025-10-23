package github

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewGitHubRootCmd(t *testing.T) {
	t.Run("creates root command", func(t *testing.T) {
		cmd := NewGitHubRootCmd()
		assert.NotNil(t, cmd)
		assert.Equal(t, "gh [command]", cmd.Use)
		assert.Equal(t, "GitHub related commands", cmd.Short)
		assert.Equal(t, "GitHub", cmd.GroupID)
	})

	t.Run("has scan subcommand", func(t *testing.T) {
		cmd := NewGitHubRootCmd()
		assert.True(t, cmd.HasSubCommands())

		scanCmd, _, err := cmd.Find([]string{"scan"})
		assert.NoError(t, err)
		assert.NotNil(t, scanCmd)
	})
}

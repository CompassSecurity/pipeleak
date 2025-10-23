package gitlab

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewGitLabRootCmd(t *testing.T) {
	t.Run("creates root command", func(t *testing.T) {
		cmd := NewGitLabRootCmd()
		assert.NotNil(t, cmd)
		assert.Equal(t, "gl [command]", cmd.Use)
		assert.Equal(t, "GitLab related commands", cmd.Short)
		assert.Equal(t, "GitLab", cmd.GroupID)
	})

	t.Run("has subcommands", func(t *testing.T) {
		cmd := NewGitLabRootCmd()
		assert.True(t, cmd.HasSubCommands())
		
		// Verify major subcommands exist
		subcommands := []string{"scan", "runners", "vuln", "variables", "secureFiles", "enum", "renovate", "cicd", "schedule"}
		for _, subcmd := range subcommands {
			found, _, err := cmd.Find([]string{subcmd})
			assert.NoError(t, err)
			assert.NotNil(t, found)
		}
	})

	t.Run("has persistent flags", func(t *testing.T) {
		cmd := NewGitLabRootCmd()
		
		gitlabFlag := cmd.PersistentFlags().Lookup("gitlab")
		assert.NotNil(t, gitlabFlag)
		
		tokenFlag := cmd.PersistentFlags().Lookup("token")
		assert.NotNil(t, tokenFlag)
		
		verboseFlag := cmd.PersistentFlags().Lookup("verbose")
		assert.NotNil(t, verboseFlag)
	})
}

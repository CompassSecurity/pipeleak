package gitlab

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewGitLabRootUnauthenticatedCmd(t *testing.T) {
	t.Run("creates unauthenticated root command", func(t *testing.T) {
		cmd := NewGitLabRootUnauthenticatedCmd()
		assert.NotNil(t, cmd)
		assert.Equal(t, "gluna [command]", cmd.Use)
		assert.Equal(t, "GitLab related commands which do not require authentication", cmd.Short)
		assert.Equal(t, "Helper", cmd.GroupID)
	})

	t.Run("has subcommands", func(t *testing.T) {
		cmd := NewGitLabRootUnauthenticatedCmd()
		assert.True(t, cmd.HasSubCommands())
		
		// Verify subcommands exist
		shodanCmd, _, err := cmd.Find([]string{"shodan"})
		assert.NoError(t, err)
		assert.NotNil(t, shodanCmd)
		
		registerCmd, _, err := cmd.Find([]string{"register"})
		assert.NoError(t, err)
		assert.NotNil(t, registerCmd)
	})

	t.Run("has verbose flag", func(t *testing.T) {
		cmd := NewGitLabRootUnauthenticatedCmd()
		
		verboseFlag := cmd.PersistentFlags().Lookup("verbose")
		assert.NotNil(t, verboseFlag)
		assert.Equal(t, "false", verboseFlag.DefValue)
	})
}

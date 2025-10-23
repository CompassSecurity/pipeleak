package gitlab

import (
	"testing"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
)

func init() {
	zerolog.SetGlobalLevel(zerolog.Disabled)
}

func TestNewEnumCmd(t *testing.T) {
	t.Run("creates enum command", func(t *testing.T) {
		cmd := NewEnumCmd()
		assert.NotNil(t, cmd)
		assert.Equal(t, "enum", cmd.Use)
		assert.Equal(t, "Enumerate access rights of a GitLab access token", cmd.Short)
	})

	t.Run("has required flags", func(t *testing.T) {
		cmd := NewEnumCmd()
		
		gitlabFlag := cmd.Flags().Lookup("gitlab")
		assert.NotNil(t, gitlabFlag)
		
		tokenFlag := cmd.Flags().Lookup("token")
		assert.NotNil(t, tokenFlag)
	})

	t.Run("has optional flags", func(t *testing.T) {
		cmd := NewEnumCmd()
		
		levelFlag := cmd.PersistentFlags().Lookup("level")
		assert.NotNil(t, levelFlag)
		
		verboseFlag := cmd.PersistentFlags().Lookup("verbose")
		assert.NotNil(t, verboseFlag)
	})

	t.Run("has Run function assigned", func(t *testing.T) {
		cmd := NewEnumCmd()
		assert.NotNil(t, cmd.Run)
	})
}

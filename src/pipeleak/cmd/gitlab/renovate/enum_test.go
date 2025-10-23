package renovate

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
		assert.Equal(t, "enum [no options!]", cmd.Use)
		assert.NotNil(t, cmd.Run)
	})

	t.Run("has flags", func(t *testing.T) {
		cmd := NewEnumCmd()
		gitlabFlag := cmd.PersistentFlags().Lookup("gitlab")
		assert.NotNil(t, gitlabFlag)
		
		tokenFlag := cmd.PersistentFlags().Lookup("token")
		assert.NotNil(t, tokenFlag)
		
		repoFlag := cmd.Flags().Lookup("repo")
		assert.NotNil(t, repoFlag)
	})
}

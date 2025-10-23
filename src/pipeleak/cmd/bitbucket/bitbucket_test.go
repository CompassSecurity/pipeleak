package bitbucket

import (
	"testing"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
)

func init() {
	zerolog.SetGlobalLevel(zerolog.Disabled)
}

func TestNewBitBucketRootCmd(t *testing.T) {
	t.Run("creates BitBucket root command", func(t *testing.T) {
		cmd := NewBitBucketRootCmd()
		assert.NotNil(t, cmd)
		assert.Equal(t, "bb [command]", cmd.Use)
		assert.Equal(t, "BitBucket related commands", cmd.Short)
		assert.Equal(t, "BitBucket", cmd.GroupID)
	})

	t.Run("has scan subcommand", func(t *testing.T) {
		cmd := NewBitBucketRootCmd()
		commands := cmd.Commands()
		assert.Greater(t, len(commands), 0)

		hasScan := false
		for _, subcmd := range commands {
			if subcmd.Name() == "scan" {
				hasScan = true
				break
			}
		}
		assert.True(t, hasScan, "should have scan subcommand")
	})
}

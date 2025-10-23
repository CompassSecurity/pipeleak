package devops

import (
	"testing"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
)

func init() {
	zerolog.SetGlobalLevel(zerolog.Disabled)
}

func TestNewScanCmd(t *testing.T) {
	t.Run("creates scan command", func(t *testing.T) {
		cmd := NewScanCmd()
		assert.NotNil(t, cmd)
		assert.Equal(t, "scan [no options!]", cmd.Use)
		assert.Equal(t, "Scan Azure DevOps Actions", cmd.Short)
	})

	t.Run("has flags", func(t *testing.T) {
		cmd := NewScanCmd()

		threadsFlag := cmd.PersistentFlags().Lookup("threads")
		assert.NotNil(t, threadsFlag)

		trufflehogFlag := cmd.PersistentFlags().Lookup("truffleHogVerification")
		assert.NotNil(t, trufflehogFlag)

		maxBuildsFlag := cmd.PersistentFlags().Lookup("maxBuilds")
		assert.NotNil(t, maxBuildsFlag)

		artifactsFlag := cmd.PersistentFlags().Lookup("artifacts")
		assert.NotNil(t, artifactsFlag)

		verboseFlag := cmd.PersistentFlags().Lookup("verbose")
		assert.NotNil(t, verboseFlag)
	})

	t.Run("has default flag values", func(t *testing.T) {
		cmd := NewScanCmd()

		threadsFlag := cmd.PersistentFlags().Lookup("threads")
		assert.Equal(t, "4", threadsFlag.DefValue)

		trufflehogFlag := cmd.PersistentFlags().Lookup("truffleHogVerification")
		assert.Equal(t, "true", trufflehogFlag.DefValue)

		maxBuildsFlag := cmd.PersistentFlags().Lookup("maxBuilds")
		assert.Equal(t, "-1", maxBuildsFlag.DefValue)

		artifactsFlag := cmd.PersistentFlags().Lookup("artifacts")
		assert.Equal(t, "false", artifactsFlag.DefValue)

		verboseFlag := cmd.PersistentFlags().Lookup("verbose")
		assert.Equal(t, "false", verboseFlag.DefValue)
	})

	t.Run("has Run function assigned", func(t *testing.T) {
		cmd := NewScanCmd()
		assert.NotNil(t, cmd.Run)
	})
}

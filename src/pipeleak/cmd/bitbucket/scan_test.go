package bitbucket

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
		assert.Equal(t, "scan", cmd.Use)
		assert.Equal(t, "Scan BitBucket Pipelines", cmd.Short)
		assert.NotEmpty(t, cmd.Long)
		assert.NotEmpty(t, cmd.Example)
	})

	t.Run("has required flags", func(t *testing.T) {
		cmd := NewScanCmd()

		tokenFlag := cmd.Flags().Lookup("token")
		assert.NotNil(t, tokenFlag)
		assert.Equal(t, "t", tokenFlag.Shorthand)

		usernameFlag := cmd.Flags().Lookup("username")
		assert.NotNil(t, usernameFlag)
		assert.Equal(t, "u", usernameFlag.Shorthand)

		cookieFlag := cmd.Flags().Lookup("cookie")
		assert.NotNil(t, cookieFlag)
		assert.Equal(t, "c", cookieFlag.Shorthand)
	})

	t.Run("has optional flags", func(t *testing.T) {
		cmd := NewScanCmd()

		assert.NotNil(t, cmd.Flags().Lookup("confidence"))
		assert.NotNil(t, cmd.PersistentFlags().Lookup("threads"))
		assert.NotNil(t, cmd.PersistentFlags().Lookup("truffleHogVerification"))
		assert.NotNil(t, cmd.PersistentFlags().Lookup("maxPipelines"))
		assert.NotNil(t, cmd.Flags().Lookup("workspace"))
		assert.NotNil(t, cmd.PersistentFlags().Lookup("owned"))
		assert.NotNil(t, cmd.PersistentFlags().Lookup("public"))
		assert.NotNil(t, cmd.PersistentFlags().Lookup("after"))
		assert.NotNil(t, cmd.PersistentFlags().Lookup("verbose"))
		assert.NotNil(t, cmd.PersistentFlags().Lookup("artifacts"))
	})

	t.Run("has default flag values", func(t *testing.T) {
		cmd := NewScanCmd()

		threadsFlag := cmd.PersistentFlags().Lookup("threads")
		assert.Equal(t, "4", threadsFlag.DefValue)

		truffleHogFlag := cmd.PersistentFlags().Lookup("truffleHogVerification")
		assert.Equal(t, "true", truffleHogFlag.DefValue)

		maxPipelinesFlag := cmd.PersistentFlags().Lookup("maxPipelines")
		assert.Equal(t, "-1", maxPipelinesFlag.DefValue)

		verboseFlag := cmd.PersistentFlags().Lookup("verbose")
		assert.Equal(t, "false", verboseFlag.DefValue)

		artifactsFlag := cmd.PersistentFlags().Lookup("artifacts")
		assert.Equal(t, "false", artifactsFlag.DefValue)

		ownedFlag := cmd.PersistentFlags().Lookup("owned")
		assert.Equal(t, "false", ownedFlag.DefValue)

		publicFlag := cmd.PersistentFlags().Lookup("public")
		assert.Equal(t, "false", publicFlag.DefValue)
	})

	t.Run("has Run function assigned", func(t *testing.T) {
		cmd := NewScanCmd()
		assert.NotNil(t, cmd.Run)
	})
}

func TestScanStatus(t *testing.T) {
	t.Run("scanStatus does not panic", func(t *testing.T) {
		// Just verify the function doesn't panic when called
		// The actual event may be nil if logging is disabled
		assert.NotPanics(t, func() {
			_ = scanStatus()
		})
	})
}

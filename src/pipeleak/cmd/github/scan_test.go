package github

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
		assert.Equal(t, "Scan GitHub Actions", cmd.Short)
	})

	t.Run("has flags", func(t *testing.T) {
		cmd := NewScanCmd()

		threadsFlag := cmd.PersistentFlags().Lookup("threads")
		assert.NotNil(t, threadsFlag)

		trufflehogFlag := cmd.PersistentFlags().Lookup("truffleHogVerification")
		assert.NotNil(t, trufflehogFlag)

		maxWorkflowsFlag := cmd.PersistentFlags().Lookup("maxWorkflows")
		assert.NotNil(t, maxWorkflowsFlag)

		artifactsFlag := cmd.PersistentFlags().Lookup("artifacts")
		assert.NotNil(t, artifactsFlag)

		ownedFlag := cmd.PersistentFlags().Lookup("owned")
		assert.NotNil(t, ownedFlag)

		publicFlag := cmd.PersistentFlags().Lookup("public")
		assert.NotNil(t, publicFlag)

		verboseFlag := cmd.PersistentFlags().Lookup("verbose")
		assert.NotNil(t, verboseFlag)
	})

	t.Run("has default flag values", func(t *testing.T) {
		cmd := NewScanCmd()

		threadsFlag := cmd.PersistentFlags().Lookup("threads")
		assert.Equal(t, "4", threadsFlag.DefValue)

		trufflehogFlag := cmd.PersistentFlags().Lookup("truffleHogVerification")
		assert.Equal(t, "true", trufflehogFlag.DefValue)

		maxWorkflowsFlag := cmd.PersistentFlags().Lookup("maxWorkflows")
		assert.Equal(t, "-1", maxWorkflowsFlag.DefValue)

		artifactsFlag := cmd.PersistentFlags().Lookup("artifacts")
		assert.Equal(t, "false", artifactsFlag.DefValue)

		ownedFlag := cmd.PersistentFlags().Lookup("owned")
		assert.Equal(t, "false", ownedFlag.DefValue)

		publicFlag := cmd.PersistentFlags().Lookup("public")
		assert.Equal(t, "false", publicFlag.DefValue)

		verboseFlag := cmd.PersistentFlags().Lookup("verbose")
		assert.Equal(t, "false", verboseFlag.DefValue)
	})

	t.Run("has Run function assigned", func(t *testing.T) {
		cmd := NewScanCmd()
		assert.NotNil(t, cmd.Run)
	})
}

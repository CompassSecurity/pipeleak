package cmd

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTerminalRestorer(t *testing.T) {
	t.Run("TerminalRestorer_can_be_set", func(t *testing.T) {
		originalRestorer := TerminalRestorer
		defer func() { TerminalRestorer = originalRestorer }()

		called := false
		TerminalRestorer = func() { called = true }

		TerminalRestorer()
		assert.True(t, called, "TerminalRestorer should be callable")
	})

	t.Run("TerminalRestorer_nil_safe", func(t *testing.T) {
		originalRestorer := TerminalRestorer
		defer func() { TerminalRestorer = originalRestorer }()

		TerminalRestorer = nil
		assert.NotPanics(t, func() {
			if TerminalRestorer != nil {
				TerminalRestorer()
			}
		})
	})
}

func TestFatalHook(t *testing.T) {
	t.Run("fatal_level_calls_TerminalRestorer", func(t *testing.T) {
		originalRestorer := TerminalRestorer
		defer func() { TerminalRestorer = originalRestorer }()

		called := false
		TerminalRestorer = func() { called = true }

		hook := FatalHook{}

		// Create a dummy event and run the hook with fatal level
		var buf bytes.Buffer
		logger := zerolog.New(&buf)
		event := logger.Fatal()

		hook.Run(event, zerolog.FatalLevel, "test")

		assert.True(t, called, "TerminalRestorer should be called for fatal level")
	})

	t.Run("non_fatal_level_does_not_call_TerminalRestorer", func(t *testing.T) {
		originalRestorer := TerminalRestorer
		defer func() { TerminalRestorer = originalRestorer }()

		called := false
		TerminalRestorer = func() { called = true }

		hook := FatalHook{}

		levels := []zerolog.Level{
			zerolog.TraceLevel,
			zerolog.DebugLevel,
			zerolog.InfoLevel,
			zerolog.WarnLevel,
			zerolog.ErrorLevel,
		}

		var buf bytes.Buffer
		logger := zerolog.New(&buf)
		for _, lvl := range levels {
			event := logger.WithLevel(lvl)
			hook.Run(event, lvl, "test")
			assert.False(t, called, "TerminalRestorer should not be called for non-fatal levels")
		}
	})

	t.Run("nil_TerminalRestorer_does_not_panic", func(t *testing.T) {
		originalRestorer := TerminalRestorer
		defer func() { TerminalRestorer = originalRestorer }()

		TerminalRestorer = nil
		hook := FatalHook{}

		assert.NotPanics(t, func() {
			var buf bytes.Buffer
			logger := zerolog.New(&buf)
			event := logger.Fatal()
			hook.Run(event, zerolog.FatalLevel, "test")
		})
	})
}

func TestCustomWriter_WritesCorrectly(t *testing.T) {
	t.Run("Writes_log_to_file", func(t *testing.T) {
		tmpDir := t.TempDir()
		logFile := filepath.Join(tmpDir, "test.log")

	f, err := os.OpenFile(logFile, os.O_CREATE|os.O_WRONLY, 0644)
	require.NoError(t, err)
	defer func() { _ = f.Close() }()

		writer := &CustomWriter{Writer: f}

		testLog := []byte(`{"level":"info","message":"test"}` + "\n")
		n, err := writer.Write(testLog)

		assert.NoError(t, err)
		assert.Equal(t, len(testLog), n, "Should return original length")

	// Read back and verify
	_ = f.Close()
		content, err := os.ReadFile(logFile)
		require.NoError(t, err)
		assert.Contains(t, string(content), "test", "Log content should be written")
	})
}

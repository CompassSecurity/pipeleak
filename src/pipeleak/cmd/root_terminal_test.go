package cmd

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTerminalRestorer(t *testing.T) {
	t.Run("TerminalRestorer_can_be_set", func(t *testing.T) {
		// Save original value
		originalRestorer := TerminalRestorer
		defer func() { TerminalRestorer = originalRestorer }()

		called := false
		TerminalRestorer = func() {
			called = true
		}

		TerminalRestorer()
		assert.True(t, called, "TerminalRestorer should be callable")
	})

	t.Run("TerminalRestorer_nil_safe", func(t *testing.T) {
		// Save original value
		originalRestorer := TerminalRestorer
		defer func() { TerminalRestorer = originalRestorer }()

		TerminalRestorer = nil
		// Should not panic
		assert.NotPanics(t, func() {
			if TerminalRestorer != nil {
				TerminalRestorer()
			}
		})
	})
}

func TestCustomWriter_DetectsFatalLogs(t *testing.T) {
	t.Run("JSON_fatal_log_calls_TerminalRestorer", func(t *testing.T) {
		// Save original value
		originalRestorer := TerminalRestorer
		defer func() { TerminalRestorer = originalRestorer }()

		called := false
		TerminalRestorer = func() {
			called = true
		}

		tmpFile, err := os.CreateTemp("", "test-log-*.txt")
		require.NoError(t, err)
		defer func() {
			_ = os.Remove(tmpFile.Name())
		}()
		defer func() {
			_ = tmpFile.Close()
		}()

		writer := &CustomWriter{Writer: tmpFile}

		jsonLog := []byte(`{"level":"fatal","message":"test fatal"}` + "\n")
		_, err = writer.Write(jsonLog)

		assert.NoError(t, err)
		assert.True(t, called, "TerminalRestorer should be called for JSON fatal logs")
	})

	t.Run("Console_fatal_log_calls_TerminalRestorer", func(t *testing.T) {
		// Save original value
		originalRestorer := TerminalRestorer
		defer func() { TerminalRestorer = originalRestorer }()

		called := false
		TerminalRestorer = func() {
			called = true
		}

		tmpFile, err := os.CreateTemp("", "test-log-*.txt")
		require.NoError(t, err)
		defer func() {
			_ = os.Remove(tmpFile.Name())
		}()
		defer func() {
			_ = tmpFile.Close()
		}()

		writer := &CustomWriter{Writer: tmpFile}

		// Console-formatted fatal log
		consoleLog := []byte("2025-11-12T10:00:00Z fatal test fatal message\n")
		_, err = writer.Write(consoleLog)

		assert.NoError(t, err)
		assert.True(t, called, "TerminalRestorer should be called for console fatal logs")
	})

	t.Run("Non_fatal_log_does_not_call_TerminalRestorer", func(t *testing.T) {
		// Save original value
		originalRestorer := TerminalRestorer
		defer func() { TerminalRestorer = originalRestorer }()

		called := false
		TerminalRestorer = func() {
			called = true
		}

		tmpFile, err := os.CreateTemp("", "test-log-*.txt")
		require.NoError(t, err)
		defer os.Remove(tmpFile.Name())
		defer tmpFile.Close()

		writer := &CustomWriter{Writer: tmpFile}

		// Info level log
		jsonLog := []byte(`{"level":"info","message":"test info"}` + "\n")
		_, err = writer.Write(jsonLog)

		assert.NoError(t, err)
		assert.False(t, called, "TerminalRestorer should not be called for non-fatal logs")
	})

	t.Run("TerminalRestorer_nil_does_not_panic", func(t *testing.T) {
		// Save original value
		originalRestorer := TerminalRestorer
		defer func() { TerminalRestorer = originalRestorer }()

		TerminalRestorer = nil

		tmpFile, err := os.CreateTemp("", "test-log-*.txt")
		require.NoError(t, err)
		defer os.Remove(tmpFile.Name())
		defer tmpFile.Close()

		writer := &CustomWriter{Writer: tmpFile}

		jsonLog := []byte(`{"level":"fatal","message":"test fatal"}` + "\n")

		assert.NotPanics(t, func() {
			_, _ = writer.Write(jsonLog)
		}, "Should not panic when TerminalRestorer is nil")
	})
}

func TestCustomWriter_WritesCorrectly(t *testing.T) {
	t.Run("Writes_log_to_file", func(t *testing.T) {
		tmpDir := t.TempDir()
		logFile := filepath.Join(tmpDir, "test.log")

		f, err := os.OpenFile(logFile, os.O_CREATE|os.O_WRONLY, 0644)
		require.NoError(t, err)
		defer f.Close()

		writer := &CustomWriter{Writer: f}

		testLog := []byte(`{"level":"info","message":"test"}` + "\n")
		n, err := writer.Write(testLog)

		assert.NoError(t, err)
		assert.Equal(t, len(testLog), n, "Should return original length")

		// Read back and verify
		f.Close()
		content, err := os.ReadFile(logFile)
		require.NoError(t, err)
		assert.Contains(t, string(content), "test", "Log content should be written")
	})
}

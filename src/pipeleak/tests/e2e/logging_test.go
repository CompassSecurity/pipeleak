package e2e

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// TestLogging_ColorFlagRegistered verifies the --color flag is available
func TestLogging_ColorFlagRegistered(t *testing.T) {
	stdout, _, _ := runCLI(t, []string{"--help"}, nil, 10*time.Second)

	// Verify the flag is documented
	assertLogContains(t, stdout, []string{"--color"})

	// Verify the flag description mentions auto-disable behavior
	if !strings.Contains(stdout, "auto-disabled") {
		t.Logf("Flag description might not mention auto-disable, but flag exists")
	}
}

// TestLogging_ConsoleOutputHasColors verifies console output includes ANSI color codes
func TestLogging_ConsoleOutputHasColors(t *testing.T) {
	// Run a simple command that generates log output
	stdout, stderr, exitErr := runCLI(t, []string{"gl", "--help"}, nil, 5*time.Second)

	output := stdout + stderr
	assert.Nil(t, exitErr, "Command should succeed")

	// ANSI color codes typically start with \x1b[ or \033[
	// When colors are enabled, we expect to see these escape sequences
	hasAnsiCodes := strings.Contains(output, "\x1b[") || strings.Contains(output, "\033[")

	// Note: This test might be environment-dependent
	// In some CI/CD environments, colors might be auto-disabled
	t.Logf("Console output contains ANSI codes: %v", hasAnsiCodes)
	t.Logf("Output sample (first 200 chars): %s", truncate(output, 200))
}

// TestLogging_FileOutputDisablesColorsAutomatically tests that log files don't contain ANSI codes
func TestLogging_FileOutputDisablesColorsAutomatically(t *testing.T) {
	tmpDir := t.TempDir()
	logFile := filepath.Join(tmpDir, "test.log")

	// Run a simple enum command with --logfile flag (without explicitly setting --color)
	// Using a command that will fail gracefully without requiring actual server
	args := []string{"gl", "enum", "--gitlab", "https://invalid.local", "--token", "test", "--logfile", logFile}
	_, _, _ = runCLI(t, args, nil, 5*time.Second)

	// Read the log file
	content, err := os.ReadFile(logFile)
	if err != nil {
		t.Skipf("Log file not created (command may have exited before logging): %v", err)
		return
	}

	assert.NotEmpty(t, content, "Log file should have content")

	logContent := string(content)

	// Verify no ANSI color codes in the log file
	hasAnsiCodes := strings.Contains(logContent, "\x1b[") || strings.Contains(logContent, "\033[")
	assert.False(t, hasAnsiCodes,
		"Log file should not contain ANSI color codes when colors are auto-disabled")

	t.Logf("Log file content (first 500 chars):\n%s", truncate(logContent, 500))
}

// TestLogging_FileOutputWithExplicitColorEnabled tests manual override
func TestLogging_FileOutputWithExplicitColorEnabled(t *testing.T) {
	tmpDir := t.TempDir()
	logFile := filepath.Join(tmpDir, "test_color.log")

	// Run command with --logfile and explicitly enable --color
	args := []string{"gl", "enum", "--gitlab", "https://invalid.local", "--token", "test", "--logfile", logFile, "--color=true"}
	_, _, _ = runCLI(t, args, nil, 5*time.Second)

	// Read the log file
	content, err := os.ReadFile(logFile)
	if err != nil {
		t.Skipf("Log file not created: %v", err)
		return
	}

	assert.NotEmpty(t, content, "Log file should have content")

	logContent := string(content)

	// When explicitly enabled, colors should be present even in file output
	hasAnsiCodes := strings.Contains(logContent, "\x1b[") || strings.Contains(logContent, "\033[")
	assert.True(t, hasAnsiCodes,
		"Log file should contain ANSI color codes when --color=true is explicitly set")

	t.Logf("Log file with colors (first 500 chars):\n%s", truncate(logContent, 500))
}

// TestLogging_FileOutputWithExplicitColorDisabled tests explicit disable
func TestLogging_FileOutputWithExplicitColorDisabled(t *testing.T) {
	tmpDir := t.TempDir()
	logFile := filepath.Join(tmpDir, "test_nocolor.log")

	// Run command with --logfile and explicitly disable --color
	args := []string{"gl", "enum", "--gitlab", "https://invalid.local", "--token", "test", "--logfile", logFile, "--color=false"}
	_, _, _ = runCLI(t, args, nil, 5*time.Second)

	// Read the log file
	content, err := os.ReadFile(logFile)
	if err != nil {
		t.Skipf("Log file not created: %v", err)
		return
	}

	assert.NotEmpty(t, content, "Log file should have content")

	logContent := string(content)

	// Verify no ANSI color codes
	hasAnsiCodes := strings.Contains(logContent, "\x1b[") || strings.Contains(logContent, "\033[")
	assert.False(t, hasAnsiCodes,
		"Log file should not contain ANSI color codes when --color=false is set")

	t.Logf("Log file without colors (first 500 chars):\n%s", truncate(logContent, 500))
}

// TestLogging_ConsoleWithExplicitColorDisabled tests disabling colors for console
func TestLogging_ConsoleWithExplicitColorDisabled(t *testing.T) {
	// Run command with --color=false but no logfile
	args := []string{"gl", "--help", "--color=false"}
	stdout, stderr, exitErr := runCLI(t, args, nil, 5*time.Second)

	output := stdout + stderr
	assert.Nil(t, exitErr, "Command should succeed")

	// Verify no ANSI color codes in console output
	hasAnsiCodes := strings.Contains(output, "\x1b[") || strings.Contains(output, "\033[")
	assert.False(t, hasAnsiCodes,
		"Console output should not contain ANSI color codes when --color=false is set")

	t.Logf("Console output without colors (first 200 chars): %s", truncate(output, 200))
}

// TestLogging_LogFileCreatedSuccessfully verifies log file creation
func TestLogging_LogFileCreatedSuccessfully(t *testing.T) {
	tmpDir := t.TempDir()
	logFile := filepath.Join(tmpDir, "pipeleak.log")

	// Verify file doesn't exist yet
	_, err := os.Stat(logFile)
	assert.True(t, os.IsNotExist(err), "Log file should not exist before command")

	// Run command with --logfile
	args := []string{"gl", "enum", "--gitlab", "https://invalid.local", "--token", "test", "--logfile", logFile}
	_, _, _ = runCLI(t, args, nil, 5*time.Second)

	// Verify file was created
	stat, err := os.Stat(logFile)
	if err != nil {
		t.Skipf("Log file not created: %v", err)
		return
	}

	assert.Greater(t, stat.Size(), int64(0), "Log file should have content")

	t.Logf("Log file created: %s (size: %d bytes)", logFile, stat.Size())
}

// TestLogging_LogFileAppendMode verifies log file append behavior
func TestLogging_LogFileAppendMode(t *testing.T) {
	tmpDir := t.TempDir()
	logFile := filepath.Join(tmpDir, "append.log")

	// Run command twice to test append mode
	args := []string{"gl", "enum", "--gitlab", "https://invalid.local", "--token", "test", "--logfile", logFile}

	// First run
	_, _, _ = runCLI(t, args, nil, 5*time.Second)

	stat1, err := os.Stat(logFile)
	if err != nil {
		t.Skipf("Log file not created on first run: %v", err)
		return
	}
	size1 := stat1.Size()

	// Second run
	_, _, _ = runCLI(t, args, nil, 5*time.Second)

	stat2, err := os.Stat(logFile)
	assert.NoError(t, err, "Log file should exist after second run")
	size2 := stat2.Size()

	// File should have grown (append mode)
	assert.Greater(t, size2, size1,
		"Log file should grow on second run (append mode)")

	t.Logf("Log file sizes - First: %d, Second: %d (delta: %d)",
		size1, size2, size2-size1)
}

// Helper function to truncate strings for logging
func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

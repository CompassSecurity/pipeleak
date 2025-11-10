package cmd

import (
	"testing"

	"github.com/rs/zerolog"
	"github.com/spf13/cobra"
)

func TestGlobalVerboseFlagRegistered(t *testing.T) {
	flag := rootCmd.PersistentFlags().Lookup("verbose")
	if flag == nil {
		t.Fatal("Global verbose flag not registered")
	}
}

func TestGlobalLogLevelFlagRegistered(t *testing.T) {
	flag := rootCmd.PersistentFlags().Lookup("log-level")
	if flag == nil {
		t.Fatal("Global log-level flag not registered")
	}
}

func TestSetGlobalLogLevel_VerboseFlag(t *testing.T) {
	LogDebug = true
	LogLevel = ""
	setGlobalLogLevel(nil)
	if zerolog.GlobalLevel() != zerolog.DebugLevel {
		t.Errorf("Expected DebugLevel with -v flag, got %v", zerolog.GlobalLevel())
	}
	// Reset
	LogDebug = false
}

func TestSetGlobalLogLevel_LogLevelDebug(t *testing.T) {
	LogDebug = false
	LogLevel = "debug"
	setGlobalLogLevel(nil)
	if zerolog.GlobalLevel() != zerolog.DebugLevel {
		t.Errorf("Expected DebugLevel, got %v", zerolog.GlobalLevel())
	}
	// Reset
	LogLevel = ""
}

func TestSetGlobalLogLevel_Info(t *testing.T) {
	LogDebug = false
	LogLevel = "info"
	setGlobalLogLevel(nil)
	if zerolog.GlobalLevel() != zerolog.InfoLevel {
		t.Errorf("Expected InfoLevel, got %v", zerolog.GlobalLevel())
	}
	// Reset
	LogLevel = ""
}

func TestSetGlobalLogLevel_Warn(t *testing.T) {
	LogDebug = false
	LogLevel = "warn"
	setGlobalLogLevel(nil)
	if zerolog.GlobalLevel() != zerolog.WarnLevel {
		t.Errorf("Expected WarnLevel, got %v", zerolog.GlobalLevel())
	}
	// Reset
	LogLevel = ""
}

func TestSetGlobalLogLevel_Error(t *testing.T) {
	LogDebug = false
	LogLevel = "error"
	setGlobalLogLevel(nil)
	if zerolog.GlobalLevel() != zerolog.ErrorLevel {
		t.Errorf("Expected ErrorLevel, got %v", zerolog.GlobalLevel())
	}
	// Reset
	LogLevel = ""
}

func TestSetGlobalLogLevel_Default(t *testing.T) {
	LogDebug = false
	LogLevel = ""
	setGlobalLogLevel(nil)
	if zerolog.GlobalLevel() != zerolog.InfoLevel {
		t.Errorf("Expected InfoLevel for default, got %v", zerolog.GlobalLevel())
	}
}

func TestSetGlobalLogLevel_Invalid(t *testing.T) {
	LogDebug = false
	LogLevel = "invalid"
	setGlobalLogLevel(nil)
	if zerolog.GlobalLevel() != zerolog.InfoLevel {
		t.Errorf("Expected InfoLevel for invalid, got %v", zerolog.GlobalLevel())
	}
}

func TestGlobalColorFlagRegistered(t *testing.T) {
	flag := rootCmd.PersistentFlags().Lookup("color")
	if flag == nil {
		t.Fatal("Global color flag not registered")
	}

	// Verify default value is true
	if flag.DefValue != "true" {
		t.Errorf("Expected default value 'true' for color flag, got %s", flag.DefValue)
	}
}

func TestGlobalLogFileFlagRegistered(t *testing.T) {
	flag := rootCmd.PersistentFlags().Lookup("logfile")
	if flag == nil {
		t.Fatal("Global logfile flag not registered")
	}
}

func TestInitLogger_ConsoleDefault(t *testing.T) {
	// Save original values
	origLogFile := LogFile
	origLogColor := LogColor
	origJsonLogoutput := JsonLogoutput

	// Setup: No logfile, colors enabled (default console behavior)
	LogFile = ""
	LogColor = true
	JsonLogoutput = false

	// Create a command without the color flag being explicitly set
	cmd := rootCmd

	// This should initialize with colors enabled
	initLogger(cmd)

	// We can't directly test the logger output, but we can verify the function runs without panic
	// The actual color behavior would be tested in e2e tests

	// Restore original values
	LogFile = origLogFile
	LogColor = origLogColor
	JsonLogoutput = origJsonLogoutput
}

func TestInitLogger_FileOutputAutoDisablesColor(t *testing.T) {
	// Save original values
	origLogFile := LogFile
	origLogColor := LogColor
	origJsonLogoutput := JsonLogoutput

	// Setup: Logfile set, color not explicitly changed
	LogFile = "/tmp/test_pipeleak.log"
	LogColor = true // Default value
	JsonLogoutput = false

	// Create a fresh command to simulate color flag not being explicitly set
	testCmd := &cobra.Command{
		Use: "test",
	}
	testCmd.PersistentFlags().BoolVar(&LogColor, "color", true, "Enable colored log output")

	// This should initialize with colors disabled because logfile is set
	// and color flag was not explicitly changed
	initLogger(testCmd)

	// Restore original values
	LogFile = origLogFile
	LogColor = origLogColor
	JsonLogoutput = origJsonLogoutput
}

func TestInitLogger_FileOutputWithExplicitColor(t *testing.T) {
	// Save original values
	origLogFile := LogFile
	origLogColor := LogColor
	origJsonLogoutput := JsonLogoutput

	// Setup: Logfile set, color explicitly enabled
	LogFile = "/tmp/test_pipeleak_color.log"
	LogColor = true
	JsonLogoutput = false

	// Create a command and simulate the color flag being explicitly set
	testCmd := &cobra.Command{
		Use: "test",
	}
	testCmd.PersistentFlags().BoolVar(&LogColor, "color", true, "Enable colored log output")
	_ = testCmd.PersistentFlags().Set("color", "true") // Explicitly set

	// This should respect the explicit color=true even with logfile
	initLogger(testCmd)

	// The logger should have been initialized (no panic)

	// Restore original values
	LogFile = origLogFile
	LogColor = origLogColor
	JsonLogoutput = origJsonLogoutput
}

func TestInitLogger_JsonOutput(t *testing.T) {
	// Save original values
	origLogFile := LogFile
	origLogColor := LogColor
	origJsonLogoutput := JsonLogoutput

	// Setup: JSON output enabled
	LogFile = ""
	LogColor = true
	JsonLogoutput = true

	cmd := rootCmd

	// This should initialize with JSON logger
	initLogger(cmd)

	// Restore original values
	LogFile = origLogFile
	LogColor = origLogColor
	JsonLogoutput = origJsonLogoutput
}

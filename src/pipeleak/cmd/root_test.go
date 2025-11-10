package cmd

import (
	"testing"

	"github.com/rs/zerolog"
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

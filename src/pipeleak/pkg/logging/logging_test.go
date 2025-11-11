package logging

import (
	"io"
	"testing"

	"github.com/rs/zerolog"
)

func TestSetLogLevel(t *testing.T) {
	tests := []struct {
		name     string
		verbose  bool
		expected zerolog.Level
	}{
		{
			name:     "verbose enabled",
			verbose:  true,
			expected: zerolog.DebugLevel,
		},
		{
			name:     "verbose disabled",
			verbose:  false,
			expected: zerolog.GlobalLevel(),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			originalLevel := zerolog.GlobalLevel()
			defer zerolog.SetGlobalLevel(originalLevel)

			SetLogLevel(tt.verbose)

			if tt.verbose && zerolog.GlobalLevel() != zerolog.DebugLevel {
				t.Errorf("Expected log level to be DebugLevel, got %v", zerolog.GlobalLevel())
			}
		})
	}
}

func TestShortcutStatusFN(t *testing.T) {
	called := false
	statusFn := func() *zerolog.Event {
		called = true
		logger := zerolog.New(io.Discard)
		evt := logger.Info()
		return evt
	}

	event := statusFn()
	if !called {
		t.Error("Expected status function to be called")
	}
	if event == nil {
		t.Error("Expected non-nil zerolog.Event")
	}
}

func TestRegisterStatusHook(t *testing.T) {
	// Reset to default
	statusHook = nil

	customHook := func() *zerolog.Event {
		logger := zerolog.New(io.Discard)
		return logger.Info().Str("custom", "hook")
	}

	RegisterStatusHook(customHook)

	hook := GetStatusHook()
	if hook == nil {
		t.Fatal("Expected status hook to be registered")
	}

	// Verify the hook works
	event := hook()
	if event == nil {
		t.Error("Expected non-nil event from registered hook")
	}
}

func TestGetStatusHook_Default(t *testing.T) {
	// Reset to default
	statusHook = nil

	hook := GetStatusHook()
	if hook == nil {
		t.Fatal("Expected default status hook")
	}

	// Verify default hook works
	event := hook()
	if event == nil {
		t.Error("Expected non-nil event from default hook")
	}
}

func TestDefaultStatusHook(t *testing.T) {
	event := defaultStatusHook()
	if event == nil {
		t.Error("Expected non-nil event from default status hook")
	}
}

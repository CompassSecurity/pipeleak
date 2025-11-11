package logging

import (
	"bytes"
	"io"
	"strings"
	"testing"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
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
	statusHook = nil

	var buf bytes.Buffer
	logger := zerolog.New(&buf)

	customHook := func() *zerolog.Event {
		return logger.Info().Str("custom", "hook")
	}

	RegisterStatusHook(customHook)

	hook := GetStatusHook()
	if hook == nil {
		t.Fatal("Expected status hook to be registered")
	}

	event := hook()
	if event == nil {
		t.Fatal("Expected non-nil event from registered hook")
	}
	event.Msg("Status")

	output := buf.String()
	if !strings.Contains(output, `"custom":"hook"`) {
		t.Errorf("Expected output to contain custom hook data, got: %s", output)
	}
	if !strings.Contains(output, `"message":"Status"`) {
		t.Errorf("Expected output to contain Status message, got: %s", output)
	}
}

func TestGetStatusHook_Default(t *testing.T) {
	statusHook = nil

	hook := GetStatusHook()
	if hook == nil {
		t.Fatal("Expected default status hook")
	}

	var buf bytes.Buffer
	logger := zerolog.New(&buf)
	zerolog.SetGlobalLevel(zerolog.InfoLevel)

	defaultHookFn := defaultStatusHook
	event := defaultHookFn()
	if event == nil {
		t.Fatal("Expected non-nil event from default hook")
	}

	oldLogger := log.Logger
	defer func() { log.Logger = oldLogger }()
	log.Logger = logger

	event = defaultStatusHook()
	event.Msg("Status")

	output := buf.String()
	if !strings.Contains(output, `"status":"nothing to show"`) {
		t.Errorf("Expected default status output to contain 'nothing to show', got: %s", output)
	}
}

func TestDefaultStatusHook(t *testing.T) {
	var buf bytes.Buffer
	logger := zerolog.New(&buf)
	oldLogger := log.Logger
	defer func() { log.Logger = oldLogger }()
	log.Logger = logger

	event := defaultStatusHook()
	if event == nil {
		t.Fatal("Expected non-nil event from default status hook")
	}
	event.Msg("Status")

	output := buf.String()
	if !strings.Contains(output, `"status":"nothing to show"`) {
		t.Errorf("Expected output to contain 'nothing to show', got: %s", output)
	}
	if !strings.Contains(output, `"message":"Status"`) {
		t.Errorf("Expected output to contain Status message, got: %s", output)
	}
}

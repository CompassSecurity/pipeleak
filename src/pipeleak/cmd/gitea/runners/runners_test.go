package runners

import (
	"testing"
)

func TestNewRunnersRootCmd(t *testing.T) {
	cmd := NewRunnersRootCmd()

	if cmd == nil {
		t.Fatal("Expected non-nil command")
	}

	if cmd.Use != "runners" {
		t.Errorf("Expected Use to be 'runners', got %q", cmd.Use)
	}
}

package list

import (
	"testing"
)

func TestNewListCmd(t *testing.T) {
	url := "https://example.com"
	token := "test-token"
	
	cmd := NewListCmd(&url, &token)

	if cmd == nil {
		t.Fatal("Expected non-nil command")
	}

	if cmd.Use != "list" {
		t.Errorf("Expected Use to be 'list', got %q", cmd.Use)
	}

	if cmd.Short == "" {
		t.Error("Expected non-empty Short description")
	}

	if cmd.Long == "" {
		t.Error("Expected non-empty Long description")
	}

	if cmd.Example == "" {
		t.Error("Expected non-empty Example")
	}
}

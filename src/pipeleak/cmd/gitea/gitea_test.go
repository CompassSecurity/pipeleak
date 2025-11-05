package gitea

import (
	"testing"
)

func TestNewGiteaRootCmd(t *testing.T) {
	cmd := NewGiteaRootCmd()

	if cmd == nil {
		t.Fatal("Expected non-nil command")
	}

	if cmd.Use != "gitea [command]" {
		t.Errorf("Expected Use to be 'gitea [command]', got %q", cmd.Use)
	}

	if cmd.Short == "" {
		t.Error("Expected non-empty Short description")
	}

	if cmd.Long == "" {
		t.Error("Expected non-empty Long description")
	}

	if cmd.GroupID != "Gitea" {
		t.Errorf("Expected GroupID 'Gitea', got %q", cmd.GroupID)
	}

	flags := cmd.PersistentFlags()
	if flags.Lookup("verbose") == nil {
		t.Error("Expected 'verbose' persistent flag to exist")
	}

	if len(cmd.Commands()) < 2 {
		t.Errorf("Expected at least 2 subcommands, got %d", len(cmd.Commands()))
	}

	hasEnumCmd := false
	hasScanCmd := false
	for _, subCmd := range cmd.Commands() {
		if subCmd.Use == "enum" {
			hasEnumCmd = true
		}
		if subCmd.Use == "scan" {
			hasScanCmd = true
		}
	}

	if !hasEnumCmd {
		t.Error("Expected 'enum' subcommand to exist")
	}
	if !hasScanCmd {
		t.Error("Expected 'scan' subcommand to exist")
	}
}

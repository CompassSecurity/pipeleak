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

	if cmd.Short == "" {
		t.Error("Expected non-empty Short description")
	}

	if len(cmd.Commands()) != 2 {
		t.Errorf("Expected 2 subcommands (list and exploit), got %d", len(cmd.Commands()))
	}

	hasListCmd := false
	hasExploitCmd := false
	for _, subCmd := range cmd.Commands() {
		if subCmd.Use == "list" {
			hasListCmd = true
		}
		if subCmd.Use == "exploit" {
			hasExploitCmd = true
		}
	}

	if !hasListCmd {
		t.Error("Expected 'list' subcommand to exist")
	}
	if !hasExploitCmd {
		t.Error("Expected 'exploit' subcommand to exist")
	}
}

func TestNewRunnersListCmd(t *testing.T) {
	cmd := NewRunnersListCmd()

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

func TestNewRunnersExploitCmd(t *testing.T) {
	cmd := NewRunnersExploitCmd()

	if cmd == nil {
		t.Fatal("Expected non-nil command")
	}

	if cmd.Use != "exploit" {
		t.Errorf("Expected Use to be 'exploit', got %q", cmd.Use)
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

	flags := cmd.Flags()
	if flags.Lookup("labels") == nil {
		t.Error("Expected 'labels' flag to be defined")
	}
	if flags.Lookup("age-public-key") == nil {
		t.Error("Expected 'age-public-key' flag to be defined")
	}
	if flags.Lookup("repo-name") == nil {
		t.Error("Expected 'repo-name' flag to be defined")
	}

	persistentFlags := cmd.PersistentFlags()
	if persistentFlags.Lookup("dry") == nil {
		t.Error("Expected 'dry' persistent flag to be defined")
	}
	if persistentFlags.Lookup("shell") == nil {
		t.Error("Expected 'shell' persistent flag to be defined")
	}
}

package renovate

import (
	"testing"
)

func TestNewRenovateRootCmd(t *testing.T) {
	cmd := NewRenovateRootCmd()

	if cmd == nil {
		t.Fatal("Expected non-nil command")
	}

	if cmd.Use != "renovate" {
		t.Errorf("Expected Use to be 'renovate', got %q", cmd.Use)
	}

	if cmd.Short == "" {
		t.Error("Expected non-empty Short description")
	}

	if cmd.Long == "" {
		t.Error("Expected non-empty Long description")
	}

	flags := cmd.PersistentFlags()
	if flags.Lookup("gitlab") == nil {
		t.Error("Expected 'gitlab' persistent flag to exist")
	}
	if flags.Lookup("token") == nil {
		t.Error("Expected 'token' persistent flag to exist")
	}
	if flags.Lookup("verbose") == nil {
		t.Error("Expected 'verbose' persistent flag to exist")
	}

	if len(cmd.Commands()) < 3 {
		t.Errorf("Expected at least 3 subcommands, got %d", len(cmd.Commands()))
	}
}

func TestNewEnumCmd(t *testing.T) {
	cmd := NewEnumCmd()

	if cmd == nil {
		t.Fatal("Expected non-nil command")
	}

	if cmd.Use != "enum [no options!]" {
		t.Errorf("Expected Use to be 'enum [no options!]', got %q", cmd.Use)
	}

	if cmd.Short == "" {
		t.Error("Expected non-empty Short description")
	}
}

func TestNewAutodiscoveryCmd(t *testing.T) {
	cmd := NewAutodiscoveryCmd()

	if cmd == nil {
		t.Fatal("Expected non-nil command")
	}

	if cmd.Use != "autodiscovery" {
		t.Errorf("Expected Use to be 'autodiscovery', got %q", cmd.Use)
	}

	if cmd.Short == "" {
		t.Error("Expected non-empty Short description")
	}

	flags := cmd.Flags()
	if flags.Lookup("repoName") == nil {
		t.Error("Expected 'repoName' flag to exist")
	}
	if flags.Lookup("username") == nil {
		t.Error("Expected 'username' flag to exist")
	}
}

func TestNewPrivescCmd(t *testing.T) {
	cmd := NewPrivescCmd()

	if cmd == nil {
		t.Fatal("Expected non-nil command")
	}

	if cmd.Use != "privesc" {
		t.Errorf("Expected Use to be 'privesc', got %q", cmd.Use)
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
	if flags.Lookup("renovateBranchesRegex") == nil {
		t.Error("Expected 'renovateBranchesRegex' flag to exist")
	}
	if flags.Lookup("repoName") == nil {
		t.Error("Expected 'repoName' flag to exist")
	}
}

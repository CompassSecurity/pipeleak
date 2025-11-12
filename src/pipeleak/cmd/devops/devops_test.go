package devops

import (
	"testing"
)

func TestNewAzureDevOpsRootCmd(t *testing.T) {
	cmd := NewAzureDevOpsRootCmd()

	if cmd == nil {
		t.Fatal("Expected non-nil command")
		return
	}

	if cmd.Use != "ad [command]" {
		t.Errorf("Expected Use to be 'ad [command]', got %q", cmd.Use)
	}

	if cmd.Short == "" {
		t.Error("Expected non-empty Short description")
	}

	if cmd.GroupID != "AzureDevOps" {
		t.Errorf("Expected GroupID 'AzureDevOps', got %q", cmd.GroupID)
	}

	if len(cmd.Commands()) < 1 {
		t.Errorf("Expected at least 1 subcommand, got %d", len(cmd.Commands()))
	}

	scanCmd := cmd.Commands()[0]
	if scanCmd.Use != "scan [no options!]" {
		t.Errorf("Expected first subcommand Use to be 'scan [no options!]', got %q", scanCmd.Use)
	}
}

func TestNewScanCmd(t *testing.T) {
	cmd := NewScanCmd()

	if cmd == nil {
		t.Fatal("Expected non-nil command")
	}

	if cmd.Use != "scan [no options!]" {
		t.Errorf("Expected Use to be 'scan [no options!]', got %q", cmd.Use)
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
	persistentFlags := cmd.PersistentFlags()

	if flags.Lookup("token") == nil {
		t.Error("Expected 'token' flag to exist")
	}
	if flags.Lookup("organization") == nil {
		t.Error("Expected 'organization' flag to exist")
	}
	if flags.Lookup("project") == nil {
		t.Error("Expected 'project' flag to exist")
	}
	if flags.Lookup("confidence") == nil {
		t.Error("Expected 'confidence' flag to exist")
	}
	if persistentFlags.Lookup("threads") == nil {
		t.Error("Expected 'threads' persistent flag to exist")
	}
	if persistentFlags.Lookup("truffle-hog-verification") == nil {
		t.Error("Expected 'truffle-hog-verification' persistent flag to exist")
	}
	if persistentFlags.Lookup("max-builds") == nil {
		t.Error("Expected 'max-builds' persistent flag to exist")
	}
	if persistentFlags.Lookup("max-artifact-size") == nil {
		t.Error("Expected 'max-artifact-size' persistent flag to exist")
	}
}

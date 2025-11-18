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

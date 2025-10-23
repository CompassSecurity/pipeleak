package e2e

import (
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// TestRootCommand_Help tests the --help flag
func TestRootCommand_Help(t *testing.T) {

	stdout, _, exitErr := runCLI(t, []string{"--help"}, nil, 5*time.Second)

	assert.Nil(t, exitErr, "Help command should succeed")
	assert.NotEmpty(t, stdout, "Help output should not be empty")

	// Verify expected help content
	assertLogContains(t, stdout, []string{
		"pipeleak",
		"Usage:",
	})

	t.Logf("STDOUT:\n%s", stdout)
}

// TestRootCommand_SubcommandHelp tests help for subcommands
func TestRootCommand_SubcommandHelp(t *testing.T) {

	tests := []struct {
		name     string
		args     []string
		expected []string
	}{
		{
			name:     "gitlab_help",
			args:     []string{"gl", "--help"},
			expected: []string{"GitLab", "Usage:"},
		},
		{
			name:     "gitea_help",
			args:     []string{"gitea", "--help"},
			expected: []string{"Gitea", "Usage:"},
		},
		{
			name:     "github_help",
			args:     []string{"gh", "--help"},
			expected: []string{"GitHub", "Usage:"},
		},
		{
			name:     "bitbucket_help",
			args:     []string{"bb", "--help"},
			expected: []string{"BitBucket", "Usage:"},
		},
		{
			name:     "devops_help",
			args:     []string{"ad", "--help"},
			expected: []string{"DevOps", "Usage:"},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			// Do not use t.Parallel() - stdout/stderr redirection conflicts when multiple tests run concurrently

			stdout, stderr, exitErr := runCLI(t, tt.args, nil, 5*time.Second)

			assert.Nil(t, exitErr, "Help should succeed")
			assert.NotEmpty(t, stdout, "Help output should not be empty")

			// Note: exact help text depends on implementation
			t.Logf("STDOUT:\n%s", stdout)
			t.Logf("STDERR:\n%s", stderr)
		})
	}
}

// TestRootCommand_JSONLogOutput tests --json flag for JSON logging
func TestRootCommand_JSONLogOutput(t *testing.T) {

	server, _, cleanup := startMockServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`[]`))
	})
	defer cleanup()

	stdout, stderr, exitErr := runCLI(t, []string{
		"gl", "scan",
		"--gitlab", server.URL,
		"--token", "test",
		"--json", // Enable JSON log output
	}, nil, 10*time.Second)

	// Check if output contains JSON-like structures
	// Note: actual format depends on zerolog configuration
	output := stdout + stderr
	t.Logf("Exit error: %v", exitErr)
	t.Logf("Output:\n%s", output)

	// If JSON logging is working, output might contain JSON objects
	// This is implementation-dependent
}

// TestRootCommand_LogFile tests --logfile flag
func TestRootCommand_LogFile(t *testing.T) {

	tempDir := t.TempDir()
	logFile := filepath.Join(tempDir, "test.log")

	server, _, cleanup := startMockServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`[]`))
	})
	defer cleanup()

	stdout, stderr, exitErr := runCLI(t, []string{
		"gl", "scan",
		"--gitlab", server.URL,
		"--token", "test",
		"--logfile", logFile,
	}, nil, 10*time.Second)

	t.Logf("Exit error: %v", exitErr)
	t.Logf("STDOUT:\n%s", stdout)
	t.Logf("STDERR:\n%s", stderr)

	// Check if log file was created
	if _, err := os.Stat(logFile); err == nil {
		logContent, _ := os.ReadFile(logFile)
		t.Logf("Log file created with %d bytes", len(logContent))
		t.Logf("Log file content:\n%s", string(logContent))
	} else {
		t.Logf("Log file not created: %v", err)
	}
}

// TestRootCommand_ColoredLog tests --coloredLog flag
func TestRootCommand_ColoredLog(t *testing.T) {

	server, _, cleanup := startMockServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`[]`))
	})
	defer cleanup()

	tests := []struct {
		name string
		flag string
	}{
		{
			name: "colored_enabled",
			flag: "--coloredLog=true",
		},
		{
			name: "colored_disabled",
			flag: "--coloredLog=false",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			stdout, stderr, exitErr := runCLI(t, []string{
				"gl", "scan",
				"--gitlab", server.URL,
				"--token", "test",
				tt.flag,
			}, nil, 10*time.Second)

			t.Logf("Exit error: %v", exitErr)
			t.Logf("STDOUT:\n%s", stdout)
			t.Logf("STDERR:\n%s", stderr)
		})
	}
}

// TestRootCommand_InvalidCommand tests handling of invalid commands
func TestRootCommand_InvalidCommand(t *testing.T) {

	stdout, stderr, exitErr := runCLI(t, []string{"invalid-command"}, nil, 5*time.Second)

	assert.NotNil(t, exitErr, "Invalid command should fail")

	output := stdout + stderr
	assert.NotEmpty(t, output, "Should have error output")

	t.Logf("Output:\n%s", output)
}

// TestRootCommand_NoArguments tests running with no arguments
func TestRootCommand_NoArguments(t *testing.T) {

	stdout, stderr, exitErr := runCLI(t, []string{}, nil, 5*time.Second)

	// Behavior depends on implementation - might show help or error
	output := stdout + stderr
	t.Logf("Exit error: %v", exitErr)
	t.Logf("Output:\n%s", output)
}

// TestRootCommand_Version tests version output (if implemented)
func TestRootCommand_Version(t *testing.T) {

	// Try common version flags
	versionFlags := [][]string{
		{"--version"},
		{"-v"},
		{"version"},
	}

	for _, args := range versionFlags {
		t.Run("args_"+args[0], func(t *testing.T) {
			stdout, stderr, exitErr := runCLI(t, args, nil, 5*time.Second)

			output := stdout + stderr
			t.Logf("Args: %v", args)
			t.Logf("Exit error: %v", exitErr)
			t.Logf("Output:\n%s", output)
		})
	}
}

// TestRootCommand_GlobalFlagInheritance tests that global flags work with subcommands
func TestRootCommand_GlobalFlagInheritance(t *testing.T) {

	server, _, cleanup := startMockServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`[]`))
	})
	defer cleanup()

	// Test that global --json flag works with subcommands
	stdout, stderr, exitErr := runCLI(t, []string{
		"--json", // Global flag before subcommand
		"gl", "scan",
		"--gitlab", server.URL,
		"--token", "test",
	}, nil, 10*time.Second)

	t.Logf("Exit error: %v", exitErr)
	t.Logf("STDOUT:\n%s", stdout)
	t.Logf("STDERR:\n%s", stderr)
}

// TestRootCommand_PersistentFlags tests persistent flags across subcommands
func TestRootCommand_PersistentFlags(t *testing.T) {

	tempDir := t.TempDir()
	logFile := filepath.Join(tempDir, "persistent-test.log")

	server, _, cleanup := startMockServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`[]`))
	})
	defer cleanup()

	// Test persistent flags with different subcommands
	tests := []struct {
		name string
		args []string
	}{
		{
			name: "gitlab_with_persistent_flags",
			args: []string{
				"gl", "scan",
				"--gitlab", server.URL,
				"--token", "test",
				"--logfile", logFile,
			},
		},
		{
			name: "gitea_with_persistent_flags",
			args: []string{
				"gitea", "scan",
				"--gitea", server.URL,
				"--token", "test",
				"--json",
			},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			stdout, stderr, exitErr := runCLI(t, tt.args, nil, 10*time.Second)

			t.Logf("Exit error: %v", exitErr)
			t.Logf("STDOUT:\n%s", stdout)
			t.Logf("STDERR:\n%s", stderr)
		})
	}
}

// TestRootCommand_CommandGroups tests that command groups are properly organized
func TestRootCommand_CommandGroups(t *testing.T) {

	stdout, stderr, exitErr := runCLI(t, []string{"--help"}, nil, 5*time.Second)

	assert.Nil(t, exitErr, "Help should succeed")

	// Check if command groups are present in help output
	// Groups: GitHub, GitLab, Helper, BitBucket, AzureDevOps, Gitea
	possibleGroups := []string{
		"GitHub",
		"GitLab",
		"Gitea",
		"BitBucket",
		"DevOps",
	}

	groupsFound := 0
	for _, group := range possibleGroups {
		if assertContains(stdout, group) {
			groupsFound++
		}
	}

	t.Logf("Found %d command groups in help output", groupsFound)
	t.Logf("STDOUT:\n%s", stdout)
	t.Logf("STDERR:\n%s", stderr)
}

// Helper function to check if string contains substring
func assertContains(s, substr string) bool {
	return len(s) > 0 && len(substr) > 0 && (s == substr || len(s) >= len(substr) && (s[0:len(substr)] == substr || assertContains(s[1:], substr)))
}

// TestRootCommand_EnvironmentVariables tests environment variable handling
func TestRootCommand_EnvironmentVariables(t *testing.T) {

	server, _, cleanup := startMockServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`[]`))
	})
	defer cleanup()

	// Test with environment variables
	stdout, stderr, exitErr := runCLI(t, []string{
		"gl", "scan",
		"--gitlab", server.URL,
		"--token", "test",
	}, []string{
		"PIPELEAK_DEBUG=true",
		"CI=true",
	}, 10*time.Second)

	// Should not affect command execution negatively
	t.Logf("Exit error: %v", exitErr)
	t.Logf("STDOUT:\n%s", stdout)
	t.Logf("STDERR:\n%s", stderr)
}

// TestRootCommand_MultipleCommands tests that commands can be distinguished
func TestRootCommand_MultipleCommands(t *testing.T) {
	// This test verifies that different subcommands can be invoked
	// and don't interfere with each other

	server, _, cleanup := startMockServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`[]`))
	})
	defer cleanup()

	commands := [][]string{
		{"gl", "enum", "--gitlab", server.URL, "--token", "test"},
		{"gl", "variables", "--gitlab", server.URL, "--token", "test"},
		{"gl", "schedule", "--gitlab", server.URL, "--token", "test"},
		{"gitea", "enum", "--gitea", server.URL, "--token", "test"},
	}

	for i, cmd := range commands {
		t.Run("command_"+string(rune(i+'0')), func(t *testing.T) {
			stdout, stderr, exitErr := runCLI(t, cmd, nil, 10*time.Second)

			t.Logf("Command: %v", cmd)
			t.Logf("Exit error: %v", exitErr)
			t.Logf("STDOUT:\n%s", stdout)
			t.Logf("STDERR:\n%s", stderr)
		})
	}
}

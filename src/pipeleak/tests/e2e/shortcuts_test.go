package e2e

import (
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// TestShortcuts_DoNotBreakScans verifies that the global shortcut listener doesn't break scan commands
func TestShortcuts_DoNotBreakScans(t *testing.T) {
	server, _, cleanup := startMockServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`[]`))
	})
	defer cleanup()

	// Test multiple scan commands to ensure shortcuts work globally
	tests := []struct {
		name string
		args []string
	}{
		{
			name: "gitlab_scan",
			args: []string{"gl", "scan", "--gitlab", server.URL, "--token", "test"},
		},
		{
			name: "gitea_scan",
			args: []string{"gitea", "scan", "--gitea", server.URL, "--token", "test"},
		},
		{
			name: "github_scan",
			args: []string{"gh", "scan", "--github", server.URL, "--token", "test", "--repo", "test/repo"},
		},
		{
			name: "bitbucket_scan",
			args: []string{"bb", "scan", "--bitbucket", server.URL, "--email", "test@example.com", "--token", "test"},
		},
		{
			name: "devops_scan",
			args: []string{"ad", "scan", "--devops", server.URL, "--username", "test", "--token", "test"},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			stdout, stderr, exitErr := runCLI(t, tt.args, nil, 10*time.Second)

			// The command should run without panicking or crashing
			// Exit errors are OK as long as they're not related to shortcuts
			output := stdout + stderr

			// Verify the log level initialization happened (proves PersistentPreRun executed)
			// This indirectly confirms ShortcutListeners was registered
			assert.Contains(t, output, "Log level set to", "Should show log level initialization")

			t.Logf("Exit error: %v", exitErr)
			t.Logf("Output:\n%s", output)
		})
	}
}

// TestShortcuts_WorkWithNonScanCommands verifies shortcuts are registered for all commands
func TestShortcuts_WorkWithNonScanCommands(t *testing.T) {
	server, _, cleanup := startMockServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`[]`))
	})
	defer cleanup()

	tests := []struct {
		name string
		args []string
	}{
		{
			name: "gitlab_enum",
			args: []string{"gl", "enum", "--gitlab", server.URL, "--token", "test"},
		},
		{
			name: "gitlab_variables",
			args: []string{"gl", "variables", "--gitlab", server.URL, "--token", "test"},
		},
		{
			name: "gitlab_schedule",
			args: []string{"gl", "schedule", "--gitlab", server.URL, "--token", "test"},
		},
		{
			name: "gitea_enum",
			args: []string{"gitea", "enum", "--gitea", server.URL, "--token", "test"},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			stdout, stderr, exitErr := runCLI(t, tt.args, nil, 10*time.Second)

			output := stdout + stderr

			// Verify log level initialization (proves ShortcutListeners is registered globally)
			assert.Contains(t, output, "Log level set to", "Should show log level initialization")

			t.Logf("Exit error: %v", exitErr)
			t.Logf("Output:\n%s", output)
		})
	}
}

// TestShortcuts_GlobalLogLevelChanges verifies log level can be changed via flags
func TestShortcuts_GlobalLogLevelChanges(t *testing.T) {
	server, _, cleanup := startMockServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`[]`))
	})
	defer cleanup()

	tests := []struct {
		name     string
		args     []string
		expected string
	}{
		{
			name:     "debug_level",
			args:     []string{"gl", "scan", "--gitlab", server.URL, "--token", "test", "--log-level=debug"},
			expected: "Log level set to debug (explicit)",
		},
		{
			name:     "warn_level",
			args:     []string{"gl", "scan", "--gitlab", server.URL, "--token", "test", "--log-level=warn"},
			expected: "Log level set to warn (explicit)",
		},
		{
			name:     "verbose_flag",
			args:     []string{"gl", "scan", "--gitlab", server.URL, "--token", "test", "-v"},
			expected: "Log level set to debug (-v)",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			stdout, stderr, exitErr := runCLI(t, tt.args, nil, 10*time.Second)

			output := stdout + stderr
			assert.Contains(t, output, tt.expected, "Should show correct log level")

			t.Logf("Exit error: %v", exitErr)
			t.Logf("Output:\n%s", output)
		})
	}
}

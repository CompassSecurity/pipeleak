package e2e

import (
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestShortcuts_DoNotBreakScans(t *testing.T) {
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

			output := stdout + stderr

			assert.NotEmpty(t, output, "Output should not be empty")
			assert.Contains(t, output, "Log level set to", "Should show log level initialization")
			assert.NotContains(t, output, "Failed hooking keyboard bindings", 
				"Should not fail to hook keyboard bindings")
			assert.NotContains(t, output, "panic", 
				"Should not contain panic messages")

			t.Logf("Exit error: %v", exitErr)
			t.Logf("Output:\n%s", output)
		})
	}
}

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

			assert.NotEmpty(t, output, "Output should not be empty")
			assert.Contains(t, output, "Log level set to", "Should show log level initialization")
			assert.NotContains(t, output, "Failed hooking keyboard bindings", 
				"Should not fail to hook keyboard bindings")

			t.Logf("Exit error: %v", exitErr)
			t.Logf("Output:\n%s", output)
		})
	}
}

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
			assert.NotEmpty(t, output, "Output should not be empty")

			t.Logf("Exit error: %v", exitErr)
			t.Logf("Output:\n%s", output)
		})
	}
}

func TestShortcuts_NoPanicOnStartup(t *testing.T) {
	stdout, stderr, exitErr := runCLI(t, []string{"--help"}, nil, 5*time.Second)

	output := stdout + stderr
	
	assert.NotEmpty(t, output, "Help output should not be empty")
	assert.Contains(t, output, "pipeleak", "Help should mention pipeleak")
	assert.Contains(t, output, "Usage:", "Help should show usage")
	assert.Nil(t, exitErr, "Help command should succeed without error")
	assert.NotContains(t, output, "panic", "Should not contain panic messages")
	assert.NotContains(t, output, "runtime error", "Should not contain runtime errors")

	t.Logf("Output:\n%s", output)
}

func TestShortcuts_OnlyOneLogLevelMessage(t *testing.T) {
	server, _, cleanup := startMockServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`[]`))
	})
	defer cleanup()

	stdout, stderr, exitErr := runCLI(t, []string{"gl", "scan", "--gitlab", server.URL, "--token", "test"}, nil, 10*time.Second)

	output := stdout + stderr
	
	count := 0
	logLevelMsg := "Log level set to info (default)"
	
	remaining := output
	for {
		idx := len(remaining)
		for i := 0; i <= len(remaining)-len(logLevelMsg); i++ {
			if remaining[i:i+len(logLevelMsg)] == logLevelMsg {
				count++
				idx = i + len(logLevelMsg)
				break
			}
		}
		if idx >= len(remaining) {
			break
		}
		remaining = remaining[idx:]
		if len(remaining) < len(logLevelMsg) {
			break
		}
	}

	assert.Equal(t, 1, count, "Log level initialization message should appear exactly once, not %d times", count)
	assert.NotEmpty(t, output, "Output should not be empty")

	t.Logf("Exit error: %v", exitErr)
	t.Logf("Found %d occurrences of log level message", count)
	t.Logf("Output:\n%s", output)
}

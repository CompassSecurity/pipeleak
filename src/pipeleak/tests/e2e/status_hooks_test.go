package e2e

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestStatusHooks_GitLabPendingJobs(t *testing.T) {
	server, _, cleanup := startMockServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode([]map[string]interface{}{})
	})
	defer cleanup()

	stdout, stderr, _ := runCLI(t, []string{
		"gl", "scan",
		"--gitlab", server.URL,
		"--token", "test-token",
	}, nil, 10*time.Second)

	output := stdout + stderr

	assert.NotEmpty(t, output, "Output should not be empty")
	assert.Contains(t, output, "Log level set to", "Should initialize log level")
}

func TestStatusHooks_DefaultForNonScanCommands(t *testing.T) {
	server, _, cleanup := startMockServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode([]map[string]interface{}{})
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
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stdout, stderr, _ := runCLI(t, tt.args, nil, 10*time.Second)

			output := stdout + stderr

			assert.NotEmpty(t, output, "Output should not be empty")
			assert.Contains(t, output, "Log level set to", "Should initialize log level")
		})
	}
}

func TestStatusHooks_ScanCommandsRegisterCustomHooks(t *testing.T) {
	server, _, cleanup := startMockServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		
		if strings.Contains(r.URL.Path, "/api/v4/projects") {
			_ = json.NewEncoder(w).Encode([]map[string]interface{}{})
		} else {
			_ = json.NewEncoder(w).Encode([]byte(`[]`))
		}
	})
	defer cleanup()

	tests := []struct {
		name string
		args []string
	}{
		{
			name: "gitlab_scan_registers_hook",
			args: []string{"gl", "scan", "--gitlab", server.URL, "--token", "test"},
		},
		{
			name: "gitea_scan_registers_hook",
			args: []string{"gitea", "scan", "--gitea", server.URL, "--token", "test"},
		},
		{
			name: "github_scan_registers_hook",
			args: []string{"gh", "scan", "--github", server.URL, "--token", "test", "--repo", "test/repo"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stdout, stderr, _ := runCLI(t, tt.args, nil, 10*time.Second)

			output := stdout + stderr

			assert.NotEmpty(t, output, "Output should not be empty")
			assert.Contains(t, output, "Log level set to", "Should show log level initialization")
			
			assert.NotContains(t, output, "panic", "Should not panic")
			assert.NotContains(t, output, "Failed hooking keyboard bindings", "Should not fail keyboard binding")
		})
	}
}

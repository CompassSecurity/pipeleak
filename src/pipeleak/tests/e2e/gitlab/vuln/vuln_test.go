//go:build e2e

package e2e

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/CompassSecurity/pipeleak/tests/e2e/internal/testutil"
	"github.com/stretchr/testify/assert"
)

func setupMockGitLabVulnAPI(t *testing.T) string {
	mux := http.NewServeMux()

	// Metadata endpoint (returns GitLab version)
	mux.HandleFunc("/api/v4/metadata", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{
			"version":"15.10.0",
			"revision":"abc123",
			"kas":{"enabled":true,"version":"15.10.0"},
			"enterprise":false
		}`))
	})

	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)
	return server.URL
}

func TestGLVuln(t *testing.T) {
	apiURL := setupMockGitLabVulnAPI(t)
	stdout, stderr, exitErr := testutil.RunCLI(t, []string{
		"gl", "vuln",
		"--gitlab", apiURL,
		"--token", "mock-token",
	}, nil, 15*time.Second)

	assert.Nil(t, exitErr, "Vuln command should succeed")
	assert.Contains(t, stdout, "15.10.0", "Should show GitLab version")
	assert.NotContains(t, stderr, "fatal")
}

func TestGLVuln_MissingToken(t *testing.T) {
	_, stderr, exitErr := testutil.RunCLI(t, []string{
		"gl", "vuln",
		"--gitlab", "https://gitlab.com",
	}, nil, 5*time.Second)

	assert.NotNil(t, exitErr, "Should fail without token")
	assert.Contains(t, stderr, "required flag(s)", "Should mention missing required flag")
}

func TestGLVuln_MissingGitlab(t *testing.T) {
	_, stderr, exitErr := testutil.RunCLI(t, []string{
		"gl", "vuln",
		"--token", "mock-token",
	}, nil, 5*time.Second)

	assert.NotNil(t, exitErr, "Should fail without gitlab URL")
	assert.Contains(t, stderr, "required flag(s)", "Should mention missing required flag")
}

func TestGLVuln_Unauthorized(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v4/metadata", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(`{"message":"401 Unauthorized"}`))
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	stdout, _, _ := testutil.RunCLI(t, []string{
		"gl", "vuln",
		"--gitlab", server.URL,
		"--token", "invalid-token",
	}, nil, 10*time.Second)

	// Vuln command checks NIST database regardless of auth failure
	assert.Contains(t, stdout, "Finished vuln scan", "Should complete vulnerability scan")
}

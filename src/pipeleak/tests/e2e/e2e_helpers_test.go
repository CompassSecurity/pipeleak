package e2e

// Package e2e contains end-to-end tests for the Pipeleak CLI.
//
// These tests run the CLI commands programmatically (in-process) using mock HTTP servers
// to simulate external APIs. All tests are self-contained and do not require external dependencies.
//
// To run tests:
//   go test ./tests/e2e/... -v
//
// To run a specific test:
//   go test ./tests/e2e/... -v -run TestGitLabScan

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"strings"
	"sync"
	"testing"
	"time"
)

// RecordedRequest captures details of an HTTP request received by the mock server
type RecordedRequest struct {
	Method      string
	Path        string
	RawQuery    string
	Headers     http.Header
	Body        []byte
	ReceivedAt  time.Time
	ContentType string
}

// MockServerHandler is a custom handler that records requests
type MockServerHandler struct {
	mu       sync.Mutex
	requests []RecordedRequest
	handler  http.HandlerFunc
}

func (m *MockServerHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Record the request
	bodyBytes, _ := io.ReadAll(r.Body)
	r.Body = io.NopCloser(bytes.NewBuffer(bodyBytes)) // Reset body for handler

	m.mu.Lock()
	m.requests = append(m.requests, RecordedRequest{
		Method:      r.Method,
		Path:        r.URL.Path,
		RawQuery:    r.URL.RawQuery,
		Headers:     r.Header.Clone(),
		Body:        bodyBytes,
		ReceivedAt:  time.Now(),
		ContentType: r.Header.Get("Content-Type"),
	})
	m.mu.Unlock()

	// Call the actual handler
	m.handler(w, r)
}

func (m *MockServerHandler) GetRequests() []RecordedRequest {
	m.mu.Lock()
	defer m.mu.Unlock()
	return append([]RecordedRequest{}, m.requests...)
}

func (m *MockServerHandler) Reset() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.requests = []RecordedRequest{}
}

// startMockServer creates a new HTTP test server with request recording
//
// Parameters:
//   - t: testing.T instance
//   - handler: HTTP handler function to process requests
//
// Returns:
//   - server: httptest.Server instance
//   - getRequests: function to retrieve recorded requests
//   - cleanup: function to close server and clean up
//
// Example:
//
//	server, getRequests, cleanup := startMockServer(t, func(w http.ResponseWriter, r *http.Request) {
//	    w.WriteHeader(http.StatusOK)
//	    _ = json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
//	})
//	defer cleanup()
func startMockServer(t *testing.T, handler http.HandlerFunc) (*httptest.Server, func() []RecordedRequest, func()) {
	t.Helper()

	mockHandler := &MockServerHandler{
		requests: []RecordedRequest{},
		handler:  handler,
	}

	server := httptest.NewServer(mockHandler)

	cleanup := func() {
		server.Close()
	}

	return server, mockHandler.GetRequests, cleanup
}

// runCLI executes the Pipeleak CLI in a separate process with the given arguments
//
// This function captures stdout, stderr, and the exit code by using exec.Command
// with dedicated pipes. This approach is reliable across all platforms (Linux, macOS, Windows).
//
// Parameters:
//   - t: testing.T instance
//   - args: command line arguments (excluding program name)
//   - env: environment variables to set (format: "KEY=VALUE"), can be nil
//   - timeout: maximum execution time before context cancellation
//
// Returns:
//   - stdout: captured standard output as string
//   - stderr: captured standard error as string
//   - exitErr: error returned by command execution (nil on success)
//
// Example:
//
//	stdout, stderr, err := runCLI(t, []string{"gl", "scan", "--token", "xxx"}, nil, 5*time.Second)
func runCLI(t *testing.T, args []string, env []string, timeout time.Duration) (stdout, stderr string, exitErr error) {
	t.Helper()

	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	if !useLiveExecution {
		return "", "", fmt.Errorf("e2e tests in framework mode - enable useLiveExecution")
	}

	// Resolve and prepare the binary
	resolveBinaryPath()
	if err := ensureBinaryBuilt(); err != nil {
		return "", "", fmt.Errorf("failed to build pipeleak binary: %w", err)
	}

	// Create the command with context
	// #nosec G204 - Test helper executing built binary with controlled args in test environment
	cmd := exec.CommandContext(ctx, pipeleakBinaryResolved, args...)
	
	// Set up environment variables
	if len(env) > 0 {
		cmd.Env = append(os.Environ(), env...)
	} else {
		cmd.Env = os.Environ()
	}

	// Create pipes for stdout and stderr
	// This approach works reliably on all platforms including Windows
	var outBuf, errBuf bytes.Buffer
	cmd.Stdout = &outBuf
	cmd.Stderr = &errBuf

	// Run the command
	err := cmd.Run()

	return outBuf.String(), errBuf.String(), err
}// assertLogContains checks if the output contains all expected strings
//
// Parameters:
//   - t: testing.T instance
//   - output: log output to search
//   - expected: slice of strings that must all be present
//
// Example:
//
//	assertLogContains(t, stdout, []string{"Scanning projects", "Found 5 secrets"})
func assertLogContains(t *testing.T, output string, expected []string) {
	t.Helper()
	for _, exp := range expected {
		if !strings.Contains(output, exp) {
			t.Errorf("Expected output to contain %q, but it didn't.\nOutput:\n%s", exp, output)
		}
	}
}

// assertRequestMethodAndPath verifies a request has the expected method and path
func assertRequestMethodAndPath(t *testing.T, req RecordedRequest, method, path string) {
	t.Helper()
	if req.Method != method {
		t.Errorf("Expected method %s, got %s for path %s", method, req.Method, req.Path)
	}
	if req.Path != path {
		t.Errorf("Expected path %s, got %s", path, req.Path)
	}
}

// assertRequestHeader verifies a request has the expected header value
func assertRequestHeader(t *testing.T, req RecordedRequest, header, expected string) {
	t.Helper()
	actual := req.Headers.Get(header)
	if actual != expected {
		t.Errorf("Expected header %s=%q, got %q", header, expected, actual)
	}
}

// dumpRequests prints all recorded requests for debugging
func dumpRequests(t *testing.T, requests []RecordedRequest) {
	t.Helper()
	t.Log("Recorded HTTP requests:")
	for i, req := range requests {
		t.Logf("Request %d:", i+1)
		t.Logf("  Method: %s", req.Method)
		t.Logf("  Path: %s", req.Path)
		if req.RawQuery != "" {
			t.Logf("  Query: %s", req.RawQuery)
		}
		t.Logf("  Headers:")
		for k, v := range req.Headers {
			t.Logf("    %s: %s", k, strings.Join(v, ", "))
		}
		if len(req.Body) > 0 {
			t.Logf("  Body: %s", string(req.Body))
		}
	}
}

// withError returns a handler that always returns an error status
func withError(statusCode int, message string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(statusCode)
		_ = json.NewEncoder(w).Encode(map[string]string{
			"error":   message,
			"message": message,
		})
	}
}

// mockSuccessResponse returns a generic success response
func mockSuccessResponse() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"status":  "success",
			"message": "Operation completed successfully",
		})
	}
}

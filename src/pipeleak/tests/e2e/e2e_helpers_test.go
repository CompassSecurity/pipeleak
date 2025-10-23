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
	"regexp"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
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
//   server, getRequests, cleanup := startMockServer(t, func(w http.ResponseWriter, r *http.Request) {
//       w.WriteHeader(http.StatusOK)
//       _ = json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
//   })
//   defer cleanup()
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

// runCLI executes the Pipeleak CLI in-process with the given arguments
//
// This function captures stdout, stderr, and the exit code by temporarily
// redirecting os.Stdout/os.Stderr and using cobra's Execute() method.
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
//   stdout, stderr, err := runCLI(t, []string{"gl", "scan", "--token", "xxx"}, nil, 5*time.Second)
func runCLI(t *testing.T, args []string, env []string, timeout time.Duration) (stdout, stderr string, exitErr error) {
	t.Helper()

	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	// Set environment variables
	if len(env) > 0 {
		oldEnv := os.Environ()
		defer func() {
			// Restore original environment
			os.Clearenv()
			for _, e := range oldEnv {
				parts := strings.SplitN(e, "=", 2)
				if len(parts) == 2 {
					_ = os.Setenv(parts[0], parts[1])
				}
			}
		}()

		for _, e := range env {
			parts := strings.SplitN(e, "=", 2)
			if len(parts) == 2 {
				_ = os.Setenv(parts[0], parts[1])
			}
		}
	}

	// Capture stdout and stderr
	oldStdout := os.Stdout
	oldStderr := os.Stderr
	rOut, wOut, _ := os.Pipe()
	rErr, wErr, _ := os.Pipe()
	os.Stdout = wOut
	os.Stderr = wErr

	// Buffers to capture output
	var outBuf, errBuf bytes.Buffer

	// Start reading from pipes concurrently to prevent blocking
	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		_, _ = io.Copy(&outBuf, rOut)
	}()
	go func() {
		defer wg.Done()
		_, _ = io.Copy(&errBuf, rErr)
	}()

	// Channel to capture command result
	type result struct {
		err error
	}
	resultChan := make(chan result, 1)

	// Run command in goroutine
	go func() {
		var err error
		if useLiveExecution {
			// Execute the actual CLI command
			err = executeCLI(args)
		} else {
			// Framework demonstration mode - skip execution
			err = fmt.Errorf("e2e tests in framework mode - enable useLiveExecution")
		}
		resultChan <- result{err: err}
	}()

	// Wait for command to complete or timeout
	var cmdErr error
	select {
	case res := <-resultChan:
		cmdErr = res.err
	case <-ctx.Done():
		cmdErr = fmt.Errorf("command timed out after %v", timeout)
	}

	// Close write pipes and restore original stdout/stderr
	_ = wOut.Close()
	_ = wErr.Close()
	os.Stdout = oldStdout
	os.Stderr = oldStderr

	// Wait for all output to be read
	wg.Wait()

	return outBuf.String(), errBuf.String(), cmdErr
}

// assertLogContains checks if the output contains all expected strings
//
// Parameters:
//   - t: testing.T instance
//   - output: log output to search
//   - expected: slice of strings that must all be present
//
// Example:
//   assertLogContains(t, stdout, []string{"Scanning projects", "Found 5 secrets"})
func assertLogContains(t *testing.T, output string, expected []string) {
	t.Helper()
	for _, exp := range expected {
		if !strings.Contains(output, exp) {
			t.Errorf("Expected output to contain %q, but it didn't.\nOutput:\n%s", exp, output)
		}
	}
}

// assertLogNotContains checks if the output does NOT contain specified strings
//nolint:unused
func assertLogNotContains(t *testing.T, output string, forbidden []string) {
	t.Helper()
	for _, forb := range forbidden {
		if strings.Contains(output, forb) {
			t.Errorf("Expected output to NOT contain %q, but it did.\nOutput:\n%s", forb, output)
		}
	}
}

// assertLogMatchesRegex checks if the output matches all provided regex patterns
//
// Parameters:
//   - t: testing.T instance
//   - output: log output to match against
//   - patterns: slice of regex pattern strings
//
// Example:
//   assertLogMatchesRegex(t, stdout, []string{`\d+ secrets found`, `Scan completed in \d+\.\d+s`})
//nolint:unused
func assertLogMatchesRegex(t *testing.T, output string, patterns []string) {
	t.Helper()
	for _, pattern := range patterns {
		matched, err := regexp.MatchString(pattern, output)
		if err != nil {
			t.Fatalf("Invalid regex pattern %q: %v", pattern, err)
		}
		if !matched {
			t.Errorf("Expected output to match pattern %q, but it didn't.\nOutput:\n%s", pattern, output)
		}
	}
}

// compareJSON compares two JSON strings for structural equality
//
// This function unmarshals both strings and uses go-cmp to compare them,
// providing detailed diff output on failure.
//
// Parameters:
//   - t: testing.T instance
//   - got: actual JSON string
//   - want: expected JSON string
//
// Example:
//   compareJSON(t, stdout, `{"status":"success","count":5}`)
//nolint:unused
func compareJSON(t *testing.T, got, want string) {
	t.Helper()

	var gotData, wantData interface{}

	if err := json.Unmarshal([]byte(got), &gotData); err != nil {
		t.Fatalf("Failed to unmarshal 'got' JSON: %v\nJSON:\n%s", err, got)
	}

	if err := json.Unmarshal([]byte(want), &wantData); err != nil {
		t.Fatalf("Failed to unmarshal 'want' JSON: %v\nJSON:\n%s", err, want)
	}

	if diff := cmp.Diff(wantData, gotData); diff != "" {
		t.Errorf("JSON mismatch (-want +got):\n%s", diff)
	}
}

// assertRequestCount verifies the number of HTTP requests received
//nolint:unused
func assertRequestCount(t *testing.T, requests []RecordedRequest, expected int) {
	t.Helper()
	if len(requests) != expected {
		t.Errorf("Expected %d requests, got %d", expected, len(requests))
		for i, req := range requests {
			t.Logf("Request %d: %s %s", i+1, req.Method, req.Path)
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

// assertRequestHeaderContains verifies a request header contains a substring
//nolint:unused
func assertRequestHeaderContains(t *testing.T, req RecordedRequest, header, substring string) {
	t.Helper()
	actual := req.Headers.Get(header)
	if !strings.Contains(actual, substring) {
		t.Errorf("Expected header %s to contain %q, got %q", header, substring, actual)
	}
}

// assertRequestBody verifies the request body matches expected content
//nolint:unused
func assertRequestBody(t *testing.T, req RecordedRequest, expected string) {
	t.Helper()
	actual := string(req.Body)
	if actual != expected {
		t.Errorf("Request body mismatch:\nExpected: %s\nGot: %s", expected, actual)
	}
}

// assertRequestBodyJSON compares request body as JSON
//nolint:unused
//nolint:unused
func assertRequestBodyJSON(t *testing.T, req RecordedRequest, expected string) {
	t.Helper()
	compareJSON(t, string(req.Body), expected)
}

// dumpRequests prints all recorded requests for debugging
//nolint:unused
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

// mockGitLabHandler returns a handler for common GitLab API endpoints
//nolint:unused
func mockGitLabHandler(t *testing.T, responses map[string]interface{}) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Set common headers
		w.Header().Set("Content-Type", "application/json")

		// Route to appropriate response
		key := r.Method + " " + r.URL.Path
		if response, ok := responses[key]; ok {
			if statusCode, ok := response.(int); ok && statusCode >= 400 {
				w.WriteHeader(statusCode)
				_ = json.NewEncoder(w).Encode(map[string]string{
					"error": "API error",
				})
				return
			}
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(response)
			return
		}

		// Default 404 response
		w.WriteHeader(http.StatusNotFound)
		_ = json.NewEncoder(w).Encode(map[string]string{
			"error": "not found",
		})
	}
}

// withTimeout wraps a handler with a delay for testing timeout scenarios
//nolint:unused
func withTimeout(handler http.HandlerFunc, delay time.Duration) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(delay)
		handler(w, r)
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

// createTempConfigFile creates a temporary config file for testing
//nolint:unused
func createTempConfigFile(t *testing.T, content string) string {
	t.Helper()
	tmpDir := t.TempDir()
	configPath := fmt.Sprintf("%s/config.yaml", tmpDir)
	err := os.WriteFile(configPath, []byte(content), 0644)
	if err != nil {
		t.Fatalf("Failed to create temp config file: %v", err)
	}
	return configPath
}

// skipIfShort skips the test if running in short mode
//nolint:unused
func skipIfShort(t *testing.T, reason string) {
	if testing.Short() {
		t.Skipf("Skipping in short mode: %s", reason)
	}
}

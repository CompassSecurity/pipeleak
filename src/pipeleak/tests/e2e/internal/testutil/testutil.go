package testutil

// Shared test utilities for e2e tests.

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
	"path/filepath"
	"runtime"
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

type mockServerHandler struct {
	mu       sync.Mutex
	requests []RecordedRequest
	handler  http.HandlerFunc
}

func (m *mockServerHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Record the request
	bodyBytes, _ := io.ReadAll(r.Body)
	r.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))

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

	m.handler(w, r)
}

// StartMockServer creates a new HTTP test server with request recording
func StartMockServer(t *testing.T, handler http.HandlerFunc) (*httptest.Server, func() []RecordedRequest, func()) {
	t.Helper()
	mh := &mockServerHandler{handler: handler}
	server := httptest.NewServer(mh)
	cleanup := func() { server.Close() }
	get := func() []RecordedRequest {
		mh.mu.Lock()
		defer mh.mu.Unlock()
		return append([]RecordedRequest{}, mh.requests...)
	}
	return server, get, cleanup
}

// AssertLogContains checks if the output contains all expected strings
func AssertLogContains(t *testing.T, output string, expected []string) {
	t.Helper()
	for _, exp := range expected {
		if !strings.Contains(output, exp) {
			t.Errorf("expected output to contain %q, but it didn't.\nOutput:\n%s", exp, output)
		}
	}
}

// RunCLI executes the Pipeleak CLI binary with args, capturing stdout/stderr, with timeout
func RunCLI(t *testing.T, args []string, env []string, timeout time.Duration) (stdout, stderr string, exitErr error) {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	// Apply env overrides
	if len(env) > 0 {
		oldEnv := os.Environ()
		defer func() {
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

	// Capture stdout/stderr via pipes
	oldStdout := os.Stdout
	oldStderr := os.Stderr
	rOut, wOut, _ := os.Pipe()
	rErr, wErr, _ := os.Pipe()
	os.Stdout = wOut
	os.Stderr = wErr

	var outBuf, errBuf bytes.Buffer
	var wg sync.WaitGroup
	wg.Add(2)
	go func() { defer wg.Done(); _, _ = io.Copy(&outBuf, rOut) }()
	go func() { defer wg.Done(); _, _ = io.Copy(&errBuf, rErr) }()

	resCh := make(chan error, 1)
	go func() {
		resCh <- executeCLIWithContext(ctx, args)
	}()

	var err error
	select {
	case err = <-resCh:
	case <-ctx.Done():
		err = fmt.Errorf("command timed out after %v", timeout)
	}

	_ = wOut.Close()
	_ = wErr.Close()
	os.Stdout = oldStdout
	os.Stderr = oldStderr
	wg.Wait()

	return outBuf.String(), errBuf.String(), err
}

// --- Binary execution integration ---

var (
	cliMutex                sync.Mutex
	pipeleakBinary          string
	pipeleakBinaryResolved  string
	resolveOnce, buildOnce  sync.Once
)

func init() {
	pipeleakBinary = os.Getenv("PIPELEAK_BINARY")
	if pipeleakBinary == "" {
		pipeleakBinary = "../../pipeleak"
	}
	if os.Getenv("PIPELEAK_BINARY") == "" {
		if moduleDir, err := findModuleRoot(); err == nil {
			if tmpDir, err := os.MkdirTemp("", "pipeleak-e2e-"); err == nil {
				tmpBin := filepath.Join(tmpDir, "pipeleak")
				if runtime.GOOS == "windows" { tmpBin += ".exe" }
				if err := buildBinary(moduleDir, tmpBin); err == nil { pipeleakBinaryResolved = tmpBin }
			}
		}
	}
}

func resolveBinaryPath() {
	resolveOnce.Do(func() {
		pipeleakBinaryResolved = pipeleakBinary
		if filepath.IsAbs(pipeleakBinary) { return }
		candidates := []string{ pipeleakBinary, filepath.Join("..","..","pipeleak") }
		if runtime.GOOS == "windows" {
			candidates = append(candidates, pipeleakBinary+".exe", filepath.Join("..","..","pipeleak.exe"))
		}
		if os.Getenv("PIPELEAK_BINARY") != "" {
			wd,_ := os.Getwd(); candidates = append(candidates, filepath.Join(wd, pipeleakBinary))
		}
		for _, c := range candidates {
			if _, err := os.Stat(c); err == nil { if abs, err := filepath.Abs(c); err == nil { pipeleakBinaryResolved = abs; return } }
		}
		if abs, err := filepath.Abs(pipeleakBinary); err == nil { pipeleakBinaryResolved = abs }
	})
}

func buildBinary(moduleDir, outputPath string) error {
	cmd := exec.Command("go", "build", "-o", outputPath, ".")
	cmd.Dir = moduleDir
	cmd.Env = os.Environ()
	return cmd.Run()
}

// findModuleRoot searches upwards for a directory containing go.mod and main.go (the CLI entry)
func findModuleRoot() (string, error) {
	wd, err := os.Getwd()
	if err != nil { return "", err }
	for dir := wd; dir != "/" && dir != "."; dir = filepath.Dir(dir) {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			// Prefer a module that has main.go (our CLI root)
			if _, err := os.Stat(filepath.Join(dir, "main.go")); err == nil {
				return dir, nil
			}
			// If no main.go here, see if this is the src/pipeleak module root
			// In our repository layout, tests live under src/pipeleak/tests/e2e
			// so go.mod at src/pipeleak is what we want
			return dir, nil
		}
		if filepath.Dir(dir) == dir { break }
	}
	return "", fmt.Errorf("module root not found from %s", wd)
}

// executeCLIWithContext calls the actual CLI as a separate process so cobra globals don't conflict
func executeCLIWithContext(ctx context.Context, args []string) error {
	resolveBinaryPath()
	cliMutex.Lock(); defer cliMutex.Unlock()

	if os.Getenv("PIPELEAK_BINARY") == "" {
		if pipeleakBinaryResolved != "" { if _, err := os.Stat(pipeleakBinaryResolved); err != nil { pipeleakBinaryResolved = "" } }
		buildOnce.Do(func() {
			tmpDir, err := os.MkdirTemp("", "pipeleak-e2e-"); if err != nil { pipeleakBinaryResolved = ""; return }
			tmpBin := filepath.Join(tmpDir, "pipeleak"); if runtime.GOOS == "windows" { tmpBin += ".exe" }
			moduleDir, err := findModuleRoot(); if err != nil { pipeleakBinaryResolved = ""; return }
			if err := buildBinary(moduleDir, tmpBin); err != nil { pipeleakBinaryResolved = ""; return }
			pipeleakBinaryResolved = tmpBin
		})
		if pipeleakBinaryResolved == "" { return fmt.Errorf("failed to build pipeleak test binary") }
	}
	cmd := exec.CommandContext(ctx, pipeleakBinaryResolved, args...)
	cmd.Env = os.Environ()
	cmd.Stdout = os.Stdout; cmd.Stderr = os.Stderr; cmd.Stdin = os.Stdin
	return cmd.Run()
}

// JSON helpers
func JSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

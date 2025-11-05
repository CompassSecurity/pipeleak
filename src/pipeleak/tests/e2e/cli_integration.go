package e2e

// Package e2e - Integration helper
// This file provides the actual CLI execution integration

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
)

// cliMutex ensures only one CLI execution at a time to avoid cobra race conditions
var cliMutex sync.Mutex

// pipeleakBinary holds the path to the compiled pipeleak binary
var pipeleakBinary string
var pipeleakBinaryResolved string
var resolveOnce sync.Once
var buildOnce sync.Once

func init() {
	// Get the pipeleak binary path from environment or use default
	pipeleakBinary = os.Getenv("PIPELEAK_BINARY")
	if pipeleakBinary == "" {
		pipeleakBinary = "../../pipeleak" // relative to tests/e2e directory
	}

	// Proactively build a fresh test binary once at package init when no explicit binary is provided.
	// This avoids per-test timeouts where short-lived commands (like --help) include build time.
	if os.Getenv("PIPELEAK_BINARY") == "" {
		// Best-effort build; errors will be surfaced on first execution if any.
		// Build from the module root (../../ relative to this test directory)
		if wd, err := os.Getwd(); err == nil {
			moduleDir := filepath.Clean(filepath.Join(wd, "..", ".."))
			if tmpDir, err := os.MkdirTemp("", "pipeleak-e2e-"); err == nil {
				tmpBin := filepath.Join(tmpDir, "pipeleak")
				cmd := exec.Command("/bin/bash", "-lc", fmt.Sprintf("cd %q && go build -o %q .", moduleDir, tmpBin))
				cmd.Env = os.Environ()
				// Do not wire stdout/stderr here to keep test init quiet
				if err := cmd.Run(); err == nil {
					pipeleakBinaryResolved = tmpBin
				}
			}
		}
	}
}

// resolveBinaryPath resolves the binary path once, the first time executeCLIWithContext is called
func resolveBinaryPath() {
	resolveOnce.Do(func() {
		pipeleakBinaryResolved = pipeleakBinary

		// If already absolute, use as-is
		if filepath.IsAbs(pipeleakBinary) {
			return
		}

		// Try to find the binary - check multiple possible locations
		candidates := []string{
			pipeleakBinary,                        // As specified (e.g., "../../pipeleak" or "./pipeleak")
			filepath.Join("..", "..", "pipeleak"), // Relative to tests/e2e
		}

		// If PIPELEAK_BINARY was set, also try it from current working directory
		if os.Getenv("PIPELEAK_BINARY") != "" {
			wd, _ := os.Getwd()
			candidates = append(candidates, filepath.Join(wd, pipeleakBinary))
		}

		// Find the first candidate that exists
		for _, candidate := range candidates {
			if _, err := os.Stat(candidate); err == nil {
				// File exists, convert to absolute path
				if absPath, err := filepath.Abs(candidate); err == nil {
					pipeleakBinaryResolved = absPath
					return
				}
			}
		}

		// If nothing found, try to resolve the original path anyway
		if absPath, err := filepath.Abs(pipeleakBinary); err == nil {
			pipeleakBinaryResolved = absPath
		}
	})
}

// executeCLIWithContext calls the actual CLI command execution via exec.Command with context support
// This avoids cobra global state issues by running the binary as a separate process
func executeCLIWithContext(ctx context.Context, args []string) error {
	// Resolve binary path on first call
	resolveBinaryPath()

	// Serialize CLI execution
	cliMutex.Lock()
	defer cliMutex.Unlock()

	// If no explicit binary provided, build a test binary ONCE per test process to avoid staleness and reduce rebuild overhead
	if os.Getenv("PIPELEAK_BINARY") == "" {
		// If we already have a built binary from init() and it exists, reuse it
		if pipeleakBinaryResolved != "" {
			if _, statErr := os.Stat(pipeleakBinaryResolved); statErr == nil {
				// already built and present
			} else {
				// reset to force rebuild below
				pipeleakBinaryResolved = ""
			}
		}
		buildOnce.Do(func() {
			tmpDir, err := os.MkdirTemp("", "pipeleak-e2e-")
			if err != nil {
				pipeleakBinaryResolved = ""
				return
			}
			tmpBin := filepath.Join(tmpDir, "pipeleak")

			// Build from the module root containing main.go (./ relative to src/pipeleak)
			// We assume tests run from the repo module at src/pipeleak/tests/e2e
			buildDir, err := os.Getwd()
			if err != nil {
				pipeleakBinaryResolved = ""
				return
			}
			// tests run in src/pipeleak/tests/e2e; module with main.go is at ../../
			moduleDir := filepath.Clean(filepath.Join(buildDir, "..", ".."))

			// Use bash -lc to ensure proper PATH resolution for 'go' and allow 'cd' semantics
			buildCmd := exec.Command("/bin/bash", "-lc", fmt.Sprintf("cd %q && go build -o %q .", moduleDir, tmpBin))
			buildCmd.Env = os.Environ()
			buildCmd.Stdout = os.Stdout
			buildCmd.Stderr = os.Stderr
			if err := buildCmd.Run(); err != nil {
				pipeleakBinaryResolved = ""
				return
			}
			pipeleakBinaryResolved = tmpBin
		})

		// If for some reason build failed (pipeleakBinaryResolved empty), return an error
		if pipeleakBinaryResolved == "" {
			return fmt.Errorf("failed to build pipeleak test binary")
		}
	}

	cmd := exec.CommandContext(ctx, pipeleakBinaryResolved, args...)
	cmd.Env = os.Environ()

	// Inherit stdout/stderr from the test process
	// This allows runCLI to capture the output via os.Stdout/os.Stderr redirection
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin

	return cmd.Run()
}

// useLiveExecution controls whether to use real CLI execution
// Set to true to run actual commands in tests
const useLiveExecution = true

package e2e

// Package e2e - Integration helper
// This file provides the actual CLI execution integration

import (
	"context"
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

func init() {
	// Get the pipeleak binary path from environment or use default
	pipeleakBinary = os.Getenv("PIPELEAK_BINARY")
	if pipeleakBinary == "" {
		pipeleakBinary = "../../pipeleak" // relative to tests/e2e directory
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
			pipeleakBinary,                    // As specified (e.g., "../../pipeleak" or "./pipeleak")
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

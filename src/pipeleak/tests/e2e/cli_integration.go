package e2e

// Package e2e - Integration helper
// This file provides the actual CLI execution integration

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
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
				if runtime.GOOS == "windows" {
					tmpBin += ".exe"
				}
				if err := buildBinary(moduleDir, tmpBin); err == nil {
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

		// On Windows, also try with .exe extension
		if runtime.GOOS == "windows" {
			candidates = append(candidates, pipeleakBinary+".exe")
			candidates = append(candidates, filepath.Join("..", "..", "pipeleak.exe"))
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

// buildBinary builds the pipeleak binary in a cross-platform way
func buildBinary(moduleDir, outputPath string) error {
	var cmd *exec.Cmd
	if runtime.GOOS == "windows" {
		// Use Go build directly on Windows
		cmd = exec.Command("go", "build", "-o", outputPath, ".")
	} else {
		// Use Go build directly on Unix-like systems
		cmd = exec.Command("go", "build", "-o", outputPath, ".")
	}
	cmd.Dir = moduleDir
	cmd.Env = os.Environ()
	// Note: stdout/stderr are intentionally not wired to keep init() quiet
	// Errors will be surfaced on first execution if build fails
	return cmd.Run()
}

// ensureBinaryBuilt ensures the pipeleak binary is built and ready for testing
// This function is called by runCLI and should not be called directly
func ensureBinaryBuilt() error {
	// If no explicit binary provided, build a test binary ONCE per test process
	if os.Getenv("PIPELEAK_BINARY") == "" {
		// If we already have a built binary from init() and it exists, reuse it
		if pipeleakBinaryResolved != "" {
			if _, statErr := os.Stat(pipeleakBinaryResolved); statErr == nil {
				// already built and present
				return nil
			}
			// reset to force rebuild below
			pipeleakBinaryResolved = ""
		}

		var buildErr error
		buildOnce.Do(func() {
			tmpDir, err := os.MkdirTemp("", "pipeleak-e2e-")
			if err != nil {
				buildErr = fmt.Errorf("failed to create temp dir: %w", err)
				return
			}
			tmpBin := filepath.Join(tmpDir, "pipeleak")
			if runtime.GOOS == "windows" {
				tmpBin += ".exe"
			}

			// Build from the module root containing main.go (./ relative to src/pipeleak)
			// We assume tests run from the repo module at src/pipeleak/tests/e2e
			buildDir, err := os.Getwd()
			if err != nil {
				buildErr = fmt.Errorf("failed to get working directory: %w", err)
				return
			}
			// tests run in src/pipeleak/tests/e2e; module with main.go is at ../../
			moduleDir := filepath.Clean(filepath.Join(buildDir, "..", ".."))

			if err := buildBinary(moduleDir, tmpBin); err != nil {
				buildErr = fmt.Errorf("failed to build binary: %w", err)
				return
			}
			pipeleakBinaryResolved = tmpBin
		})

		if buildErr != nil {
			return buildErr
		}

		// If for some reason build failed (pipeleakBinaryResolved empty), return an error
		if pipeleakBinaryResolved == "" {
			return fmt.Errorf("failed to build pipeleak test binary")
		}
	}

	return nil
}

// useLiveExecution controls whether to use real CLI execution
// Set to true to run actual commands in tests
const useLiveExecution = true

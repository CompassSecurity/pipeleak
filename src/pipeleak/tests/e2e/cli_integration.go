package e2e

// Package e2e - Integration helper
// This file provides the actual CLI execution integration

import (
	"os"
	"os/exec"
	"sync"
)

// cliMutex ensures only one CLI execution at a time to avoid cobra race conditions
var cliMutex sync.Mutex

// pipeleakBinary holds the path to the compiled pipeleak binary
var pipeleakBinary string

func init() {
	// Build the pipeleak binary once for all tests
	pipeleakBinary = os.Getenv("PIPELEAK_BINARY")
	if pipeleakBinary == "" {
		pipeleakBinary = "../../pipeleak" // relative to tests/e2e directory
	}
}

// executeCLI calls the actual CLI command execution via exec.Command
// This avoids cobra global state issues by running the binary as a separate process
func executeCLI(args []string) error {
	// Serialize CLI execution
	cliMutex.Lock()
	defer cliMutex.Unlock()
	
	cmd := exec.Command(pipeleakBinary, args...)
	cmd.Env = os.Environ()
	
	// The output will be captured by the caller via stdout/stderr redirection
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	
	return cmd.Run()
}

// useLiveExecution controls whether to use real CLI execution
// Set to true to run actual commands in tests
const useLiveExecution = true

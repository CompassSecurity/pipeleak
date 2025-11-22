package main

import (
	"os"

	"github.com/CompassSecurity/pipeleak/cmd"
	"golang.org/x/term"
)

var originalTermState *term.State

func main() {
	saveTerminalState()
	defer restoreTerminalState()

	cmd.TerminalRestorer = restoreTerminalState

	err := cmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func saveTerminalState() {
	if term.IsTerminal(int(os.Stdin.Fd())) {
		state, err := term.GetState(int(os.Stdin.Fd()))
		if err == nil {
			originalTermState = state
		}
	}
}

func restoreTerminalState() {
	if originalTermState != nil {
		_ = term.Restore(int(os.Stdin.Fd()), originalTermState)
	}
}

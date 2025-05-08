package main

import (
	"github.com/CompassSecurity/pipeleak/cmd"
	"golang.org/x/term"
	_ "net/http/pprof"
	"os"
)

var originalTerminalState *term.State

func main() {
	saveTerminalState()
	defer restoreTerminalState()
	err := cmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func saveTerminalState() {
	if term.IsTerminal(int(os.Stdin.Fd())) {
		state, err := term.GetState(int(os.Stdin.Fd()))
		if err == nil {
			originalTerminalState = state
		}
	}
}

func restoreTerminalState() {
	if originalTerminalState != nil {
		_ = term.Restore(int(os.Stdin.Fd()), originalTerminalState)
	}
}

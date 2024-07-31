package main

import (
	"os"

	"github.com/CompassSecurity/pipeleak/cmd"
)

func main() {
	err := cmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

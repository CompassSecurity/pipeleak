package main

import (
	"os"

	"github.com/CompassSecurity/pipeleak/cmd
	_ "net/http/pprof"
)

func main() {
	err := cmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

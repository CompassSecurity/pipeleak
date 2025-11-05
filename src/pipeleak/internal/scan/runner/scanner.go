package runner

import (
	"github.com/CompassSecurity/pipeleak/scanner"
)

type InitOptions struct {
	ConfidenceFilter []string
}

func InitScanner(confidenceFilter []string) {
	scanner.InitRules(confidenceFilter)
}

func InitScannerWithOptions(opts InitOptions) {
	scanner.InitRules(opts.ConfidenceFilter)
}

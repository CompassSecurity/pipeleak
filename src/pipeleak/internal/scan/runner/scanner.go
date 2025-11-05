package runner

import (
	"github.com/CompassSecurity/pipeleak/scanner"
)

// InitOptions contains configuration for scanner initialization
type InitOptions struct {
	ConfidenceFilter []string
}

// InitScanner initializes the scanner with the given confidence filter
// This should be called once at the start of each scan command
func InitScanner(confidenceFilter []string) {
	scanner.InitRules(confidenceFilter)
}

// InitScannerWithOptions initializes the scanner with structured options
func InitScannerWithOptions(opts InitOptions) {
	scanner.InitRules(opts.ConfidenceFilter)
}

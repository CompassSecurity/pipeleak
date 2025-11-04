package processor

import (
	"github.com/CompassSecurity/pipeleak/scanner"
)

// LogProcessingResult contains the findings from processing logs
type LogProcessingResult struct {
	Findings []scanner.Finding
	BuildURL string
}

// ProcessLogContent scans log content for secrets and returns findings
// This is a pure function that separates business logic from I/O operations
func ProcessLogContent(logs []byte, buildURL string, maxGoRoutines int, verifyCredentials bool) (*LogProcessingResult, error) {
	findings, err := scanner.DetectHits(logs, maxGoRoutines, verifyCredentials)
	if err != nil {
		return nil, err
	}

	return &LogProcessingResult{
		Findings: findings,
		BuildURL: buildURL,
	}, nil
}

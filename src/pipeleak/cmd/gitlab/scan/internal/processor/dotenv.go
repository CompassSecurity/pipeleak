package processor

import (
	"github.com/CompassSecurity/pipeleak/scanner"
)

// DotenvResult contains the results of processing dotenv artifacts
type DotenvResult struct {
	Findings []scanner.Finding
	Error    error
}

// ProcessDotenvArtifact scans dotenv file content for secrets
func ProcessDotenvArtifact(dotenvText []byte, jobInfo JobInfo, maxGoRoutines int, verifySecrets bool) *DotenvResult {
	if len(dotenvText) < 1 {
		return &DotenvResult{Findings: []scanner.Finding{}}
	}

	findings, err := scanner.DetectHits(dotenvText, maxGoRoutines, verifySecrets)
	if err != nil {
		return &DotenvResult{Error: err}
	}

	return &DotenvResult{Findings: findings}
}

package processor

import (
	"github.com/CompassSecurity/pipeleak/scanner"
)

// JobInfo contains metadata about a GitLab CI job
type JobInfo struct {
	ProjectID int
	JobID     int
	JobWebURL string
	JobName   string
}

// TraceResult contains the results of processing a job trace
type TraceResult struct {
	Findings []scanner.Finding
	Error    error
}

// ProcessJobTrace scans job trace content for secrets
func ProcessJobTrace(trace []byte, jobInfo JobInfo, maxGoRoutines int, verifySecrets bool) *TraceResult {
	if len(trace) < 1 {
		return &TraceResult{Findings: []scanner.Finding{}}
	}

	findings, err := scanner.DetectHits(trace, maxGoRoutines, verifySecrets)
	if err != nil {
		return &TraceResult{Error: err}
	}

	return &TraceResult{Findings: findings}
}

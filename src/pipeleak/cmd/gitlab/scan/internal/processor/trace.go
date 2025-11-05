package processor

import (
	"github.com/CompassSecurity/pipeleak/scanner"
)

type JobInfo struct {
	ProjectID int
	JobID     int
	JobWebURL string
	JobName   string
}

type TraceResult struct {
	Findings []scanner.Finding
	Error    error
}

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

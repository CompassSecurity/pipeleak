package processor

import (
	"github.com/CompassSecurity/pipeleak/pkg/scanner"
)

type LogProcessingResult struct {
	Findings []scanner.Finding
	BuildURL string
}

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

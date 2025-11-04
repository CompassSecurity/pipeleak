package processor

import (
	"github.com/CompassSecurity/pipeleak/scanner"
	"github.com/h2non/filetype"
)

// StepLogResult contains the findings from processing pipeline step logs
type StepLogResult struct {
	WorkspaceSlug string
	RepoSlug      string
	PipelineUUID  string
	StepUUID      string
	Findings      []scanner.Finding
	Error         error
}

// ProcessStepLogs processes pipeline step logs and returns findings
// This is a pure function that separates business logic from I/O operations
func ProcessStepLogs(logBytes []byte, workspaceSlug, repoSlug, pipelineUUID, stepUUID string, maxGoRoutines int, verifyCredentials bool) (*StepLogResult, error) {
	result := &StepLogResult{
		WorkspaceSlug: workspaceSlug,
		RepoSlug:      repoSlug,
		PipelineUUID:  pipelineUUID,
		StepUUID:      stepUUID,
	}

	findings, err := scanner.DetectHits(logBytes, maxGoRoutines, verifyCredentials)
	if err != nil {
		result.Error = err
		return result, err
	}

	result.Findings = findings
	return result, nil
}

// ArtifactResult contains the findings from processing an artifact
type ArtifactResult struct {
	Filename      string
	DownloadURL   string
	WebURL        string
	IsArchive     bool
	ProcessedFile bool
	Error         error
}

// ProcessArtifactContent processes artifact content (archive or file) for scanning
// Separates the artifact processing logic from API calls for testability
func ProcessArtifactContent(fileBytes []byte, filename, webURL string, verifyCredentials bool) *ArtifactResult {
	result := &ArtifactResult{
		Filename: filename,
		WebURL:   webURL,
	}

	if len(fileBytes) == 0 {
		return result
	}

	if filetype.IsArchive(fileBytes) {
		result.IsArchive = true
		// Note: scanner functions have side effects (logging)
		scanner.HandleArchiveArtifact(filename, fileBytes, webURL, "Download Artifact", verifyCredentials)
		result.ProcessedFile = true
	} else {
		result.IsArchive = false
		scanner.DetectFileHits(fileBytes, webURL, "Download Artifact", filename, "", verifyCredentials)
		result.ProcessedFile = true
	}

	return result
}

// ShouldContinueScanning determines if scanning should continue based on limits
// Pure function for flow control logic
func ShouldContinueScanning(currentCount, maxLimit int) bool {
	if maxLimit <= 0 {
		return true // No limit set
	}
	return currentCount < maxLimit
}

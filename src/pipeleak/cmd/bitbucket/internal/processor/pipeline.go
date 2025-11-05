package processor

import (
	"github.com/CompassSecurity/pipeleak/pkg/scanner"
	"github.com/h2non/filetype"
)

type StepLogResult struct {
	WorkspaceSlug string
	RepoSlug      string
	PipelineUUID  string
	StepUUID      string
	Findings      []scanner.Finding
	Error         error
}

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

type ArtifactResult struct {
	Filename      string
	DownloadURL   string
	WebURL        string
	IsArchive     bool
	ProcessedFile bool
	Error         error
}

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
		scanner.HandleArchiveArtifact(filename, fileBytes, webURL, "Download Artifact", verifyCredentials)
		result.ProcessedFile = true
	} else {
		result.IsArchive = false
		scanner.DetectFileHits(fileBytes, webURL, "Download Artifact", filename, "", verifyCredentials)
		result.ProcessedFile = true
	}

	return result
}

func ShouldContinueScanning(currentCount, maxLimit int) bool {
	if maxLimit <= 0 {
		return true
	}
	return currentCount < maxLimit
}

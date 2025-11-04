package processor

import (
	"archive/zip"
	"bytes"
	"io"

	"github.com/CompassSecurity/pipeleak/scanner"
)

// WorkflowLogResult contains the findings from processing workflow logs
type WorkflowLogResult struct {
	WorkflowURL string
	Findings    []scanner.Finding
	Error       error
}

// ProcessWorkflowLogs processes workflow run logs and returns findings
// Pure function that separates business logic from I/O operations
func ProcessWorkflowLogs(logs []byte, workflowURL string, maxGoRoutines int, verifyCredentials bool) (*WorkflowLogResult, error) {
	result := &WorkflowLogResult{
		WorkflowURL: workflowURL,
	}

	findings, err := scanner.DetectHits(logs, maxGoRoutines, verifyCredentials)
	if err != nil {
		result.Error = err
		return result, err
	}

	result.Findings = findings
	return result, nil
}

// ZipLogResult contains extracted log content from a zip file
type ZipLogResult struct {
	TotalBytes  int
	FileCount   int
	ExtractedLogs []byte
	Errors      []error
}

// ExtractLogsFromZip extracts all log files from a zip archive
// Pure function for zip extraction logic
func ExtractLogsFromZip(zipBytes []byte) (*ZipLogResult, error) {
	result := &ZipLogResult{
		ExtractedLogs: make([]byte, 0),
		Errors:       make([]error, 0),
	}

	zipReader, err := zip.NewReader(bytes.NewReader(zipBytes), int64(len(zipBytes)))
	if err != nil {
		return nil, err
	}

	result.FileCount = len(zipReader.File)

	for _, zipFile := range zipReader.File {
		unzippedBytes, err := readZipFile(zipFile)
		if err != nil {
			result.Errors = append(result.Errors, err)
			continue
		}

		result.ExtractedLogs = append(result.ExtractedLogs, unzippedBytes...)
	}

	result.TotalBytes = len(result.ExtractedLogs)
	return result, nil
}

// readZipFile reads a single file from a zip archive
func readZipFile(zf *zip.File) ([]byte, error) {
	f, err := zf.Open()
	if err != nil {
		return nil, err
	}
	defer func() { _ = f.Close() }()
	return io.ReadAll(f)
}

// WorkflowRunFilter determines which workflow runs should be scanned
type WorkflowRunFilter struct {
	MaxWorkflows int
	CurrentCount int
}

// ShouldContinueScanning determines if more workflow runs should be scanned
// Pure function for flow control logic
func (f *WorkflowRunFilter) ShouldContinueScanning() bool {
	if f.MaxWorkflows <= 0 {
		return true // No limit set
	}
	return f.CurrentCount < f.MaxWorkflows
}

// IncrementCount increments the workflow count
func (f *WorkflowRunFilter) IncrementCount() {
	f.CurrentCount++
}

// ReachedLimit checks if the limit has been reached
func (f *WorkflowRunFilter) ReachedLimit() bool {
	if f.MaxWorkflows <= 0 {
		return false
	}
	return f.CurrentCount >= f.MaxWorkflows
}

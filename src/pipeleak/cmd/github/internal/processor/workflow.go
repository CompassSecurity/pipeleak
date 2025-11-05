package processor

import (
	"archive/zip"
	"bytes"
	"io"

	"github.com/CompassSecurity/pipeleak/pkg/scanner"
)

type WorkflowLogResult struct {
	WorkflowURL string
	Findings    []scanner.Finding
	Error       error
}

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

type ZipLogResult struct {
	TotalBytes    int
	FileCount     int
	ExtractedLogs []byte
	Errors        []error
}

func ExtractLogsFromZip(zipBytes []byte) (*ZipLogResult, error) {
	result := &ZipLogResult{
		ExtractedLogs: make([]byte, 0),
		Errors:        make([]error, 0),
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

func readZipFile(zf *zip.File) ([]byte, error) {
	f, err := zf.Open()
	if err != nil {
		return nil, err
	}
	defer func() { _ = f.Close() }()
	return io.ReadAll(f)
}

type WorkflowRunFilter struct {
	MaxWorkflows int
	CurrentCount int
}

func (f *WorkflowRunFilter) ShouldContinueScanning() bool {
	if f.MaxWorkflows <= 0 {
		return true
	}
	return f.CurrentCount < f.MaxWorkflows
}

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

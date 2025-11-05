package logline

import (
	"archive/zip"
	"bytes"
	"io"

	"github.com/CompassSecurity/pipeleak/scanner"
	"github.com/rs/zerolog/log"
)

// ProcessOptions contains configuration for log processing
type ProcessOptions struct {
	MaxGoRoutines     int
	VerifyCredentials bool
	BuildURL          string
	JobName           string
}

// LogProcessingResult contains the result of processing logs
type LogProcessingResult struct {
	Findings  []scanner.Finding
	BytesRead int
	Error     error
}

// ProcessLogs processes log content and returns findings
// This is the common function used by all scan commands for log scanning
func ProcessLogs(logs []byte, opts ProcessOptions) (*LogProcessingResult, error) {
	result := &LogProcessingResult{
		BytesRead: len(logs),
	}

	findings, err := scanner.DetectHits(logs, opts.MaxGoRoutines, opts.VerifyCredentials)
	if err != nil {
		result.Error = err
		return result, err
	}

	result.Findings = findings
	return result, nil
}

// ZipLogResult contains extracted log content from a zip file
type ZipLogResult struct {
	TotalBytes    int
	FileCount     int
	ExtractedLogs []byte
	Errors        []error
}

// ExtractLogsFromZip extracts all log files from a zip archive
// This is used by commands that receive logs as zip files
func ExtractLogsFromZip(zipBytes []byte) (*ZipLogResult, error) {
	result := &ZipLogResult{
		ExtractedLogs: make([]byte, 0),
		Errors:        make([]error, 0),
	}

	if len(zipBytes) == 0 {
		return result, nil
	}

	zipReader, err := zip.NewReader(bytes.NewReader(zipBytes), int64(len(zipBytes)))
	if err != nil {
		return nil, err
	}

	result.FileCount = len(zipReader.File)

	for _, zipFile := range zipReader.File {
		log.Trace().Str("zipFile", zipFile.Name).Msg("Extracting zip file")
		unzippedBytes, err := readZipFile(zipFile)
		if err != nil {
			log.Err(err).Str("file", zipFile.Name).Msg("Failed reading zip file")
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

// ProcessLogsFromZip combines zip extraction and log processing
// This is a convenience function for common use cases
func ProcessLogsFromZip(zipBytes []byte, opts ProcessOptions) (*LogProcessingResult, error) {
	// Extract logs from zip
	zipResult, err := ExtractLogsFromZip(zipBytes)
	if err != nil {
		return nil, err
	}

	// Process the extracted logs
	return ProcessLogs(zipResult.ExtractedLogs, opts)
}

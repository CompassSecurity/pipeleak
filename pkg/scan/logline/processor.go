package logline

import (
	"archive/zip"
	"bytes"
	"io"
	"time"

	"github.com/CompassSecurity/pipeleek/pkg/scanner"
	"github.com/rs/zerolog/log"
)

type ProcessOptions struct {
	MaxGoRoutines     int
	VerifyCredentials bool
	BuildURL          string
	JobName           string
	HitTimeout        time.Duration
}

type LogProcessingResult struct {
	Findings  []scanner.Finding
	BytesRead int
	Error     error
}

func ProcessLogs(logs []byte, opts ProcessOptions) (*LogProcessingResult, error) {
	result := &LogProcessingResult{
		BytesRead: len(logs),
	}

	findings, err := scanner.DetectHits(logs, opts.MaxGoRoutines, opts.VerifyCredentials, opts.HitTimeout)
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

func readZipFile(zf *zip.File) ([]byte, error) {
	f, err := zf.Open()
	if err != nil {
		return nil, err
	}
	defer func() { _ = f.Close() }()
	return io.ReadAll(f)
}

func ProcessLogsFromZip(zipBytes []byte, opts ProcessOptions) (*LogProcessingResult, error) {
	zipResult, err := ExtractLogsFromZip(zipBytes)
	if err != nil {
		return nil, err
	}

	return ProcessLogs(zipResult.ExtractedLogs, opts)
}

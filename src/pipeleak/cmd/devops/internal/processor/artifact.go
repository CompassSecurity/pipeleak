package processor

import (
	"archive/zip"
	"bytes"
	"context"
	"io"

	"github.com/CompassSecurity/pipeleak/scanner"
	"github.com/h2non/filetype"
	"github.com/wandb/parallel"
)

// ArtifactProcessingResult contains findings from artifact processing
type ArtifactProcessingResult struct {
	ArtifactName string
	BuildURL     string
	FileResults  []FileProcessingResult
	Error        error
}

// FileProcessingResult contains findings from a single file in an artifact
type FileProcessingResult struct {
	FileName string
	FileType string
	Findings []scanner.Finding
	Error    error
}

// ProcessArtifactZip processes a zip artifact and scans its contents for secrets
// This extracts the zip processing logic for testability
func ProcessArtifactZip(zipBytes []byte, artifactName string, buildURL string, maxGoRoutines int, verifyCredentials bool) (*ArtifactProcessingResult, error) {
	zipListing, err := zip.NewReader(bytes.NewReader(zipBytes), int64(len(zipBytes)))
	if err != nil {
		return nil, err
	}

	result := &ArtifactProcessingResult{
		ArtifactName: artifactName,
		BuildURL:     buildURL,
		FileResults:  make([]FileProcessingResult, 0),
	}

	ctx := context.Background()
	group := parallel.Limited(ctx, maxGoRoutines)
	resultsChan := make(chan FileProcessingResult, len(zipListing.File))

	for _, file := range zipListing.File {
		fileCopy := file
		group.Go(func(ctx context.Context) {
			fileResult := processZipFile(fileCopy, artifactName, buildURL, verifyCredentials)
			resultsChan <- fileResult
		})
	}

	group.Wait()
	close(resultsChan)

	for fileResult := range resultsChan {
		result.FileResults = append(result.FileResults, fileResult)
	}

	return result, nil
}

// processZipFile processes a single file from a zip archive
func processZipFile(file *zip.File, artifactName, buildURL string, verifyCredentials bool) FileProcessingResult {
	result := FileProcessingResult{
		FileName: file.Name,
	}

	fc, err := file.Open()
	if err != nil {
		result.Error = err
		return result
	}
	defer func() { _ = fc.Close() }()

	content, err := io.ReadAll(fc)
	if err != nil {
		result.Error = err
		return result
	}

	kind, _ := filetype.Match(content)
	result.FileType = kind.MIME.Value

	// Scan unknown file types (likely text files)
	// Note: scanner functions have side effects (logging), but we track what we process
	if kind == filetype.Unknown {
		scanner.DetectFileHits(content, buildURL, artifactName, file.Name, "", verifyCredentials)
		// Mark that we processed this file
		result.Findings = []scanner.Finding{} // Actual findings are logged by scanner
	} else if filetype.IsArchive(content) {
		// Handle nested archives
		scanner.HandleArchiveArtifact(file.Name, content, buildURL, artifactName, verifyCredentials)
		result.Findings = []scanner.Finding{} // Actual findings are logged by scanner
	}

	return result
}

// ProcessFileContent processes a single file's content for scanning
// Separate function for easier testing of file processing logic
func ProcessFileContent(content []byte, filename, artifactName, buildURL string, verifyCredentials bool) (*FileProcessingResult, error) {
	result := &FileProcessingResult{
		FileName: filename,
	}

	kind, _ := filetype.Match(content)
	result.FileType = kind.MIME.Value

	if kind == filetype.Unknown {
		scanner.DetectFileHits(content, buildURL, artifactName, filename, "", verifyCredentials)
		result.Findings = []scanner.Finding{} // Actual findings are logged by scanner
	} else if filetype.IsArchive(content) {
		scanner.HandleArchiveArtifact(filename, content, buildURL, artifactName, verifyCredentials)
		result.Findings = []scanner.Finding{} // Actual findings are logged by scanner
	}

	return result, nil
}

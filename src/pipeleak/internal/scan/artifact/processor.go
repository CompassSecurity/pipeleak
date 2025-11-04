package artifact

import (
	"archive/zip"
	"bytes"
	"context"
	"io"

	"github.com/CompassSecurity/pipeleak/scanner"
	"github.com/h2non/filetype"
	"github.com/rs/zerolog/log"
	"github.com/wandb/parallel"
)

// ProcessOptions contains configuration for artifact processing
type ProcessOptions struct {
	MaxGoRoutines      int
	VerifyCredentials  bool
	BuildURL           string
	ArtifactName       string
	WorkflowRunName    string
}

// FileProcessingResult contains the result of processing a single file
type FileProcessingResult struct {
	FileName  string
	FileType  string
	IsArchive bool
	IsUnknown bool
	Error     error
}

// ProcessZipArtifact processes a zip artifact and scans its contents for secrets
// This is the common function used by all scan commands
func ProcessZipArtifact(zipBytes []byte, opts ProcessOptions) ([]FileProcessingResult, error) {
	if len(zipBytes) == 0 {
		return []FileProcessingResult{}, nil
	}

	zipListing, err := zip.NewReader(bytes.NewReader(zipBytes), int64(len(zipBytes)))
	if err != nil {
		return nil, err
	}

	results := make([]FileProcessingResult, 0)
	resultsChan := make(chan FileProcessingResult, len(zipListing.File))

	ctx := context.Background()
	group := parallel.Limited(ctx, opts.MaxGoRoutines)

	for _, file := range zipListing.File {
		fileCopy := file
		group.Go(func(ctx context.Context) {
			result := processZipFile(fileCopy, opts)
			resultsChan <- result
		})
	}

	group.Wait()
	close(resultsChan)

	for result := range resultsChan {
		results = append(results, result)
	}

	return results, nil
}

// processZipFile processes a single file from a zip archive
func processZipFile(file *zip.File, opts ProcessOptions) FileProcessingResult {
	result := FileProcessingResult{
		FileName: file.Name,
	}

	fc, err := file.Open()
	if err != nil {
		log.Error().Stack().Err(err).Msg("Unable to open raw artifact zip file")
		result.Error = err
		return result
	}
	defer func() { _ = fc.Close() }()

	content, err := io.ReadAll(fc)
	if err != nil {
		log.Error().Stack().Err(err).Msg("Unable to readAll artifact zip file")
		result.Error = err
		return result
	}

	kind, _ := filetype.Match(content)
	result.FileType = kind.MIME.Value
	result.IsUnknown = kind == filetype.Unknown
	result.IsArchive = filetype.IsArchive(content)

	// Scan unknown file types (likely text files)
	if result.IsUnknown {
		scanner.DetectFileHits(content, opts.BuildURL, opts.ArtifactName, file.Name, "", opts.VerifyCredentials)
	} else if result.IsArchive {
		// Handle nested archives
		scanner.HandleArchiveArtifact(file.Name, content, opts.BuildURL, opts.ArtifactName, opts.VerifyCredentials)
	}

	return result
}

// ExtractZipFile extracts the contents of a single file from a zip entry
// This is a utility function for reading zip file contents
func ExtractZipFile(zf *zip.File) ([]byte, error) {
	f, err := zf.Open()
	if err != nil {
		return nil, err
	}
	defer func() { _ = f.Close() }()
	return io.ReadAll(f)
}

// DetermineFileType determines the type of file based on its content
// Returns the file type and whether it's an archive or unknown type
func DetermineFileType(content []byte) (mimeType string, isArchive bool, isUnknown bool) {
	kind, _ := filetype.Match(content)
	return kind.MIME.Value, filetype.IsArchive(content), kind == filetype.Unknown
}

// ProcessSingleFile processes a single file's content for scanning
// This is useful for processing non-zip artifacts or individual files
func ProcessSingleFile(content []byte, filename string, opts ProcessOptions) (*FileProcessingResult, error) {
	result := &FileProcessingResult{
		FileName: filename,
	}

	mimeType, isArchive, isUnknown := DetermineFileType(content)
	result.FileType = mimeType
	result.IsArchive = isArchive
	result.IsUnknown = isUnknown

	if isUnknown {
		scanner.DetectFileHits(content, opts.BuildURL, opts.ArtifactName, filename, "", opts.VerifyCredentials)
	} else if isArchive {
		scanner.HandleArchiveArtifact(filename, content, opts.BuildURL, opts.ArtifactName, opts.VerifyCredentials)
	}

	return result, nil
}

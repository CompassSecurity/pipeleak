package artifact

import (
	"archive/zip"
	"bytes"
	"context"
	"io"
	"time"

	"github.com/CompassSecurity/pipeleak/pkg/scanner"
	"github.com/h2non/filetype"
	"github.com/rs/zerolog/log"
	"github.com/wandb/parallel"
)

type ProcessOptions struct {
	MaxGoRoutines     int
	VerifyCredentials bool
	BuildURL          string
	ArtifactName      string
	WorkflowRunName   string
	HitTimeout        time.Duration
}

type FileProcessingResult struct {
	FileName  string
	FileType  string
	IsArchive bool
	IsUnknown bool
	Error     error
}

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

	if result.IsUnknown {
		scanner.DetectFileHits(content, opts.BuildURL, opts.ArtifactName, file.Name, "", opts.VerifyCredentials, opts.HitTimeout)
	} else if result.IsArchive {
		scanner.HandleArchiveArtifact(file.Name, content, opts.BuildURL, opts.ArtifactName, opts.VerifyCredentials, opts.HitTimeout)
	}

	return result
}

func ExtractZipFile(zf *zip.File) ([]byte, error) {
	f, err := zf.Open()
	if err != nil {
		return nil, err
	}
	defer func() { _ = f.Close() }()
	return io.ReadAll(f)
}

func DetermineFileType(content []byte) (mimeType string, isArchive bool, isUnknown bool) {
	kind, _ := filetype.Match(content)
	return kind.MIME.Value, filetype.IsArchive(content), kind == filetype.Unknown
}

func ProcessSingleFile(content []byte, filename string, opts ProcessOptions) (*FileProcessingResult, error) {
	result := &FileProcessingResult{
		FileName: filename,
	}

	mimeType, isArchive, isUnknown := DetermineFileType(content)
	result.FileType = mimeType
	result.IsArchive = isArchive
	result.IsUnknown = isUnknown

	if isUnknown {
		scanner.DetectFileHits(content, opts.BuildURL, opts.ArtifactName, filename, "", opts.VerifyCredentials, opts.HitTimeout)
	} else if isArchive {
		scanner.HandleArchiveArtifact(filename, content, opts.BuildURL, opts.ArtifactName, opts.VerifyCredentials, opts.HitTimeout)
	}

	return result, nil
}

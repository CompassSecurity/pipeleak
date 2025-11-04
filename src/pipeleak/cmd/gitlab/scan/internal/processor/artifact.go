package processor

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

// ArtifactZipResult contains information about processing artifact zips
type ArtifactZipResult struct {
	FilesProcessed int
	FilesSkipped   int
	Error          error
}

// ArtifactFileInfo contains metadata for each file in the artifact zip
type ArtifactFileInfo struct {
	Name      string
	Content   []byte
	IsArchive bool
	IsUnknown bool
}

// ProcessJobArtifactZip processes a zip archive of job artifacts
// Returns structured information instead of directly calling scanner
func ProcessJobArtifactZip(data []byte, maxGoRoutines int) (*ArtifactZipResult, error) {
	if len(data) == 0 {
		return &ArtifactZipResult{}, nil
	}

	reader := bytes.NewReader(data)
	zipListing, err := zip.NewReader(reader, int64(len(data)))
	if err != nil {
		return &ArtifactZipResult{Error: err}, err
	}

	result := &ArtifactZipResult{}
	ctx := context.Background()
	group := parallel.Limited(ctx, maxGoRoutines)

	for _, file := range zipListing.File {
		result.FilesProcessed++
		group.Go(func(ctx context.Context) {
			fc, err := file.Open()
			if err != nil {
				log.Error().Stack().Err(err).Msg("Unable to open raw artifact zip file")
				return
			}
			defer func() { _ = fc.Close() }()

			content, err := io.ReadAll(fc)
			if err != nil {
				log.Error().Stack().Err(err).Msg("Unable to readAll artifact zip file")
				return
			}

			ProcessArtifactFile(file.Name, content)
		})
	}

	group.Wait()
	return result, nil
}

// ProcessArtifactFile determines file type and calls appropriate scanner
func ProcessArtifactFile(fileName string, content []byte) {
	kind, _ := filetype.Match(content)

	// Unknown file types (text files, etc.) - scan directly
	if kind == filetype.Unknown {
		scanner.DetectFileHits(content, "", "", fileName, "", false)
	} else if filetype.IsArchive(content) {
		// Nested archives
		scanner.HandleArchiveArtifact(fileName, content, "", "", false)
	}
	// Other known binary types are skipped
}

// ExtractArtifactFiles extracts all files from a zip for testing/inspection
// This is a pure function for testing purposes
func ExtractArtifactFiles(data []byte) ([]ArtifactFileInfo, error) {
	if len(data) == 0 {
		return []ArtifactFileInfo{}, nil
	}

	reader := bytes.NewReader(data)
	zipListing, err := zip.NewReader(reader, int64(len(data)))
	if err != nil {
		return nil, err
	}

	var files []ArtifactFileInfo
	for _, file := range zipListing.File {
		fc, err := file.Open()
		if err != nil {
			continue
		}

		content, err := io.ReadAll(fc)
		_ = fc.Close()
		if err != nil {
			continue
		}

		kind, _ := filetype.Match(content)
		files = append(files, ArtifactFileInfo{
			Name:      file.Name,
			Content:   content,
			IsArchive: filetype.IsArchive(content),
			IsUnknown: kind == filetype.Unknown,
		})
	}

	return files, nil
}

package format

import (
	"archive/zip"
	"bytes"

	gounits "github.com/docker/go-units"
	"github.com/rs/zerolog/log"
)

// CalculateZipFileSize returns the aggregated uncompressed size of files inside a zip archive
func CalculateZipFileSize(data []byte) uint64 {
	reader := bytes.NewReader(data)
	zipListing, err := zip.NewReader(reader, int64(len(data)))
	if err != nil {
		log.Error().Msg("Failed calculcatingZipFileSize")
		return 0
	}
	totalSize := uint64(0)
	for _, file := range zipListing.File {
		totalSize = totalSize + file.UncompressedSize64
	}

	return totalSize
}

// ParseHumanSize parses a human-readable size string (e.g., "500Mb", "2Gb") into bytes
func ParseHumanSize(size string) (int64, error) {
	return gounits.FromHumanSize(size)
}

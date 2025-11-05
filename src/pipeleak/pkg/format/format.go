package format

import (
    "archive/zip"
    "bytes"
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

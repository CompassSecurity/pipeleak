package archive

import (
"archive/zip"
"bytes"
)

func CalculateZipFileSize(data []byte) uint64 {
zipListing, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
if err != nil {
return 0
}

var totalSize uint64
for _, file := range zipListing.File {
totalSize += file.UncompressedSize64
}

return totalSize
}

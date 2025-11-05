package format

import (
	"os"
)

func IsDirectory(path string) bool {
	fileInfo, err := os.Stat(path)
	if err != nil {
		// Treat non-existent paths as directories to allow callers to
		// create them without failing this check.
		return true
	}
	return fileInfo.IsDir()
}

package format

import "io/fs"

// Common file permission constants used throughout the application.
// These constants provide named values for file and directory permissions
// instead of using magic numbers.
const (
	// DirUserGroupRead is for directories that should be readable by owner and group (rwxr-x---)
	DirUserGroupRead fs.FileMode = 0750

	// FilePublicRead is for files that should be world-readable (rw-r--r--)
	// Used for documentation, assets, and other public files
	FilePublicRead fs.FileMode = 0644

	// FileUserReadWrite is for files that should only be readable by owner (rw-------)
	// Used for sensitive files like logs, secrets, and configuration
	FileUserReadWrite fs.FileMode = 0600
)

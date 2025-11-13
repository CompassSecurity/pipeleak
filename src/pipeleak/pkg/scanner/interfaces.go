package scanner

// BaseScanner defines the minimal interface that all platform-specific scanners must implement.
// This provides a common contract for scanning operations across different CI/CD platforms.
type BaseScanner interface {
// Scan performs a scan based on the configured options and returns any error encountered.
Scan() error
}

// ScannerWithStatus extends BaseScanner with methods for monitoring scan progress.
// Implement this interface for scanners that need to report their status.
type ScannerWithStatus interface {
BaseScanner

// GetStatus returns a human-readable status string describing the current scan state.
GetStatus() string
}

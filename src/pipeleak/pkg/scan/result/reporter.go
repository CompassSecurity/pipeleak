package result

import (
	"github.com/CompassSecurity/pipeleak/pkg/logging"
	"github.com/CompassSecurity/pipeleak/pkg/scanner"
)

// SecretType re-exports logging.SecretType for backward compatibility.
type SecretType = logging.SecretType

const (
	// SecretTypeLog indicates a secret found in CI/CD logs.
	SecretTypeLog = logging.SecretTypeLog
	// SecretTypeArchive indicates a secret found in an archive/artifact.
	SecretTypeArchive = logging.SecretTypeArchive
	// SecretTypeArchiveInArchive indicates a secret found in a nested archive.
	SecretTypeArchiveInArchive = logging.SecretTypeArchiveInArchive
	// SecretTypeDotenv indicates a secret found in a dotenv file.
	SecretTypeDotenv = logging.SecretTypeDotenv
	// SecretTypeFile indicates a secret found in a standalone file.
	SecretTypeFile = logging.SecretTypeFile
)

type ReportOptions struct {
	LocationURL string
	JobName     string
	BuildName   string
	Type        SecretType
}

func ReportFindings(findings []scanner.Finding, opts ReportOptions) {
	for _, finding := range findings {
		ReportFinding(finding, opts)
	}
}

func ReportFinding(finding scanner.Finding, opts ReportOptions) {
	secretType := opts.Type
	if secretType == "" {
		secretType = SecretTypeLog
	}

	event := logging.Hit().
		Str("type", string(secretType)).
		Str("confidence", finding.Pattern.Pattern.Confidence).
		Str("ruleName", finding.Pattern.Pattern.Name).
		Str("value", finding.Text)

	// Add location information if provided
	if opts.LocationURL != "" {
		event = event.Str("url", opts.LocationURL)
	}
	if opts.JobName != "" {
		event = event.Str("job", opts.JobName)
	}
	if opts.BuildName != "" {
		event = event.Str("build", opts.BuildName)
	}

	event.Msg("SECRET")
}

func ReportFindingWithCustomFields(finding scanner.Finding, customFields map[string]string) {
	// Extract type from custom fields if present, default to LOG
	secretType := SecretTypeLog
	if t, ok := customFields["type"]; ok {
		secretType = SecretType(t)
		delete(customFields, "type")
	}

	event := logging.Hit().
		Str("type", string(secretType)).
		Str("confidence", finding.Pattern.Pattern.Confidence).
		Str("ruleName", finding.Pattern.Pattern.Name).
		Str("value", finding.Text)

	for key, value := range customFields {
		event = event.Str(key, value)
	}

	event.Msg("SECRET")
}

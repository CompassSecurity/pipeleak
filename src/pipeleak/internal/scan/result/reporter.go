package result

import (
	"github.com/CompassSecurity/pipeleak/scanner"
	"github.com/rs/zerolog/log"
)

// ReportOptions contains options for reporting findings
type ReportOptions struct {
	LocationURL string
	JobName     string
	BuildName   string
}

// ReportFindings reports all findings using structured logging
// This standardizes the output format across all scan commands
func ReportFindings(findings []scanner.Finding, opts ReportOptions) {
	for _, finding := range findings {
		ReportFinding(finding, opts)
	}
}

// ReportFinding reports a single finding using structured logging
// All scan commands use this consistent format
func ReportFinding(finding scanner.Finding, opts ReportOptions) {
	event := log.Warn().
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

	event.Msg("HIT")
}

// ReportFindingWithCustomFields reports a finding with custom fields
// This allows commands to add platform-specific information
func ReportFindingWithCustomFields(finding scanner.Finding, customFields map[string]string) {
	event := log.Warn().
		Str("confidence", finding.Pattern.Pattern.Confidence).
		Str("ruleName", finding.Pattern.Pattern.Name).
		Str("value", finding.Text)

	for key, value := range customFields {
		event = event.Str(key, value)
	}

	event.Msg("HIT")
}

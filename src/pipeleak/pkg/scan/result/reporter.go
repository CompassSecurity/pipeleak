package result

import (
	"github.com/CompassSecurity/pipeleak/pkg/scanner"
	"github.com/rs/zerolog/log"
)

type ReportOptions struct {
	LocationURL string
	JobName     string
	BuildName   string
}

func ReportFindings(findings []scanner.Finding, opts ReportOptions) {
	for _, finding := range findings {
		ReportFinding(finding, opts)
	}
}

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

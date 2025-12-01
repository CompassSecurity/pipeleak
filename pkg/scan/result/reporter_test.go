package result

import (
	"bytes"
	"testing"

	"github.com/CompassSecurity/pipeleak/pkg/logging"
	"github.com/CompassSecurity/pipeleak/pkg/scanner"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/stretchr/testify/assert"
)

func TestReportFinding(t *testing.T) {
	tests := []struct {
		name           string
		finding        scanner.Finding
		opts           ReportOptions
		expectInLog    []string
		notExpectInLog []string
	}{
		{
			name: "finding with full options",
			finding: scanner.Finding{
				Pattern: scanner.PatternElement{
					Pattern: scanner.PatternPattern{
						Name:       "AWS Access Key",
						Regex:      "AKIA[0-9A-Z]{16}",
						Confidence: "high",
					},
				},
				Text: "AKIAIOSFODNN7EXAMPLE",
			},
			opts: ReportOptions{
				LocationURL: "https://example.com/build/123",
				JobName:     "test-job",
				BuildName:   "build-456",
				Type:        logging.SecretTypeLog,
			},
			expectInLog: []string{
				"high",
				"AWS Access Key",
				"AKIAIOSFODNN7EXAMPLE",
				"https://example.com/build/123",
				"test-job",
				"build-456",
				"SECRET",
				"log",
			},
		},
		{
			name: "finding with minimal options",
			finding: scanner.Finding{
				Pattern: scanner.PatternElement{
					Pattern: scanner.PatternPattern{
						Name:       "Generic Secret",
						Regex:      "secret.*",
						Confidence: "medium",
					},
				},
				Text: "secret_value_123",
			},
			opts: ReportOptions{
				LocationURL: "https://example.com/workflow/789",
			},
			expectInLog: []string{
				"medium",
				"Generic Secret",
				"secret_value_123",
				"https://example.com/workflow/789",
				"SECRET",
				"log", // default type
			},
			notExpectInLog: []string{
				"job",
				"build",
			},
		},
		{
			name: "finding with no options",
			finding: scanner.Finding{
				Pattern: scanner.PatternElement{
					Pattern: scanner.PatternPattern{
						Name:       "Test Pattern",
						Regex:      "test.*",
						Confidence: "low",
					},
				},
				Text: "test_secret",
			},
			opts: ReportOptions{},
			expectInLog: []string{
				"low",
				"Test Pattern",
				"test_secret",
				"SECRET",
				"log", // default type
			},
		},
		{
			name: "finding with archive type",
			finding: scanner.Finding{
				Pattern: scanner.PatternElement{
					Pattern: scanner.PatternPattern{
						Name:       "API Key",
						Regex:      "api_key_.*",
						Confidence: "high",
					},
				},
				Text: "api_key_12345",
			},
			opts: ReportOptions{
				LocationURL: "https://example.com/artifacts/123",
				Type:        logging.SecretTypeArchive,
			},
			expectInLog: []string{
				"high",
				"API Key",
				"api_key_12345",
				"SECRET",
				"archive",
			},
		},
		{
			name: "finding with dotenv type",
			finding: scanner.Finding{
				Pattern: scanner.PatternElement{
					Pattern: scanner.PatternPattern{
						Name:       "Database Password",
						Regex:      "DB_PASSWORD=.*",
						Confidence: "high",
					},
				},
				Text: "DB_PASSWORD=secret123",
			},
			opts: ReportOptions{
				LocationURL: "https://example.com/job/456",
				Type:        logging.SecretTypeDotenv,
			},
			expectInLog: []string{
				"high",
				"Database Password",
				"DB_PASSWORD=secret123",
				"SECRET",
				"dotenv",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Capture log output for verification with HitLevelWriter
			var buf bytes.Buffer
			hitWriter := logging.NewHitLevelWriter(&buf)
			log.Logger = zerolog.New(hitWriter)
			logging.SetGlobalHitWriter(hitWriter)

			ReportFinding(tt.finding, tt.opts)

			output := buf.String()
			for _, expected := range tt.expectInLog {
				assert.Contains(t, output, expected, "Expected to find %q in log output", expected)
			}
			for _, notExpected := range tt.notExpectInLog {
				assert.NotContains(t, output, notExpected, "Did not expect to find %q in log output", notExpected)
			}
		})
	}
}

func TestReportFindings(t *testing.T) {
	var buf bytes.Buffer
	hitWriter := logging.NewHitLevelWriter(&buf)
	log.Logger = zerolog.New(hitWriter)
	logging.SetGlobalHitWriter(hitWriter)

	findings := []scanner.Finding{
		{
			Pattern: scanner.PatternElement{
				Pattern: scanner.PatternPattern{
					Name:       "Pattern 1",
					Regex:      "pattern1",
					Confidence: "high",
				},
			},
			Text: "secret1",
		},
		{
			Pattern: scanner.PatternElement{
				Pattern: scanner.PatternPattern{
					Name:       "Pattern 2",
					Regex:      "pattern2",
					Confidence: "medium",
				},
			},
			Text: "secret2",
		},
	}

	opts := ReportOptions{
		LocationURL: "https://example.com/test",
	}

	buf.Reset()
	ReportFindings(findings, opts)

	output := buf.String()
	assert.Contains(t, output, "Pattern 1")
	assert.Contains(t, output, "secret1")
	assert.Contains(t, output, "Pattern 2")
	assert.Contains(t, output, "secret2")
	assert.Contains(t, output, "high")
	assert.Contains(t, output, "medium")
}

func TestReportFindingWithCustomFields(t *testing.T) {
	var buf bytes.Buffer
	hitWriter := logging.NewHitLevelWriter(&buf)
	log.Logger = zerolog.New(hitWriter)
	logging.SetGlobalHitWriter(hitWriter)

	finding := scanner.Finding{
		Pattern: scanner.PatternElement{
			Pattern: scanner.PatternPattern{
				Name:       "Custom Pattern",
				Regex:      "custom.*",
				Confidence: "high",
			},
		},
		Text: "custom_secret",
	}

	customFields := map[string]string{
		"type":       string(logging.SecretTypeLog),
		"workspace":  "my-workspace",
		"repository": "my-repo",
		"pipeline":   "pipeline-123",
	}

	buf.Reset()
	ReportFindingWithCustomFields(finding, customFields)

	output := buf.String()
	assert.Contains(t, output, "Custom Pattern")
	assert.Contains(t, output, "custom_secret")
	assert.Contains(t, output, "high")
	assert.Contains(t, output, "my-workspace")
	assert.Contains(t, output, "my-repo")
	assert.Contains(t, output, "pipeline-123")
	assert.Contains(t, output, "SECRET")
	assert.Contains(t, output, "log")
}

func TestReportFindings_EmptyList(t *testing.T) {
	var buf bytes.Buffer
	hitWriter := logging.NewHitLevelWriter(&buf)
	log.Logger = zerolog.New(hitWriter)
	logging.SetGlobalHitWriter(hitWriter)

	findings := []scanner.Finding{}
	opts := ReportOptions{
		LocationURL: "https://example.com/test",
	}

	buf.Reset()
	ReportFindings(findings, opts)

	// Should not log anything for empty findings
	output := buf.String()
	assert.Empty(t, output)
}

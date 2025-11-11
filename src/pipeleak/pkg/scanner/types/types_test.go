package types

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPatternPattern(t *testing.T) {
	tests := []struct {
		name       string
		pattern    PatternPattern
		expectName string
		expectConf string
	}{
		{
			name: "AWS access key pattern",
			pattern: PatternPattern{
				Name:       "AWS Access Key",
				Regex:      "AKIA[0-9A-Z]{16}",
				Confidence: "high",
			},
			expectName: "AWS Access Key",
			expectConf: "high",
		},
		{
			name: "generic secret pattern",
			pattern: PatternPattern{
				Name:       "Generic Secret",
				Regex:      "secret.*",
				Confidence: "medium",
			},
			expectName: "Generic Secret",
			expectConf: "medium",
		},
		{
			name: "empty pattern",
			pattern: PatternPattern{
				Name:       "",
				Regex:      "",
				Confidence: "",
			},
			expectName: "",
			expectConf: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expectName, tt.pattern.Name)
			assert.Equal(t, tt.expectConf, tt.pattern.Confidence)
			assert.NotNil(t, tt.pattern.Regex)
		})
	}
}

func TestPatternElement(t *testing.T) {
	element := PatternElement{
		Pattern: PatternPattern{
			Name:       "Test Pattern",
			Regex:      "test.*",
			Confidence: "high",
		},
	}

	assert.Equal(t, "Test Pattern", element.Pattern.Name)
	assert.Equal(t, "test.*", element.Pattern.Regex)
	assert.Equal(t, "high", element.Pattern.Confidence)
}

func TestSecretsPatterns(t *testing.T) {
	tests := []struct {
		name        string
		patterns    SecretsPatterns
		expectCount int
		expectEmpty bool
	}{
		{
			name: "multiple patterns",
			patterns: SecretsPatterns{
				Patterns: []PatternElement{
					{Pattern: PatternPattern{Name: "Pattern1", Regex: "regex1", Confidence: "high"}},
					{Pattern: PatternPattern{Name: "Pattern2", Regex: "regex2", Confidence: "medium"}},
					{Pattern: PatternPattern{Name: "Pattern3", Regex: "regex3", Confidence: "low"}},
				},
			},
			expectCount: 3,
			expectEmpty: false,
		},
		{
			name: "single pattern",
			patterns: SecretsPatterns{
				Patterns: []PatternElement{
					{Pattern: PatternPattern{Name: "OnlyOne", Regex: "regex", Confidence: "high"}},
				},
			},
			expectCount: 1,
			expectEmpty: false,
		},
		{
			name: "empty patterns",
			patterns: SecretsPatterns{
				Patterns: []PatternElement{},
			},
			expectCount: 0,
			expectEmpty: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Len(t, tt.patterns.Patterns, tt.expectCount)
			if tt.expectEmpty {
				assert.Empty(t, tt.patterns.Patterns)
			} else {
				assert.NotEmpty(t, tt.patterns.Patterns)
			}
		})
	}
}

func TestFinding(t *testing.T) {
	tests := []struct {
		name       string
		finding    Finding
		expectText string
		expectName string
	}{
		{
			name: "AWS key finding",
			finding: Finding{
				Pattern: PatternElement{
					Pattern: PatternPattern{
						Name:       "AWS Access Key",
						Regex:      "AKIA[0-9A-Z]{16}",
						Confidence: "high",
					},
				},
				Text: "AKIAIOSFODNN7EXAMPLE",
			},
			expectText: "AKIAIOSFODNN7EXAMPLE",
			expectName: "AWS Access Key",
		},
		{
			name: "generic finding",
			finding: Finding{
				Pattern: PatternElement{
					Pattern: PatternPattern{
						Name:       "Generic",
						Regex:      ".*",
						Confidence: "low",
					},
				},
				Text: "some text",
			},
			expectText: "some text",
			expectName: "Generic",
		},
		{
			name: "empty finding",
			finding: Finding{
				Pattern: PatternElement{
					Pattern: PatternPattern{},
				},
				Text: "",
			},
			expectText: "",
			expectName: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expectText, tt.finding.Text)
			assert.Equal(t, tt.expectName, tt.finding.Pattern.Pattern.Name)
		})
	}
}

func TestDetectionResult(t *testing.T) {
	tests := []struct {
		name           string
		result         DetectionResult
		expectError    bool
		expectFindings int
	}{
		{
			name: "successful detection with findings",
			result: DetectionResult{
				Findings: []Finding{
					{
						Pattern: PatternElement{
							Pattern: PatternPattern{Name: "Secret1", Regex: "regex1", Confidence: "high"},
						},
						Text: "secret1",
					},
					{
						Pattern: PatternElement{
							Pattern: PatternPattern{Name: "Secret2", Regex: "regex2", Confidence: "high"},
						},
						Text: "secret2",
					},
				},
				Error: nil,
			},
			expectError:    false,
			expectFindings: 2,
		},
		{
			name: "detection with no findings",
			result: DetectionResult{
				Findings: []Finding{},
				Error:    nil,
			},
			expectError:    false,
			expectFindings: 0,
		},
		{
			name: "detection with error",
			result: DetectionResult{
				Findings: nil,
				Error:    assert.AnError,
			},
			expectError:    true,
			expectFindings: 0,
		},
		{
			name: "detection with findings and error",
			result: DetectionResult{
				Findings: []Finding{
					{Text: "partial result"},
				},
				Error: assert.AnError,
			},
			expectError:    true,
			expectFindings: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.expectError {
				assert.Error(t, tt.result.Error)
			} else {
				assert.NoError(t, tt.result.Error)
			}
			assert.Len(t, tt.result.Findings, tt.expectFindings)
		})
	}
}

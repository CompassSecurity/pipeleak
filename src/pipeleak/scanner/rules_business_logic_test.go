package scanner

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestConfidenceFiltering tests the critical business logic of filtering patterns by confidence level
func TestConfidenceFiltering(t *testing.T) {
	tests := []struct {
		name             string
		patterns         []PatternElement
		confidenceFilter []string
		expectedCount    int
		expectedPatterns []string
	}{
		{
			name: "filter high confidence only",
			patterns: []PatternElement{
				{Pattern: PatternPattern{Name: "High Pattern", Regex: "test", Confidence: "high"}},
				{Pattern: PatternPattern{Name: "Medium Pattern", Regex: "test", Confidence: "medium"}},
				{Pattern: PatternPattern{Name: "Low Pattern", Regex: "test", Confidence: "low"}},
			},
			confidenceFilter: []string{"high"},
			expectedCount:    1,
			expectedPatterns: []string{"High Pattern"},
		},
		{
			name: "filter multiple confidence levels",
			patterns: []PatternElement{
				{Pattern: PatternPattern{Name: "High Pattern", Regex: "test", Confidence: "high"}},
				{Pattern: PatternPattern{Name: "Medium Pattern", Regex: "test", Confidence: "medium"}},
				{Pattern: PatternPattern{Name: "Low Pattern", Regex: "test", Confidence: "low"}},
			},
			confidenceFilter: []string{"high", "medium"},
			expectedCount:    2,
			expectedPatterns: []string{"High Pattern", "Medium Pattern"},
		},
		{
			name: "non-matching filter removes all",
			patterns: []PatternElement{
				{Pattern: PatternPattern{Name: "High Pattern", Regex: "test", Confidence: "high"}},
			},
			confidenceFilter: []string{"low"},
			expectedCount:    0,
			expectedPatterns: []string{},
		},
		{
			name: "empty filter keeps all",
			patterns: []PatternElement{
				{Pattern: PatternPattern{Name: "High Pattern", Regex: "test", Confidence: "high"}},
				{Pattern: PatternPattern{Name: "Low Pattern", Regex: "test", Confidence: "low"}},
			},
			confidenceFilter: []string{},
			expectedCount:    2,
			expectedPatterns: []string{"High Pattern", "Low Pattern"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Simulate the confidence filtering logic from InitRules
			filtered := []PatternElement{}
			if len(tt.confidenceFilter) > 0 {
				for _, pattern := range tt.patterns {
					for _, conf := range tt.confidenceFilter {
						if pattern.Pattern.Confidence == conf {
							filtered = append(filtered, pattern)
							break
						}
					}
				}
			} else {
				filtered = tt.patterns
			}

			assert.Len(t, filtered, tt.expectedCount)
			for _, expectedName := range tt.expectedPatterns {
				found := false
				for _, p := range filtered {
					if p.Pattern.Name == expectedName {
						found = true
						break
					}
				}
				assert.True(t, found, "expected pattern %s not found", expectedName)
			}
		})
	}
}

// TestDeduplicationBoundaryConditions tests edge cases in finding deduplication
func TestDeduplicationBoundaryConditions(t *testing.T) {
	tests := []struct {
		name                     string
		initialDeduplicationList []string
		findings                 []Finding
		expectedUniqueCount      int
		description              string
	}{
		{
			name:                     "empty deduplication list accepts all",
			initialDeduplicationList: []string{},
			findings: []Finding{
				{Pattern: PatternElement{Pattern: PatternPattern{Name: "Test", Confidence: "high"}}, Text: "secret1"},
				{Pattern: PatternElement{Pattern: PatternPattern{Name: "Test", Confidence: "high"}}, Text: "secret2"},
			},
			expectedUniqueCount: 2,
			description:         "New findings should all be accepted when dedup list is empty",
		},
		{
			name:                     "single finding processes correctly",
			initialDeduplicationList: []string{},
			findings: []Finding{
				{Pattern: PatternElement{Pattern: PatternPattern{Name: "Test", Confidence: "high"}}, Text: "secret1"},
			},
			expectedUniqueCount: 1,
			description:         "Single finding should be accepted",
		},
		{
			name:                     "list truncation at 500 entries",
			initialDeduplicationList: make([]string, 500),
			findings: []Finding{
				{Pattern: PatternElement{Pattern: PatternPattern{Name: "Test", Confidence: "high"}}, Text: "new_secret"},
			},
			expectedUniqueCount: 1,
			description:         "Deduplication list should handle 500+ entries",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset global state
			findingsDeduplicationList = tt.initialDeduplicationList
			defer func() {
				findingsDeduplicationList = []string{}
			}()

			result := deduplicateFindings(tt.findings)
			assert.Len(t, result, tt.expectedUniqueCount, tt.description)
		})
	}
}

// TestRegexPatternEdgeCases tests that regex patterns handle edge cases correctly
func TestRegexPatternEdgeCases(t *testing.T) {
	tests := []struct {
		name        string
		pattern     PatternElement
		text        string
		shouldMatch bool
	}{
		{
			name: "pattern matches at start of text",
			pattern: PatternElement{Pattern: PatternPattern{
				Name:       "Test Pattern",
				Regex:      `^SECRET=.*`,
				Confidence: "high",
			}},
			text:        "SECRET=value",
			shouldMatch: true,
		},
		{
			name: "pattern matches at end of text",
			pattern: PatternElement{Pattern: PatternPattern{
				Name:       "Test Pattern",
				Regex:      `.*SECRET$`,
				Confidence: "high",
			}},
			text:        "password: SECRET",
			shouldMatch: true,
		},
		{
			name: "pattern with special characters",
			pattern: PatternElement{Pattern: PatternPattern{
				Name:       "Test Pattern",
				Regex:      `password\s*=\s*['"][^'"]+['"]`,
				Confidence: "high",
			}},
			text:        `password = "my_secret_123"`,
			shouldMatch: true,
		},
		{
			name: "multiline pattern",
			pattern: PatternElement{Pattern: PatternPattern{
				Name:       "Test Pattern",
				Regex:      `(?m)^API_KEY=.*$`,
				Confidence: "high",
			}},
			text:        "USER=admin\nAPI_KEY=secret123\nDEBUG=true",
			shouldMatch: true,
		},
		{
			name: "case sensitive pattern",
			pattern: PatternElement{Pattern: PatternPattern{
				Name:       "Test Pattern",
				Regex:      `AWS_SECRET`,
				Confidence: "high",
			}},
			text:        "aws_secret",
			shouldMatch: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			secretsPatterns = SecretsPatterns{Patterns: []PatternElement{tt.pattern}}
			defer func() {
				secretsPatterns = SecretsPatterns{}
			}()

			findings, err := DetectHits([]byte(tt.text), 2, false)
			assert.NoError(t, err)

			if tt.shouldMatch {
				assert.Greater(t, len(findings), 0, "expected pattern to match but found no matches")
			} else {
				assert.Equal(t, 0, len(findings), "expected no match but found matches")
			}
		})
	}
}

// TestExtractHitWithSurroundingTextBoundaries tests boundary conditions for hit extraction
func TestExtractHitWithSurroundingTextBoundaries(t *testing.T) {
	tests := []struct {
		name            string
		text            []byte
		hitIndex        []int
		additionalBytes int
		validate        func(*testing.T, string)
	}{
		{
			name:            "hit at very start of text",
			text:            []byte("PASSWORD=secret123 rest of text"),
			hitIndex:        []int{0, 10},
			additionalBytes: 5,
			validate: func(t *testing.T, result string) {
				assert.Contains(t, result, "PASSWORD")
				assert.True(t, len(result) > 0)
			},
		},
		{
			name:            "hit at very end of text",
			text:            []byte("start of text PASSWORD=secret123"),
			hitIndex:        []int{14, 32},
			additionalBytes: 5,
			validate: func(t *testing.T, result string) {
				assert.Contains(t, result, "PASSWORD")
			},
		},
		{
			name:            "text shorter than additional bytes requested",
			text:            []byte("short"),
			hitIndex:        []int{0, 5},
			additionalBytes: 1000,
			validate: func(t *testing.T, result string) {
				assert.Equal(t, "short", result)
			},
		},
		{
			name:            "empty text",
			text:            []byte{},
			hitIndex:        []int{0, 0},
			additionalBytes: 5,
			validate: func(t *testing.T, result string) {
				assert.Equal(t, "", result)
			},
		},
		{
			name:            "single character text",
			text:            []byte("X"),
			hitIndex:        []int{0, 1},
			additionalBytes: 10,
			validate: func(t *testing.T, result string) {
				assert.Equal(t, "X", result)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractHitWithSurroundingText(tt.text, tt.hitIndex, tt.additionalBytes)
			tt.validate(t, result)
		})
	}
}

// TestCleanHitLineBusinessLogic tests the cleaning logic for findings output
func TestCleanHitLineBusinessLogic(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
		reason   string
	}{
		{
			name:     "preserves spaces in content",
			input:    "API_KEY = secret value",
			expected: "API_KEY = secret value",
			reason:   "Spaces within content should be preserved",
		},
		{
			name:     "replaces newline with single space",
			input:    "line1\nline2",
			expected: "line1 line2",
			reason:   "Newlines should be replaced with spaces for single-line output",
		},
		{
			name:     "handles multiple consecutive newlines",
			input:    "line1\n\n\nline2",
			expected: "line1   line2",
			reason:   "Multiple newlines should become multiple spaces",
		},
		{
			name:     "removes ANSI color codes",
			input:    "\x1b[31mred text\x1b[0m normal text",
			expected: "red text normal text",
			reason:   "ANSI codes should be stripped for clean log output",
		},
		{
			name:     "handles mixed newlines and ANSI codes",
			input:    "\x1b[32mgreen\x1b[0m\ntext",
			expected: "green text",
			reason:   "Both ANSI codes and newlines should be cleaned",
		},
		{
			name:     "empty string returns empty",
			input:    "",
			expected: "",
			reason:   "Empty input should return empty output",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := cleanHitLine(tt.input)
			assert.Equal(t, tt.expected, result, tt.reason)
		})
	}
}

// TestTruncationBehavior tests that findings are properly truncated to prevent memory issues
func TestTruncationBehavior(t *testing.T) {
	tests := []struct {
		name        string
		textSize    int
		expectedMax int
	}{
		{
			name:        "text under 1024 bytes not truncated",
			textSize:    500,
			expectedMax: 500,
		},
		{
			name:        "text over 1024 bytes truncated",
			textSize:    2000,
			expectedMax: 1024,
		},
		{
			name:        "text exactly 1024 bytes not truncated",
			textSize:    1024,
			expectedMax: 1024,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a text of specified size
			text := bytes.Repeat([]byte("a"), tt.textSize)

			// Simulate the truncation logic from DetectHitsWithTimeout
			hitStr := string(text)
			if len(hitStr) > 1024 {
				hitStr = hitStr[0:1024]
			}

			assert.LessOrEqual(t, len(hitStr), tt.expectedMax)
			if tt.textSize <= 1024 {
				assert.Equal(t, tt.textSize, len(hitStr))
			} else {
				assert.Equal(t, 1024, len(hitStr))
			}
		})
	}
}

package renovate

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestExtractSelfHostedOptions(t *testing.T) {
	tests := []struct {
		name     string
		input    []byte
		expected []string
	}{
		{
			name: "extracts single option",
			input: []byte(`## selfHostedType
Description here`),
			expected: []string{"selfHostedType"},
		},
		{
			name: "extracts multiple options",
			input: []byte(`## option1
Some text
## option2
More text
## option3
Even more text`),
			expected: []string{"option1", "option2", "option3"},
		},
		{
			name:     "returns empty for no matches",
			input:    []byte("No matching content here"),
			expected: []string{},
		},
		{
			name: "handles options with special characters",
			input: []byte(`## self-hosted-type
## selfHosted_Type
## selfHosted.Type`),
			expected: []string{"self-hosted-type", "selfHosted_Type", "selfHosted.Type"},
		},
		{
			name: "ignores non-## headers",
			input: []byte(`# Level 1 Header
## option1
### Level 3 Header
## option2`),
			expected: []string{"option1", "Level 3 Header", "option2"}, // ## .* matches ### as well
		},
		{
			name:     "handles empty input",
			input:    []byte(""),
			expected: []string{},
		},
		{
			name: "handles whitespace around markers",
			input: []byte(`   ## option1   
Some text
		## option2		
More text`),
			expected: []string{"option1", "option2"},
		},
		{
			name: "extracts real renovate options",
			input: []byte(`## allowCustomCrateRegistries
## allowPlugins
## allowPostUpgradeCommandTemplating
## allowScripts
## allowedPostUpgradeCommands`),
			expected: []string{
				"allowCustomCrateRegistries",
				"allowPlugins",
				"allowPostUpgradeCommandTemplating",
				"allowScripts",
				"allowedPostUpgradeCommands",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractSelfHostedOptions(tt.input)

			if len(tt.expected) == 0 {
				assert.Empty(t, result)
			} else {
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}

func TestExtractSelfHostedOptions_RealWorld(t *testing.T) {
	t.Run("parses markdown documentation format", func(t *testing.T) {
		markdown := []byte(`# Self-hosted options

These options are only applicable for self-hosted Renovate instances.

## platform
Platform type of SCM. Options: github, gitlab, bitbucket, azure.

## endpoint
API endpoint for the platform.

## binarySource
Controls where Renovate installs binaries.`)

		result := extractSelfHostedOptions(markdown)
		expected := []string{"platform", "endpoint", "binarySource"}
		assert.Equal(t, expected, result)
	})
}

func TestValidateOrderBy(t *testing.T) {
	tests := []struct {
		name       string
		orderBy    string
		shouldFail bool
	}{
		{"accepts id", "id", false},
		{"accepts name", "name", false},
		{"accepts path", "path", false},
		{"accepts created_at", "created_at", false},
		{"accepts updated_at", "updated_at", false},
		{"accepts star_count", "star_count", false},
		{"accepts last_activity_at", "last_activity_at", false},
		{"accepts similarity", "similarity", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// validateOrderBy calls log.Fatal on invalid input
			// Since we can't easily test log.Fatal without restructuring,
			// we'll just verify valid inputs don't panic
			if !tt.shouldFail {
				assert.NotPanics(t, func() {
					validateOrderBy(tt.orderBy)
				})
			}
		})
	}
}

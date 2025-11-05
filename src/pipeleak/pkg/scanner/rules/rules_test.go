package rules

import (
	"os"
	"testing"

	"github.com/CompassSecurity/pipeleak/pkg/scanner/types"
)

func TestAppendPipeleakRules(t *testing.T) {
	tests := []struct {
		name          string
		inputRules    []types.PatternElement
		expectedCount int
	}{
		{
			name:          "empty rules",
			inputRules:    []types.PatternElement{},
			expectedCount: 1,
		},
		{
			name: "with existing rules",
			inputRules: []types.PatternElement{
				{Pattern: types.PatternPattern{Name: "Test Rule", Regex: "test", Confidence: "high"}},
			},
			expectedCount: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := AppendPipeleakRules(tt.inputRules)
			if len(result) != tt.expectedCount {
				t.Errorf("Expected %d rules, got %d", tt.expectedCount, len(result))
			}

			found := false
			for _, rule := range result {
				if rule.Pattern.Name == "Gitlab - Predefined Environment Variable" {
					found = true
					if rule.Pattern.Confidence != "medium" {
						t.Errorf("Expected confidence 'medium', got %q", rule.Pattern.Confidence)
					}
					break
				}
			}
			if !found {
				t.Error("Custom GitLab rule not found in appended rules")
			}
		})
	}
}

func TestGetSecretsPatterns(t *testing.T) {
	patterns := GetSecretsPatterns()
	t.Logf("Patterns count: %d", len(patterns.Patterns))
}

func TestGetTruffleHogRules(t *testing.T) {
	rules := GetTruffleHogRules()
	t.Logf("TruffleHog rules count: %d", len(rules))
}

func TestDownloadRules(t *testing.T) {
	tmpDir := t.TempDir()
	origWd, _ := os.Getwd()
	defer os.Chdir(origWd)

	os.Chdir(tmpDir)

	t.Run("rules file does not exist", func(t *testing.T) {
		if _, err := os.Stat(ruleFileName); err == nil {
			os.Remove(ruleFileName)
		}

		DownloadRules()

		if _, err := os.Stat(ruleFileName); os.IsNotExist(err) {
			t.Error("Expected rules file to be downloaded")
		}
	})

	t.Run("rules file already exists", func(t *testing.T) {
		if _, err := os.Stat(ruleFileName); os.IsNotExist(err) {
			os.WriteFile(ruleFileName, []byte("dummy"), 0644)
		}

		DownloadRules()

		if _, err := os.Stat(ruleFileName); os.IsNotExist(err) {
			t.Error("Expected rules file to exist")
		}
	})
}

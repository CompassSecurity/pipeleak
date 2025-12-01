package format

import (
	"strings"
	"testing"
)

func TestPrettyPrintYAML(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		wantErr   bool
		checkFunc func(string) bool
	}{
		{
			name:    "simple key-value",
			input:   "key: value",
			wantErr: false,
			checkFunc: func(output string) bool {
				return strings.Contains(output, "key:") && strings.Contains(output, "value")
			},
		},
		{
			name:    "nested structure",
			input:   "parent:\n  child: value",
			wantErr: false,
			checkFunc: func(output string) bool {
				return strings.Contains(output, "parent:") && strings.Contains(output, "child:")
			},
		},
		{
			name:    "array",
			input:   "items:\n  - one\n  - two\n  - three",
			wantErr: false,
			checkFunc: func(output string) bool {
				return strings.Contains(output, "items:") && strings.Contains(output, "- one")
			},
		},
		{
			name:    "invalid YAML",
			input:   "key: [unclosed",
			wantErr: true,
			checkFunc: func(output string) bool {
				return output == ""
			},
		},
		{
			name:    "empty input",
			input:   "",
			wantErr: false,
			checkFunc: func(output string) bool {
				return true
			},
		},
		{
			name:    "multiline string",
			input:   "description: |\n  line1\n  line2",
			wantErr: false,
			checkFunc: func(output string) bool {
				return strings.Contains(output, "description:")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := PrettyPrintYAML(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("PrettyPrintYAML() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.checkFunc(result) {
				t.Errorf("PrettyPrintYAML() output validation failed for input: %q, got: %q", tt.input, result)
			}
		})
	}
}

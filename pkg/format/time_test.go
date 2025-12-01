package format

import (
	"testing"
	"time"
)

func TestParseISO8601(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		shouldErr bool
	}{
		{
			name:      "valid RFC3339 format",
			input:     "2023-01-15T10:30:00Z",
			shouldErr: false,
		},
		{
			name:      "valid RFC3339 with timezone",
			input:     "2023-01-15T10:30:00+01:00",
			shouldErr: false,
		},
		{
			name:      "valid RFC3339 with milliseconds",
			input:     "2023-01-15T10:30:00.123Z",
			shouldErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if !tt.shouldErr {
				result := ParseISO8601(tt.input)
				expected, _ := time.Parse(time.RFC3339, tt.input)
				if !result.Equal(expected) {
					t.Errorf("ParseISO8601(%q) = %v, want %v", tt.input, result, expected)
				}
			}
		})
	}
}

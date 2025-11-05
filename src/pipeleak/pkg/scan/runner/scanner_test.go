package runner

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestInitScanner(t *testing.T) {
	tests := []struct {
		name             string
		confidenceFilter []string
	}{
		{
			name:             "no filter",
			confidenceFilter: []string{},
		},
		{
			name:             "with high confidence filter",
			confidenceFilter: []string{"high"},
		},
		{
			name:             "with multiple filters",
			confidenceFilter: []string{"high", "medium"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Should not panic
			assert.NotPanics(t, func() {
				InitScanner(tt.confidenceFilter)
			})
		})
	}
}

func TestInitScannerWithOptions(t *testing.T) {
	tests := []struct {
		name string
		opts InitOptions
	}{
		{
			name: "empty options",
			opts: InitOptions{
				ConfidenceFilter: []string{},
			},
		},
		{
			name: "with confidence filter",
			opts: InitOptions{
				ConfidenceFilter: []string{"high", "medium"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Should not panic
			assert.NotPanics(t, func() {
				InitScannerWithOptions(tt.opts)
			})
		})
	}
}

package nist

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFetchVulns(t *testing.T) {
	t.Run("FetchVulns does not panic", func(t *testing.T) {
		// Note: This function makes actual HTTP call to NIST NVD API
		// We'll just verify function signature exists and doesn't panic on invalid input
		assert.NotPanics(t, func() {
			_, _ = FetchVulns("0.0.0")
		})
	})
}

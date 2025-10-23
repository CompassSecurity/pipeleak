package scan

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewScanCmd(t *testing.T) {
	t.Run("creates scan command", func(t *testing.T) {
		cmd := NewScanCmd()
		assert.NotNil(t, cmd)
		assert.Equal(t, "scan", cmd.Use)
		assert.Equal(t, "Scan a GitLab instance", cmd.Short)
	})

	t.Run("has required flags", func(t *testing.T) {
		cmd := NewScanCmd()
		gitlabFlag := cmd.Flags().Lookup("gitlab")
		assert.NotNil(t, gitlabFlag)
		
tokenFlag := cmd.Flags().Lookup("token")
		assert.NotNil(t, tokenFlag)
	})

	t.Run("parseFileSize parses sizes", func(t *testing.T) {
		v := parseFileSize("1Kb")
		assert.True(t, v > 0)
	})
}

package gitlab

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewShodanCmd(t *testing.T) {
	t.Run("creates shodan command", func(t *testing.T) {
		cmd := NewShodanCmd()
		assert.NotNil(t, cmd)
		assert.Equal(t, "shodan", cmd.Use)
		assert.Equal(t, "Find self-registerable GitLab instances from Shodan search output", cmd.Short)
	})

	t.Run("has required json flag", func(t *testing.T) {
		cmd := NewShodanCmd()
		
		jsonFlag := cmd.Flags().Lookup("json")
		assert.NotNil(t, jsonFlag)
	})

	t.Run("has Run function assigned", func(t *testing.T) {
		cmd := NewShodanCmd()
		assert.NotNil(t, cmd.Run)
	})
}

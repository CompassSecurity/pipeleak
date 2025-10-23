package cicd

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewYamlCmd(t *testing.T) {
	t.Run("creates yaml command", func(t *testing.T) {
		cmd := NewYamlCmd()
		assert.NotNil(t, cmd)
		assert.Equal(t, "yaml", cmd.Use)
		assert.Equal(t, "Fetch full CI/CD yaml of project", cmd.Short)
	})

	t.Run("has repo flag", func(t *testing.T) {
		cmd := NewYamlCmd()
		repoFlag := cmd.Flags().Lookup("repo")
		assert.NotNil(t, repoFlag)
	})
}

package variables

import (
	"testing"

	"github.com/CompassSecurity/pipeleak/cmd/gitea"
	"github.com/stretchr/testify/assert"
)

func TestNewVariablesCommand(t *testing.T) {
	// Create parent command first
	parentCmd := gitea.NewGiteaRootCmd()
	cmd := NewVariablesCommand()
	parentCmd.AddCommand(cmd)

	assert.NotNil(t, cmd)
	assert.Equal(t, "variables", cmd.Use)
	assert.Contains(t, cmd.Short, "Actions variables")

	// Check that parent has the required flags
	tokenFlag := parentCmd.PersistentFlags().Lookup("token")
	assert.NotNil(t, tokenFlag)

	urlFlag := parentCmd.PersistentFlags().Lookup("gitea")
	assert.NotNil(t, urlFlag)
}

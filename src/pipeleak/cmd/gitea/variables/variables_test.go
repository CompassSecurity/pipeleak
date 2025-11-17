package variables

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewVariablesCommand(t *testing.T) {
	cmd := NewVariablesCommand()

	assert.NotNil(t, cmd)
	assert.Equal(t, "variables", cmd.Use)
	assert.Contains(t, cmd.Short, "Actions variables")

	urlFlag := cmd.Flags().Lookup("url")
	assert.NotNil(t, urlFlag)
	assert.Equal(t, "https://gitea.com", urlFlag.DefValue)

	tokenFlag := cmd.Flags().Lookup("token")
	assert.NotNil(t, tokenFlag)
	assert.Equal(t, "", tokenFlag.DefValue)
}

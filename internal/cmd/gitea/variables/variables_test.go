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
}

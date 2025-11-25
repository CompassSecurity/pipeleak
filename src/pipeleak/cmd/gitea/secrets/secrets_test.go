package secrets

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewSecretsCommand(t *testing.T) {
	cmd := NewSecretsCommand()

	assert.NotNil(t, cmd)
	assert.Equal(t, "secrets", cmd.Use)
	assert.Contains(t, cmd.Short, "Actions secrets")
}

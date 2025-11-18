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

	urlFlag := cmd.Flags().Lookup("url")
	assert.NotNil(t, urlFlag)
	assert.Equal(t, "https://gitea.com", urlFlag.DefValue)

	tokenFlag := cmd.Flags().Lookup("token")
	assert.NotNil(t, tokenFlag)
	assert.Equal(t, "", tokenFlag.DefValue)
}

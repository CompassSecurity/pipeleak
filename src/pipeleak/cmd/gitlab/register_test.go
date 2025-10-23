package gitlab

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewRegisterCmd(t *testing.T) {
	t.Run("creates register command", func(t *testing.T) {
		cmd := NewRegisterCmd()
		assert.NotNil(t, cmd)
		assert.Equal(t, "register", cmd.Use)
		assert.Equal(t, "Register a new user to a Gitlab instance", cmd.Short)
	})

	t.Run("has required flags", func(t *testing.T) {
		cmd := NewRegisterCmd()
		
		gitlabFlag := cmd.Flags().Lookup("gitlab")
		assert.NotNil(t, gitlabFlag)
		
		usernameFlag := cmd.Flags().Lookup("username")
		assert.NotNil(t, usernameFlag)
		
		passwordFlag := cmd.Flags().Lookup("password")
		assert.NotNil(t, passwordFlag)
		
		emailFlag := cmd.Flags().Lookup("email")
		assert.NotNil(t, emailFlag)
	})

	t.Run("has Run function assigned", func(t *testing.T) {
		cmd := NewRegisterCmd()
		assert.NotNil(t, cmd.Run)
	})
}

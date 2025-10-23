package securefiles

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewSecureFilesCmd(t *testing.T) {
	t.Run("creates secureFiles command", func(t *testing.T) {
		cmd := NewSecureFilesCmd()
		assert.NotNil(t, cmd)
		assert.Equal(t, "secureFiles", cmd.Use)
		assert.Equal(t, "Print CI/CD secure files", cmd.Short)
	})

	t.Run("has required flags", func(t *testing.T) {
		cmd := NewSecureFilesCmd()
		gitlabFlag := cmd.Flags().Lookup("gitlab")
		assert.NotNil(t, gitlabFlag)
		
		tokenFlag := cmd.Flags().Lookup("token")
		assert.NotNil(t, tokenFlag)
	})
}

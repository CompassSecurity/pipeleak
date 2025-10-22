package gitea

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewGiteaRootCmd(t *testing.T) {
	cmd := NewGiteaRootCmd()

	assert.NotNil(t, cmd)
	assert.Equal(t, "gitea [command]", cmd.Use)
	assert.Equal(t, "Gitea related commands", cmd.Short)
	assert.Equal(t, "Commands to enumerate and exploit Gitea instances.", cmd.Long)
	assert.Equal(t, "Gitea", cmd.GroupID)

	verboseFlag := cmd.PersistentFlags().Lookup("verbose")
	assert.NotNil(t, verboseFlag)
	assert.Equal(t, "v", verboseFlag.Shorthand)
	assert.Equal(t, "false", verboseFlag.DefValue)

	assert.True(t, cmd.HasSubCommands(), "Should have subcommands")

	subcommands := cmd.Commands()
	assert.GreaterOrEqual(t, len(subcommands), 2, "Should have at least enum and scan subcommands")

	var hasEnum, hasScan bool
	for _, subcmd := range subcommands {
		if subcmd.Name() == "enum" {
			hasEnum = true
		}
		if subcmd.Name() == "scan" {
			hasScan = true
		}
	}

	assert.True(t, hasEnum, "Should have enum subcommand")
	assert.True(t, hasScan, "Should have scan subcommand")
}

func TestGiteaRootCmd_VerboseFlag(t *testing.T) {
	cmd := NewGiteaRootCmd()

	err := cmd.ParseFlags([]string{"--verbose"})
	assert.NoError(t, err)

	verboseFlag := cmd.PersistentFlags().Lookup("verbose")
	assert.Equal(t, "true", verboseFlag.Value.String())
}

func TestGiteaRootCmd_ShortVerboseFlag(t *testing.T) {
	cmd := NewGiteaRootCmd()

	err := cmd.ParseFlags([]string{"-v"})
	assert.NoError(t, err)

	verboseFlag := cmd.PersistentFlags().Lookup("verbose")
	assert.Equal(t, "true", verboseFlag.Value.String())
}

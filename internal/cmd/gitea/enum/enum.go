package enum

import (
	pkgenum "github.com/CompassSecurity/pipeleek/pkg/gitea/enum"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
)

func NewEnumCmd() *cobra.Command {
	enumCmd := &cobra.Command{
		Use:     "enum",
		Short:   "Enumerate access of a Gitea token",
		Long:    "Enumerate access rights of a Gitea access token by retrieving the authenticated user's information, organizations with access levels, and all accessible repositories with permissions.",
		Example: `pipeleek gitea enum --token [tokenval] --gitea https://gitea.mycompany.com`,
		Run:     Enum,
	}

	return enumCmd
}

func Enum(cmd *cobra.Command, args []string) {
	giteaApiToken, _ := cmd.Flags().GetString("token")
	giteaUrl, _ := cmd.Flags().GetString("gitea")

	if err := pkgenum.RunEnum(giteaUrl, giteaApiToken); err != nil {
		log.Fatal().Stack().Err(err).Msg("Enumeration failed")
	}
}

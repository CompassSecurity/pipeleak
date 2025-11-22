package enum

import (
	pkgenum "github.com/CompassSecurity/pipeleak/pkg/gitea/enum"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
)

var (
	giteaApiToken string
	giteaUrl      string
)

func NewEnumCmd() *cobra.Command {
	enumCmd := &cobra.Command{
		Use:     "enum",
		Short:   "Enumerate access of a Gitea token",
		Long:    "Enumerate access rights of a Gitea access token by retrieving the authenticated user's information, organizations with access levels, and all accessible repositories with permissions.",
		Example: `pipeleak gitea enum --token [tokenval] --gitea https://gitea.mycompany.com`,
		Run:     Enum,
	}
	enumCmd.Flags().StringVarP(&giteaUrl, "gitea", "g", "https://gitea.com", "Gitea instance URL")
	enumCmd.Flags().StringVarP(&giteaApiToken, "token", "t", "", "Gitea API Token")

	return enumCmd
}

func Enum(cmd *cobra.Command, args []string) {
	if err := pkgenum.RunEnum(giteaUrl, giteaApiToken); err != nil {
		log.Fatal().Stack().Err(err).Msg("Enumeration failed")
	}
}

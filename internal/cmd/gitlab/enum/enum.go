package enum

import (
	pkgenum "github.com/CompassSecurity/pipeleak/pkg/gitlab/enum"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	gitlab "gitlab.com/gitlab-org/api/client-go"
)

var (
	gitlabApiToken string
	gitlabUrl      string
	minAccessLevel int
)

func NewEnumCmd() *cobra.Command {
	enumCmd := &cobra.Command{
		Use:     "enum",
		Short:   "Enumerate access rights of a GitLab access token",
		Long:    "Enumerate access rights of a GitLab access token by listing projects, groups and users the token has access to.",
		Example: `pipeleak gl enum --token glpat-xxxxxxxxxxx --gitlab https://gitlab.mydomain.com --level 20`,
		Run:     Enum,
	}
	enumCmd.Flags().StringVarP(&gitlabUrl, "gitlab", "g", "", "GitLab instance URL")
	err := enumCmd.MarkFlagRequired("gitlab")
	if err != nil {
		log.Fatal().Stack().Err(err).Msg("Unable to require gitlab flag")
	}

	enumCmd.Flags().StringVarP(&gitlabApiToken, "token", "t", "", "GitLab API Token")
	err = enumCmd.MarkFlagRequired("token")
	if err != nil {
		log.Fatal().Msg("Unable to require token flag")
	}
	enumCmd.MarkFlagsRequiredTogether("gitlab", "token")

	enumCmd.PersistentFlags().IntVarP(&minAccessLevel, "level", "", int(gitlab.GuestPermissions), "Minimum repo access level. See https://docs.gitlab.com/api/access_requests/#valid-access-levels for integer values")

	return enumCmd
}

func Enum(cmd *cobra.Command, args []string) {
	pkgenum.RunEnum(gitlabUrl, gitlabApiToken, minAccessLevel)
}

package shodan

import (
	pkgshodan "github.com/CompassSecurity/pipeleek/pkg/gitlab/shodan"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
)

var (
	shodanJson string
)

func NewShodanCmd() *cobra.Command {
	scanCmd := &cobra.Command{
		Use:     "shodan",
		Short:   "Find self-registerable GitLab instances from Shodan search output",
		Long:    "Use the Shodan command to identify GitLab instances that might allow for anyone to register. This command assumes a JSON file from a Shodan export. Example query: product:\"GitLab Self-Managed\"",
		Example: "pipeleek gl shodan --json shodan-export.json",
		Run:     Shodan,
	}

	scanCmd.Flags().StringVarP(&shodanJson, "json", "j", "", "Shodan search export JSON file path")
	err := scanCmd.MarkFlagRequired("json")
	if err != nil {
		log.Fatal().Stack().Err(err).Msg("Unable to parse shodan json flag")
	}

	return scanCmd
}

func Shodan(cmd *cobra.Command, args []string) {
	pkgshodan.RunShodan(shodanJson)
}

package github

import (
	"context"

	"github.com/CompassSecurity/pipeleak/helper"
	"github.com/rs/zerolog/log"
	"github.com/shurcooL/githubv4"
	"github.com/spf13/cobra"
	"golang.org/x/oauth2"
)

type GitHubScanOptions struct {
	AccessToken string
	Verbose bool
}

var options = GitHubScanOptions{}

func NewScanCmd() *cobra.Command {
	scanCmd := &cobra.Command{
		Use:   "scan [no options!]",
		Short: "Scan GitHub Actions",
		Run:   Scan,
	}
	scanCmd.Flags().StringVarP(&options.AccessToken, "token", "t", "", "GitHub Peronsal Access Token")
	err := scanCmd.MarkFlagRequired("token")
	if err != nil {
		log.Fatal().Msg("Unable to require token flag")
	}

	scanCmd.PersistentFlags().BoolVarP(&options.Verbose, "verbose", "v", false, "Verbose logging")

	return scanCmd
}

func Scan(cmd *cobra.Command, args []string) {
	helper.SetLogLevel(options.Verbose)
	// @todo this is buggy, does not refresh
	go helper.ShortcutListeners(0, 0)

	src := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: options.AccessToken},
	)
	httpClient := oauth2.NewClient(context.Background(), src)
	client := githubv4.NewClient(httpClient)
	ScanGithubActions(client)
	log.Info().Msg("Scan Finished, Bye Bye 🏳️‍🌈🔥")
}

func ScanGithubActions(client *githubv4.Client) {
	log.Info().Msg("Scanning GitHub Actions")
}

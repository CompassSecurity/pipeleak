package renovate

import (
	b64 "encoding/base64"
	"io"
	"regexp"
	"strings"

	"github.com/CompassSecurity/pipeleak/cmd/gitlab/util"
	"github.com/CompassSecurity/pipeleak/helper"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	gitlab "gitlab.com/gitlab-org/api/client-go"
)

var (
	gitlabApiToken     string
	gitlabUrl          string
	verbose            bool
	owned              bool
	member             bool
	projectSearchQuery string
	fast               bool
	selfHostedOptions  []string
)

func NewRenovateCmd() *cobra.Command {
	renovateCmd := &cobra.Command{
		Use:   "renovate [no options!]",
		Short: "Enumerate Renovate configurations",
		Run:   Enumerate,
	}

	renovateCmd.PersistentFlags().StringVarP(&gitlabUrl, "gitlab", "g", "", "GitLab instance URL")
	err := renovateCmd.MarkPersistentFlagRequired("gitlab")
	if err != nil {
		log.Fatal().Stack().Err(err).Msg("Unable to require gitlab flag")
	}

	renovateCmd.PersistentFlags().StringVarP(&gitlabApiToken, "token", "t", "", "GitLab API Token")
	err = renovateCmd.MarkPersistentFlagRequired("token")
	if err != nil {
		log.Error().Stack().Err(err).Msg("Unable to require token flag")
	}
	renovateCmd.MarkFlagsRequiredTogether("gitlab", "token")

	renovateCmd.PersistentFlags().BoolVarP(&owned, "owned", "o", false, "Scan user onwed projects only")
	renovateCmd.PersistentFlags().BoolVarP(&member, "member", "m", false, "Scan projects the user is member of")
	renovateCmd.Flags().StringVarP(&projectSearchQuery, "search", "s", "", "Query string for searching projects")
	renovateCmd.Flags().BoolVarP(&fast, "fast", "f", false, "Fast mode (skip renovate config file detection, only check CIDC yml for renovate bot job)")

	renovateCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "Verbose logging")

	return renovateCmd
}

func Enumerate(cmd *cobra.Command, args []string) {
	helper.SetLogLevel(verbose)
	git, err := util.GetGitlabClient(gitlabApiToken, gitlabUrl)
	if err != nil {
		log.Fatal().Stack().Err(err).Msg("failed creating gitlab client")
	}

	fetchProjects(git)

	log.Info().Msg("Done, Bye Bye 🏳️‍🌈🔥")
}

func fetchProjects(git *gitlab.Client) {
	log.Info().Msg("Fetching projects")

	projectOpts := &gitlab.ListProjectsOptions{
		ListOptions: gitlab.ListOptions{
			PerPage: 100,
			Page:    1,
		},
		OrderBy:    gitlab.Ptr("last_activity_at"),
		Owned:      gitlab.Ptr(owned),
		Membership: gitlab.Ptr(member),
		Search:     gitlab.Ptr(projectSearchQuery),
	}

	for {
		projects, resp, err := git.Projects.ListProjects(projectOpts)
		if err != nil {
			log.Error().Stack().Err(err).Msg("Failed fetching projects")
			break
		}

		for _, project := range projects {
			log.Debug().Str("url", project.WebURL).Msg("Check project")
			identifyRenovateBotJob(git, project)
		}

		if resp.NextPage == 0 {
			break
		}

		projectOpts.Page = resp.NextPage
		log.Info().Int("currentPage", projectOpts.Page).Msg("Fetched projects page")
	}

	log.Info().Msg("Fetched all projects")
}

func identifyRenovateBotJob(git *gitlab.Client, project *gitlab.Project) {
	ciCdYml := fetchCICDYml(git, project.ID)
	hasCiCdRenovateConfig := detectCiCdConfig(ciCdYml)
	var configFile *gitlab.File = nil
	var configFileContent string
	if !fast {
		configFile, configFileContent = detectRenovateConfigFile(git, project)
	}

	if hasCiCdRenovateConfig || configFile != nil {
		selfHostedConfigFile := false
		if configFile != nil {
			selfHostedConfigFile = isSelfHostedConfig(configFileContent)
		}
		autodiscovery := detectAutodiscovery(ciCdYml, configFileContent)
		log.Warn().Str("pipelines", string(project.BuildsAccessLevel)).Bool("hasAutodiscovery", autodiscovery).Bool("hasConfigFile", configFile != nil).Bool("selfHostedConfigFile", selfHostedConfigFile).Str("url", project.WebURL).Msg("Identified Renovate (bot) configuration")

		if verbose && hasCiCdRenovateConfig {
			yml, err := helper.PrettyPrintYAML(ciCdYml)
			if err != nil {
				log.Error().Stack().Err(err).Msg("Failed pretty printing project CI/CD YML")
				return
			}
			log.Info().Msg(helper.GetPlatformAgnosticNewline() + yml)
		}
	}
}

func detectCiCdConfig(cicdConf string) bool {
	// Check for common Renovate bot job identifiers in CI/CD configuration
	return helper.ContainsI(cicdConf, "renovate/renovate") ||
		helper.ContainsI(cicdConf, "renovatebot/renovate") ||
		helper.ContainsI(cicdConf, "renovate-bot/renovate-runner") ||
		helper.ContainsI(cicdConf, "RENOVATE_")
}

func detectAutodiscovery(cicdConf string, configFileContent string) bool {
	// Check for autodiscover flag: https://docs.renovatebot.com/self-hosted-configuration/#autodiscover
	hasAutodiscoveryInConfigFile := helper.ContainsI(configFileContent, "autodiscover")

	hasAutodiscoveryinCiCD := (helper.ContainsI(cicdConf, "--autodiscover") || helper.ContainsI(cicdConf, "RENOVATE_AUTODISCOVER")) &&
		(!helper.ContainsI(cicdConf, "--autodiscover=false") && !helper.ContainsI(cicdConf, "--autodiscover false") && !helper.ContainsI(cicdConf, "RENOVATE_AUTODISCOVER: false") && !helper.ContainsI(cicdConf, "RENOVATE_AUTODISCOVER=false"))

	return hasAutodiscoveryInConfigFile || hasAutodiscoveryinCiCD
}

func detectRenovateConfigFile(git *gitlab.Client, project *gitlab.Project) (*gitlab.File, string) {
	// https://docs.renovatebot.com/configuration-options/
	configFiles := []string{
		"renovate.json",
		"renovate.json5",
		".github/renovate.json",
		".github/renovate.json5",
		".gitlab/renovate.json",
		".gitlab/renovate.json5",
		".renovaterc",
		".renovaterc.json",
		".renovaterc.json5",
	}

	opts := gitlab.GetFileOptions{Ref: gitlab.Ptr(project.DefaultBranch)}
	for _, configFile := range configFiles {
		file, _, err := git.RepositoryFiles.GetFile(project.ID, configFile, &opts)
		if err != nil {
			continue
		}

		if file != nil {
			conf, err := b64.StdEncoding.DecodeString(file.Content)
			if err != nil {
				log.Error().Stack().Err(err).Msg("Failed decoding renovate config base64 content")
				return file, ""
			}

			return file, string(conf)
		}
	}

	return nil, ""
}

func fetchCurrentSelfHostedOptions() []string {
	if len(selfHostedOptions) > 0 {
		return selfHostedOptions
	}

	log.Debug().Msg("Fetching current self-hosted configuration from GitHub")

	client := helper.GetPipeleakHTTPClient()
	res, err := client.Get("https://raw.githubusercontent.com/renovatebot/renovate/refs/heads/main/docs/usage/self-hosted-configuration.md")
	if err != nil {
		log.Fatal().Stack().Err(err).Msg("Failed fetching self-hosted configuration documentation")
		return []string{}
	}
	defer res.Body.Close()
	if res.StatusCode != 200 {
		log.Fatal().Int("status", res.StatusCode).Msg("Failed fetching self-hosted configuration documentation")
		return []string{}
	}
	data, err := io.ReadAll(res.Body)
	if err != nil {
		log.Fatal().Stack().Err(err).Msg("Failed reading self-hosted configuration documentation")
		return []string{}
	}

	selfHostedOptions = extractSelfHostedOptions(data)
	return selfHostedOptions
}

func extractSelfHostedOptions(data []byte) []string {
	var re = regexp.MustCompile(`(?m)## .*`)
	matches := re.FindAllString(string(data), -1)

	var options []string
	for _, match := range matches {
		options = append(options, strings.ReplaceAll(strings.TrimSpace(match), "## ", ""))
	}

	return options
}

func isSelfHostedConfig(config string) bool {
	selfHostedOptions := fetchCurrentSelfHostedOptions()
	for _, option := range selfHostedOptions {
		// Check if the content contains any of the self-hosted options
		if helper.ContainsI(config, option) {
			return true
		}
	}
	return false
}

func fetchCICDYml(git *gitlab.Client, pid int) string {
	lintOpts := &gitlab.ProjectLintOptions{
		IncludeJobs: gitlab.Ptr(true),
	}
	res, response, err := git.Validate.ProjectLint(pid, lintOpts)

	if response.StatusCode == 404 || response.StatusCode == 403 {
		return "" // Project does not have a CI/CD configuration or is not accessible
	}

	if err != nil {
		log.Error().Stack().Err(err).Msg("Failed fetching project CI/CD YML")
		return ""
	}

	return res.MergedYaml
}

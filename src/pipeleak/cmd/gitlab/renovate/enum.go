package renovate

import (
	b64 "encoding/base64"
	"io"
	"regexp"
	"strings"
	"sync"

	"github.com/CompassSecurity/pipeleak/cmd/gitlab/util"
	"github.com/CompassSecurity/pipeleak/helper"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	gitlab "gitlab.com/gitlab-org/api/client-go"
)

var (
	owned              bool
	member             bool
	projectSearchQuery string
	fast               bool
	selfHostedOptions  []string
	page               int
	repository         string
)

func NewEnumCmd() *cobra.Command {
	enumCmd := &cobra.Command{
		Use:   "enum [no options!]",
		Short: "Enumerate Renovate configurations",
		Run:   Enumerate,
	}

	enumCmd.PersistentFlags().StringVarP(&gitlabUrl, "gitlab", "g", "", "GitLab instance URL")
	err := enumCmd.MarkPersistentFlagRequired("gitlab")
	if err != nil {
		log.Fatal().Stack().Err(err).Msg("Unable to require gitlab flag")
	}

	enumCmd.PersistentFlags().StringVarP(&gitlabApiToken, "token", "t", "", "GitLab API Token")
	err = enumCmd.MarkPersistentFlagRequired("token")
	if err != nil {
		log.Error().Stack().Err(err).Msg("Unable to require token flag")
	}
	enumCmd.MarkFlagsRequiredTogether("gitlab", "token")

	enumCmd.PersistentFlags().BoolVarP(&owned, "owned", "o", false, "Scan user owned projects only")
	enumCmd.PersistentFlags().BoolVarP(&member, "member", "m", false, "Scan projects the user is member of")
	enumCmd.Flags().StringVarP(&repository, "repo", "r", "", "Repository to scan for Renovate configuraiton (if not set, all projects will be scanned)")
	enumCmd.Flags().StringVarP(&projectSearchQuery, "search", "s", "", "Query string for searching projects")
	enumCmd.Flags().BoolVarP(&fast, "fast", "f", false, "Fast mode - skip renovate config file detection, only check CIDC yml for renovate bot job (default false)")
	enumCmd.Flags().IntVarP(&page, "page", "p", 1, "Page number to start fetching projects from (default 1, fetch all pages)")

	enumCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "Verbose logging")

	return enumCmd
}

func Enumerate(cmd *cobra.Command, args []string) {
	helper.SetLogLevel(verbose)
	git, err := util.GetGitlabClient(gitlabApiToken, gitlabUrl)
	if err != nil {
		log.Fatal().Stack().Err(err).Msg("Failed creating gitlab client")
	}

	if repository != "" {
		scanSingleProject(git, repository)
	} else {
		fetchProjects(git)
	}

	log.Info().Msg("Done, Bye Bye 🏳️‍🌈🔥")
}

func scanSingleProject(git *gitlab.Client, projectName string) {
	log.Info().Str("repository", projectName).Msg("Scanning specific repository for Renovate configuration")
	project, resp, err := git.Projects.GetProject(projectName, &gitlab.GetProjectOptions{})
	if err != nil {
		log.Fatal().Stack().Err(err).Msg("Failed fetching project by repository name")
	}
	if resp.StatusCode == 404 {
		log.Fatal().Msg("Project not found")
	}
	identifyRenovateBotJob(git, project)
}

func fetchProjects(git *gitlab.Client) {
	log.Info().Msg("Fetching projects")

	projectOpts := &gitlab.ListProjectsOptions{
		ListOptions: gitlab.ListOptions{
			PerPage: 100,
			Page:    page,
		},
		OrderBy:    gitlab.Ptr("last_activity_at"),
		Owned:      gitlab.Ptr(owned),
		Membership: gitlab.Ptr(member),
		Search:     gitlab.Ptr(projectSearchQuery),
	}

	for {
		projects, resp, err := git.Projects.ListProjects(projectOpts)
		if err != nil {
			log.Error().Stack().Err(err).Int("page", page).Msg("Failed fetching projects")
			break
		}

		var wg sync.WaitGroup
		for _, project := range projects {
			wg.Add(1)
			go func(proj *gitlab.Project) {
				defer wg.Done()
				log.Debug().Str("url", proj.WebURL).Msg("Check project")
				identifyRenovateBotJob(git, proj)
			}(project)
		}
		wg.Wait()

		if resp.NextPage == 0 {
			break
		}

		projectOpts.Page = resp.NextPage
		log.Info().Int("currentPage", projectOpts.Page).Msg("Fetched projects page")
	}

	log.Info().Msg("Fetched all projects")
}

func identifyRenovateBotJob(git *gitlab.Client, project *gitlab.Project) {
	ciCdYml, err := util.FetchCICDYml(git, project.ID)
	if err != nil {
		// silently skip
		return
	}

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
		autodiscoveryFilters := false
		if autodiscovery {
			autodiscoveryFilters = detectAutodiscoveryFilters(ciCdYml, configFileContent)
		}

		log.Warn().Str("pipelines", string(project.BuildsAccessLevel)).Bool("hasAutodiscovery", autodiscovery).Bool("hasAutodiscoveryFilters", autodiscoveryFilters).Bool("hasConfigFile", configFile != nil).Bool("selfHostedConfigFile", selfHostedConfigFile).Str("url", project.WebURL).Msg("Identified Renovate (bot) configuration")

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
		helper.ContainsI(cicdConf, "RENOVATE_") ||
		helper.ContainsI(cicdConf, "npx renovate")
}

func detectAutodiscovery(cicdConf string, configFileContent string) bool {
	// Check for autodiscover flag: https://docs.renovatebot.com/self-hosted-configuration/#autodiscover
	hasAutodiscoveryInConfigFile := helper.ContainsI(configFileContent, "autodiscover")

	hasAutodiscoveryinCiCD := (helper.ContainsI(cicdConf, "--autodiscover") || helper.ContainsI(cicdConf, "RENOVATE_AUTODISCOVER")) &&
		(!helper.ContainsI(cicdConf, "--autodiscover=false") && !helper.ContainsI(cicdConf, "--autodiscover false") && !helper.ContainsI(cicdConf, "RENOVATE_AUTODISCOVER: false") && !helper.ContainsI(cicdConf, "RENOVATE_AUTODISCOVER=false"))

	return hasAutodiscoveryInConfigFile || hasAutodiscoveryinCiCD
}

func detectAutodiscoveryFilters(cicdConf string, configFileContent string) bool {
	// https://docs.renovatebot.com/self-hosted-configuration/#autodiscoverfilter
	// https://docs.renovatebot.com/self-hosted-configuration/#autodiscovernamespaces
	// https://docs.renovatebot.com/self-hosted-configuration/#autodiscoverprojects
	// https://docs.renovatebot.com/self-hosted-configuration/#autodiscovertopics

	hasFilter := helper.ContainsI(configFileContent, "autodiscoverFilter") || helper.ContainsI(cicdConf, "RENOVATE_AUTODISCOVER_FILTER") || helper.ContainsI(cicdConf, "--autodiscover-filter")
	hasNamespaces := helper.ContainsI(configFileContent, "autodiscoverNamespaces") || helper.ContainsI(cicdConf, "RENOVATE_AUTODISCOVER_NAMESPACES") || helper.ContainsI(cicdConf, "--autodiscover-namespaces")
	hasProjects := helper.ContainsI(configFileContent, "autodiscoverProjects") || helper.ContainsI(cicdConf, "RENOVATE_AUTODISCOVER_PROJECTS") || helper.ContainsI(cicdConf, "--autodiscover-projects")
	hasTopics := helper.ContainsI(configFileContent, "autodiscoverTopics") || helper.ContainsI(cicdConf, "RENOVATE_AUTODISCOVER_TOPICS") || helper.ContainsI(cicdConf, "--autodiscover-topics")

	return hasFilter || hasNamespaces || hasProjects || hasTopics
}

// detectRenovateConfigFile checks for common Renovate configuration files in the project repository
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

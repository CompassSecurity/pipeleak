package renovate

import (
	b64 "encoding/base64"
	"regexp"
	"strings"
	"sync"

	"github.com/CompassSecurity/pipeleak/cmd/gitlab/util"
	"github.com/CompassSecurity/pipeleak/helper"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	gitlab "gitlab.com/gitlab-org/api/client-go"
	"gopkg.in/yaml.v3"
)

var (
	owned              bool
	member             bool
	projectSearchQuery string
	fast               bool
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

	type gitlabJob struct {
		Image  string   `yaml:"image"`
		Script []string `yaml:"script"`
	}
	var parsed map[string]interface{}
	found := false
	indicatorCount := 0

	if err := yaml.Unmarshal([]byte(ciCdYml), &parsed); err == nil {

		// Job detection
		for k, v := range parsed {
			job, ok := v.(map[string]interface{})
			if !ok {
				continue
			}
			// Heuristic: job name contains renovate or dependency
			if strings.Contains(strings.ToLower(k), "renovate") || strings.Contains(strings.ToLower(k), "dependency") {
				indicatorCount++
				if !found {
					log.Info().Str("job", k).Str("url", project.WebURL).Int("indicators", indicatorCount).Msg("Detected Renovate job by job name")
					found = true
				}
			}

			// Heuristic: stage name contains renovate or dependency
			if stage, ok := job["stage"].(string); ok && (strings.Contains(strings.ToLower(stage), "renovate") || strings.Contains(strings.ToLower(stage), "dependency")) {
				indicatorCount++
				if !found {
					log.Info().Str("job", k).Str("stage", stage).Str("url", project.WebURL).Int("indicators", indicatorCount).Msg("Detected Renovate job by stage")
					found = true
				}
			}

			if img, ok := job["image"].(string); ok && (strings.Contains(strings.ToLower(img), "renovate")) {
				indicatorCount++
				if !found {
					log.Info().Str("job", k).Str("image", img).Str("url", project.WebURL).Int("indicators", indicatorCount).Msg("Detected Renovate job by image")
					found = true
				}
			}
			if script, ok := job["script"].([]interface{}); ok {
				for _, s := range script {
					if str, ok := s.(string); ok && (strings.Contains(strings.ToLower(str), "renovate")) {
						indicatorCount++
						if !found {
							log.Info().Str("job", k).Str("script", str).Str("url", project.WebURL).Int("indicators", indicatorCount).Msg("Detected Renovate job by script")
							found = true
						}
					}
				}
			}
		}

		// Variable and tag detection
		if variables, ok := parsed["variables"].(map[string]interface{}); ok {
			for varName := range variables {
				if strings.Contains(strings.ToUpper(varName), "RENOVATE") {
					indicatorCount++
					if !found {
						log.Info().Str("variable", varName).Str("url", project.WebURL).Int("indicators", indicatorCount).Msg("Detected Renovate job by variable")
						found = true
					}
				}
			}
		}

		// Check tags in jobs
		for k, v := range parsed {
			job, ok := v.(map[string]interface{})
			if !ok {
				continue
			}
			if tags, ok := job["tags"].([]interface{}); ok {
				for _, t := range tags {
					if tagStr, ok := t.(string); ok && strings.Contains(strings.ToLower(tagStr), "renovate") {
						indicatorCount++
						if !found {
							log.Info().Str("job", k).Str("tag", tagStr).Str("url", project.WebURL).Int("indicators", indicatorCount).Msg("Detected Renovate job by tag")
							found = true
						}
					}
				}
			}
		}

		// Anchors/Aliases and Includes
		// Simple regex for YAML anchors/aliases
		anchorRegex := regexp.MustCompile(`&([a-zA-Z0-9_-]*renovate[a-zA-Z0-9_-]*)|&([a-zA-Z0-9_-]*dependency[a-zA-Z0-9_-]*)`)
		aliasRegex := regexp.MustCompile(`\*([a-zA-Z0-9_-]*renovate[a-zA-Z0-9_-]*)|\*([a-zA-Z0-9_-]*dependency[a-zA-Z0-9_-]*)`)
		if anchorRegex.MatchString(ciCdYml) || aliasRegex.MatchString(ciCdYml) {
			indicatorCount++
			if !found {
				log.Info().Str("url", project.WebURL).Int("indicators", indicatorCount).Msg("Detected Renovate anchor/alias in CI/CD YAML")
				found = true
			}
		}

		// Check for includes
		if includes, ok := parsed["include"]; ok {
			// includes can be a string, a map, or a list
			var checkInclude func(val interface{}) bool
			checkInclude = func(val interface{}) bool {
				switch v := val.(type) {
				case string:
					return strings.Contains(strings.ToLower(v), "renovate") || strings.Contains(strings.ToLower(v), "dependency")
				case map[string]interface{}:
					for _, vv := range v {
						if checkInclude(vv) {
							return true
						}
					}
				case []interface{}:
					for _, vv := range v {
						if checkInclude(vv) {
							return true
						}
					}
				}
				return false
			}
			if checkInclude(includes) {
				indicatorCount++
				if !found {
					log.Info().Str("url", project.WebURL).Int("indicators", indicatorCount).Msg("Detected Renovate include in CI/CD YAML")
					found = true
				}
			}
		}
	}

	var configFile *gitlab.File = nil
	var configFileContent string
	if !fast {
		configFile, configFileContent = detectRenovateConfigFile(git, project)
	}

	if found || configFile != nil {
		selfHostedConfigFile := false
		if configFile != nil {
			selfHostedConfigFile = isSelfHostedConfig(configFileContent)
		}

		autodiscovery := detectAutodiscovery(ciCdYml, configFileContent)
		autodiscoveryFilters := false
		if autodiscovery {
			autodiscoveryFilters = detectAutodiscoveryFilters(ciCdYml, configFileContent)
		}

		log.Warn().Str("pipelines", string(project.BuildsAccessLevel)).Bool("vulnerable", autodiscovery && !autodiscoveryFilters).Bool("hasAutodiscovery", autodiscovery).Bool("hasAutodiscoveryFilters", autodiscoveryFilters).Bool("hasConfigFile", configFile != nil).Bool("selfHostedConfigFile", selfHostedConfigFile).Str("url", project.WebURL).Msg("Identified Renovate (bot) configuration")

		if verbose && found {
			yml, err := helper.PrettyPrintYAML(ciCdYml)
			if err != nil {
				log.Error().Stack().Err(err).Msg("Failed pretty printing project CI/CD YML")
				return
			}
			log.Info().Msg(helper.GetPlatformAgnosticNewline() + yml)
		}
	}
}

func detectAutodiscovery(cicdConf string, configFileContent string) bool {
	if helper.ContainsI(cicdConf, "autodiscover") {
		// Exclude explicit disables
		disables := []string{"autodiscover=false", "--autodiscover=false", "--autodiscover false", "RENOVATE_AUTODISCOVER: false", "RENOVATE_AUTODISCOVER=false"}
		for _, d := range disables {
			if helper.ContainsI(configFileContent, d) {
				return false
			}
		}
	}

	// Check for Renovate CLI flags or env vars enabling autodiscover
	cliFlags := []string{"--autodiscover", "--autodiscover=true", "--autodiscover true"}
	for _, flag := range cliFlags {
		if helper.ContainsI(cicdConf, flag) {
			return true
		}
	}

	envVars := []string{"RENOVATE_AUTODISCOVER=1", "RENOVATE_AUTODISCOVER=true", "RENOVATE_AUTODISCOVER: true"}
	for _, env := range envVars {
		if helper.ContainsI(cicdConf, env) {
			return true
		}
	}

	// Check for Renovate config keys that imply autodiscovery (e.g., autodiscoverFilter, autodiscoverNamespaces, etc.)
	configKeys := []string{"autodiscoverFilter", "autodiscoverNamespaces", "autodiscoverProjects", "autodiscoverTopics"}
	for _, key := range configKeys {
		if helper.ContainsI(configFileContent, key) {
			return true
		}
	}

	return false
}

func detectAutodiscoveryFilters(cicdConf string, configFileContent string) bool {
	// Check for filter-related keys, env vars, or CLI flags in both config and CI/CD
	filterKeys := []string{
		"autodiscoverFilter", "autodiscoverFilters", "autodiscoverNamespaces", "autodiscoverProjects", "autodiscoverTopics",
		"RENOVATE_AUTODISCOVER_FILTER", "RENOVATE_AUTODISCOVER_FILTERS", "RENOVATE_AUTODISCOVER_NAMESPACES", "RENOVATE_AUTODISCOVER_PROJECTS", "RENOVATE_AUTODISCOVER_TOPICS",
		"--autodiscover-filter", "--autodiscover-filters", "--autodiscover-namespaces", "--autodiscover-projects", "--autodiscover-topics",
	}
	for _, key := range filterKeys {
		if helper.ContainsI(configFileContent, key) || helper.ContainsI(cicdConf, key) {
			return true
		}
	}

	return false
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

func isSelfHostedConfig(config string) bool {
	// Look for common self-hosted Renovate config keys/flags
	configLower := strings.ToLower(config)
	keys := []string{
		"allowCustomCrateRegistries",
		"allowPlugins",
		"allowScripts",
		"allowedCommands",
		"allowedEnv",
		"allowedHeaders",
		"autodiscover",
		"autodiscoverFilter",
		"autodiscoverNamespaces",
		"autodiscoverProjects",
		"autodiscoverRepoOrder",
		"autodiscoverRepoSort",
		"autodiscoverTopics",
		"baseDir",
		"bbUseDevelopmentBranch",
		"binarySource",
		"cacheDir",
		"cacheHardTtlMinutes",
		"cachePrivatePackages",
		"cacheTtlOverride",
		"checkedBranches",
		"containerbaseDir",
		"customEnvVariables",
		"deleteConfigFile",
		"detectGlobalManagerConfig",
		"detectHostRulesFromEnv",
		"dockerChildPrefix",
		"dockerCliOptions",
		"dockerMaxPages",
		"dockerSidecarImage",
		"dockerUser",
		"dryRun",
		"encryptedWarning",
		"endpoint",
		"executionTimeout",
		"exposeAllEnv",
		"force",
		"forceCli",
		"forkCreation",
		"forkOrg",
		"forkToken",
		"gitNoVerify",
		"gitPrivateKey",
		"gitTimeout",
		"gitUrl",
		"githubTokenWarn",
		"globalExtends",
		"httpCacheTtlDays",
		"includeMirrors",
		"inheritConfig",
		"inheritConfigFileName",
		"inheritConfigRepoName",
		"inheritConfigStrict",
		"logContext",
		"mergeConfidenceDatasources",
		"mergeConfidenceEndpoint",
		"migratePresets",
		"onboarding",
		"onboardingBranch",
		"onboardingCommitMessage",
		"onboardingConfig",
		"onboardingConfigFileName",
		"onboardingNoDeps",
		"onboardingPrTitle",
		"onboardingRebaseCheckbox",
		"optimizeForDisabled",
		"password",
		"persistRepoData",
		"platform",
		"prCommitsPerRunLimit",
		"presetCachePersistence",
		"privateKey",
		"privateKeyOld",
		"privateKeyPath",
		"privateKeyPathOld",
		"processEnv",
		"productLinks",
		"redisPrefix",
		"redisUrl",
		"reportPath",
		"reportType",
		"repositories",
		"repositoryCache",
		"repositoryCacheType",
		"requireConfig",
		"s3Endpoint",
		"s3PathStyle",
		"secrets",
		"token",
		"unicodeEmoji",
		"useCloudMetadataServices",
		"userAgent",
		"username",
		"variables",
		"writeDiscoveredRepo",
	}
	for _, key := range keys {
		if strings.Contains(configLower, key) {
			return true
		}
	}

	return false
}

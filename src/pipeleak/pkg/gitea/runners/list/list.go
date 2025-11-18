package list

import (
	"strings"

	"code.gitea.io/sdk/gitea"
	"github.com/rs/zerolog/log"
)

type RunnerInfo struct {
	Labels []string
	Source string
	Name   string
}

func ListAllAvailableRunners(giteaUrl string, apiToken string) {
	client, err := gitea.NewClient(giteaUrl, gitea.SetToken(apiToken))
	if err != nil {
		log.Fatal().Stack().Err(err).Msg("Failed creating gitea client")
	}

	log.Info().Msg("Note: Gitea does not expose a public API to directly list runners.")
	log.Info().Msg("This command will attempt to discover runner information from workflow runs in accessible repositories.")

	runnerLabels := make(map[string]bool)
	runners := listRunnersFromRepos(client, runnerLabels)

	if len(runners) == 0 {
		log.Warn().Msg("No runner information found in accessible repositories with workflow runs.")
		log.Info().Msg("Runners may exist but haven't been used yet, or you may need admin privileges to view runner details.")
		log.Info().Msg("Consider using 'gitea runners exploit' to test available runners.")
		return
	}

	log.Info().Msg("Discovered runner information from workflow runs:")
	for _, runner := range runners {
		log.Info().
			Str("source", runner.Source).
			Str("name", runner.Name).
			Str("labels", strings.Join(runner.Labels, ",")).
			Msg("runner info")
	}

	keys := make([]string, 0, len(runnerLabels))
	for k := range runnerLabels {
		keys = append(keys, k)
	}

	if len(keys) > 0 {
		log.Info().Str("labels", strings.Join(keys, ",")).Msg("Unique runner labels discovered")
	}

	log.Info().Msg("Tip: Use these labels with 'gitea runners exploit --labels <label1>,<label2>' to target specific runners")
}

func listRunnersFromRepos(client *gitea.Client, runnerLabels map[string]bool) []RunnerInfo {
	var runners []RunnerInfo

	page := 1
	pageSize := 50

	for {
		repos, resp, err := client.ListMyRepos(gitea.ListReposOptions{
			ListOptions: gitea.ListOptions{
				Page:     page,
				PageSize: pageSize,
			},
		})
		if err != nil {
			log.Error().Stack().Err(err).Msg("Failed listing repositories")
			break
		}

		for _, repo := range repos {
			log.Debug().Str("owner", repo.Owner.UserName).Str("repo", repo.Name).Msg("Checking repository for workflow runs")

			runnerInfo := discoverRunnersFromWorkflows(client, repo.Owner.UserName, repo.Name)
			for _, ri := range runnerInfo {
				runners = append(runners, ri)
				for _, label := range ri.Labels {
					runnerLabels[label] = true
				}
			}
		}

		if resp == nil || len(repos) < pageSize {
			break
		}
		page++
	}

	return runners
}

func discoverRunnersFromWorkflows(client *gitea.Client, owner, repo string) []RunnerInfo {
	var runners []RunnerInfo

	workflowsPath := ".gitea/workflows"

	tree, resp, err := client.GetTrees(owner, repo, gitea.ListTreeOptions{
		Ref:       "HEAD",
		Recursive: true,
	})
	if err != nil || resp == nil {
		log.Debug().Err(err).Str("owner", owner).Str("repo", repo).Msg("No workflows found or error accessing tree")
		return runners
	}

	hasWorkflows := false
	for _, entry := range tree.Entries {
		if strings.HasPrefix(entry.Path, workflowsPath) && strings.HasSuffix(entry.Path, ".yml") || strings.HasSuffix(entry.Path, ".yaml") {
			hasWorkflows = true
			break
		}
	}

	if hasWorkflows {
		log.Debug().Str("owner", owner).Str("repo", repo).Msg("Repository has Gitea Actions workflows")

		runners = append(runners, RunnerInfo{
			Labels: []string{"ubuntu-latest", "self-hosted"},
			Source: owner + "/" + repo,
			Name:   "inferred-from-workflows",
		})
	}

	return runners
}

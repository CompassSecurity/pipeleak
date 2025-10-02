package github

import (
	"context"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/CompassSecurity/pipeleak/helper"
	"github.com/CompassSecurity/pipeleak/scanner"
	"github.com/google/go-github/v69/github"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
)

func NewActionScanCmd() *cobra.Command {
	scanActionCmd := &cobra.Command{
		Use:     "action",
		Short:   "Scan all jobs of the action workflow running",
		Long:    `Scan GitHub Actions workflow runs and artifacts for secrets`,
		Example: `pipeleak gh action -t $GH_TOKEN`,
		Run:     ScanAction,
	}

	scanActionCmd.PersistentFlags().BoolVarP(&options.Verbose, "verbose", "v", false, "Verbose logging")

	return scanActionCmd
}

func ScanAction(cmd *cobra.Command, args []string) {
	options.HttpClient = helper.GetPipeleakHTTPClient()
	helper.SetLogLevel(options.Verbose)
	scanner.InitRules(options.ConfidenceFilter)
	scanWorkflowRuns()
	log.Info().Msg("Scan Finished, Bye Bye 🏳️‍🌈🔥")
}

func scanWorkflowRuns() {
	log.Info().Msg("Scanning GitHub Actions workflow runs for secrets")
	ctx := context.WithValue(context.Background(), github.BypassRateLimitCheck, true)

	token := os.Getenv("GITHUB_TOKEN")
	if token == "" {
		log.Fatal().Msg("GITHUB_TOKEN not set")
	}

	client := setupClient(token)
	options.Client = client

	repoFull := os.Getenv("GITHUB_REPOSITORY")
	if repoFull == "" {
		log.Fatal().Msg("GITHUB_REPOSITORY not set")
	}

	parts := strings.Split(repoFull, "/")
	if len(parts) != 2 {
		log.Fatal().Str("repository", repoFull).Msg("invalid GITHUB_REPOSITORY")
	}

	owner, repo := parts[0], parts[1]
	log.Info().Str("owner", owner).Str("repo", repo).Msg("Repository to scan")

	repository, _, err := client.Repositories.Get(ctx, owner, repo)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to fetch repository")
	}

	sha := os.Getenv("GITHUB_SHA")
	if sha == "" {
		log.Fatal().Msg("GITHUB_SHA not set")
	} else {
		log.Info().Str("sha", sha).Msg("Current commit sha")
	}

	runIDStr := os.Getenv("GITHUB_RUN_ID")
	if runIDStr == "" {
		log.Fatal().Msg("GITHUB_RUN_ID not set")
	} else {
		log.Info().Str("runID", runIDStr).Msg("Current run ID")
	}

	currentRunID, _ := strconv.ParseInt(runIDStr, 10, 64)

	for {
		allCompleted := true

		opts := &github.ListWorkflowRunsOptions{
			ListOptions: github.ListOptions{PerPage: 100},
			HeadSHA:     sha,
		}

		for {
			runs, resp, err := client.Actions.ListRepositoryWorkflowRuns(ctx, owner, repo, opts)

			log.Info().Int("count", len(runs.WorkflowRuns)).Msg("Fetched workflow runs")

			if err != nil {
				log.Fatal().Stack().Err(err).Msg("Failed listing workflow runs")
			}

			for _, run := range runs.WorkflowRuns {
				if run.GetID() != currentRunID {
					log.Info().Int64("run", run.GetID()).Str("commit", run.GetHeadSHA()).Str("name", run.GetName()).Msg("Check run")
					status := run.GetStatus()
					log.Info().Int64("run", run.GetID()).Str("status", status).Str("name", run.GetName()).Msgf("Run")

					if status != "completed" {
						allCompleted = false
						if _, scanned := scannedRuns[run.GetID()]; !scanned {
							if status == "completed" {
								scannedRuns[run.GetID()] = struct{}{}
								wg.Add(1)
								go func(runCopy *github.WorkflowRun) {
									defer wg.Done()
									scanRun(client, repository, runCopy)
								}(run)
							}
						}
					}
				}
			}

			if resp.NextPage == 0 {
				break
			}

			opts.Page = resp.NextPage
		}

		if allCompleted {
			log.Info().Msg("✅ All *other* runs for this commit are completed")
			break
		}

		log.Info().Msg("⏳ Still waiting... retrying in 30s")
		time.Sleep(30 * time.Second)
	}
}

func scanRun(client *github.Client, repo *github.Repository, workflowRun *github.WorkflowRun) {
	downloadWorkflowRunLog(client, repo, workflowRun)
	listArtifacts(client, workflowRun)
}

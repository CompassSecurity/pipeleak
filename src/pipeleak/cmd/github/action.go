package github

import (
	"context"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/CompassSecurity/pipeleak/helper"
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
	helper.SetLogLevel(options.Verbose)
	scanWorkflowRuns()
	log.Info().Msg("Scan Finished, Bye Bye 🏳️‍🌈🔥")
}

func scanWorkflowRuns() {
	log.Info().Msg("Scanning GitHub Actions workflow runs for secrets")
	ctx := context.WithValue(context.Background(), github.BypassRateLimitCheck, true)
	client := setupClient(options.AccessToken)

	token := os.Getenv("GITHUB_TOKEN")
	if token == "" {
		log.Fatal().Msg("GITHUB_TOKEN not set")
	}

	repoFull := os.Getenv("GITHUB_REPOSITORY")
	if token == "" {
		log.Fatal().Msg("GITHUB_REPOSITORY not set")
	}

	parts := strings.Split(repoFull, "/")
	if len(parts) != 2 {
		log.Fatal().Str("repository", repoFull).Msg("invalid GITHUB_REPOSITORY")
	}

	owner, repo := parts[0], parts[1]
	log.Info().Str("owner", owner).Str("repo", repo).Msg("Repository to scan")

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
		}

		for {
			runs, resp, err := client.Actions.ListRepositoryWorkflowRuns(ctx, owner, repo, opts)
			if err != nil {
				log.Fatal().Stack().Err(err).Msg("Failed listing workflow runs")
			}

			for _, run := range runs.WorkflowRuns {
				log.Info().Int64("run", run.GetID()).Str("commit", run.GetHeadSHA()).Str("name", run.GetName()).Msg("Check run")
				if run.GetHeadSHA() == sha && run.GetID() != currentRunID {
					status := run.GetStatus()
					log.Info().Int64("run", run.GetID()).Str("status", status).Str("name", run.GetName()).Msgf("Run")

					if status != "completed" {
						allCompleted = false
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

package github

import (
	"archive/zip"
	"bytes"
	"context"
	"io"
	"net/http"
	"time"

	"github.com/CompassSecurity/pipeleak/scanner"
	"github.com/gofri/go-github-ratelimit/v2/github_ratelimit"
	"github.com/gofri/go-github-ratelimit/v2/github_ratelimit/github_primary_ratelimit"
	"github.com/gofri/go-github-ratelimit/v2/github_ratelimit/github_secondary_ratelimit"
	"github.com/google/go-github/v69/github"
	"github.com/h2non/filetype"
	"github.com/rs/zerolog/log"
	"github.com/wandb/parallel"
)

type GitHubScanOptions struct {
	AccessToken            string
	Verbose                bool
	ConfidenceFilter       []string
	MaxScanGoRoutines      int
	TruffleHogVerification bool
	MaxWorkflows           int
	Organization           string
	Owned                  bool
	User                   string
	Public                 bool
	SearchQuery            string
	Artifacts              bool
	Context                context.Context
	Client                 *github.Client
	HttpClient             *http.Client
}

var options = GitHubScanOptions{}

func setupClient(accessToken string) *github.Client {
	rateLimiter := github_ratelimit.New(nil,
		github_primary_ratelimit.WithLimitDetectedCallback(func(ctx *github_primary_ratelimit.CallbackContext) {
			resetTime := ctx.ResetTime.Add(time.Duration(time.Second * 30))
			log.Info().Str("category", string(ctx.Category)).Time("reset", resetTime).Msg("Primary rate limit detected, will resume automatically")
			time.Sleep(time.Until(resetTime))
			log.Info().Str("category", string(ctx.Category)).Msg("Resuming")
		}),
		github_secondary_ratelimit.WithLimitDetectedCallback(func(ctx *github_secondary_ratelimit.CallbackContext) {
			resetTime := ctx.ResetTime.Add(time.Duration(time.Second * 30))
			log.Info().Time("reset", *ctx.ResetTime).Dur("totalSleep", *ctx.TotalSleepTime).Msg("Secondary rate limit detected, will resume automatically")
			time.Sleep(time.Until(resetTime))
			log.Info().Msg("Resuming")
		}),
	)

	return github.NewClient(&http.Client{Transport: rateLimiter}).WithAuthToken(accessToken)
}

func downloadWorkflowRunLog(client *github.Client, repo *github.Repository, workflowRun *github.WorkflowRun) {
	logURL, resp, err := client.Actions.GetWorkflowRunLogs(options.Context, *repo.Owner.Login, *repo.Name, *workflowRun.ID, 5)

	if resp == nil {
		log.Trace().Msg("downloadWorkflowRunLog Empty response")
		return
	}

	// already deleted, skip
	if resp.StatusCode == 410 {
		log.Debug().Str("workflowRunName", *workflowRun.Name).Msg("Skipped expired")
		return
	} else if resp.StatusCode == 404 {
		return
	}

	if err != nil {
		log.Error().Stack().Err(err).Msg("Failed getting workflow run log URL")
		return
	}

	log.Trace().Msg("Downloading run log")
	logs := downloadRunLogZIP(logURL.String())
	log.Trace().Msg("Finished downloading run log")
	findings, err := scanner.DetectHits(logs, options.MaxScanGoRoutines, options.TruffleHogVerification)
	if err != nil {
		log.Debug().Err(err).Str("workflowRun", *workflowRun.HTMLURL).Msg("Failed detecting secrets")
		return
	}

	for _, finding := range findings {
		log.Warn().Str("confidence", finding.Pattern.Pattern.Confidence).Str("ruleName", finding.Pattern.Pattern.Name).Str("value", finding.Text).Str("workflowRun", *workflowRun.HTMLURL).Msg("HIT")
	}
	log.Trace().Msg("Finished scannig run log")
}

func iterateWorkflowRuns(client *github.Client, repo *github.Repository) {
	opt := github.ListWorkflowRunsOptions{
		ListOptions: github.ListOptions{PerPage: 100},
	}
	wfCount := 0
	for {
		workflowRuns, resp, err := client.Actions.ListRepositoryWorkflowRuns(options.Context, *repo.Owner.Login, *repo.Name, &opt)

		if resp == nil {
			log.Trace().Msg("Empty response due to rate limit, resume now<")
			continue
		}

		if resp.StatusCode == 404 {
			return
		}

		if err != nil {
			log.Error().Stack().Err(err).Msg("Failed fetching workflow runs")
			return
		}

		for _, workflowRun := range workflowRuns.WorkflowRuns {
			log.Debug().Str("name", *workflowRun.DisplayTitle).Str("url", *workflowRun.HTMLURL).Msg("Workflow run")
			downloadWorkflowRunLog(client, repo, workflowRun)

			if options.Artifacts {
				listArtifacts(client, workflowRun)
			}

			wfCount = wfCount + 1
			if wfCount >= options.MaxWorkflows && options.MaxWorkflows > 0 {
				log.Debug().Str("name", *workflowRun.DisplayTitle).Str("url", *workflowRun.HTMLURL).Msg("Reached MaxWorkflow runs, skip remaining")
				return
			}
		}

		if resp.NextPage == 0 {
			break
		}
		opt.Page = resp.NextPage
	}
}

func downloadRunLogZIP(url string) []byte {
	res, err := options.HttpClient.Get(url)
	logLines := make([]byte, 0)

	if err != nil {
		return logLines
	}

	if res.StatusCode == 200 {
		body, err := io.ReadAll(res.Body)
		if err != nil {
			log.Err(err).Msg("Failed reading response log body")
			return logLines
		}

		zipReader, err := zip.NewReader(bytes.NewReader(body), int64(len(body)))
		if err != nil {
			log.Err(err).Msg("Failed creating zip reader")
			return logLines
		}

		for _, zipFile := range zipReader.File {
			log.Trace().Str("zipFile", zipFile.Name).Msg("Zip file")
			unzippedFileBytes, err := readZipFile(zipFile)
			if err != nil {
				log.Err(err).Msg("Failed reading zip file")
				continue
			}

			logLines = append(logLines, unzippedFileBytes...)
		}
	}

	return logLines
}

func readZipFile(zf *zip.File) ([]byte, error) {
	f, err := zf.Open()
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return io.ReadAll(f)
}

func listArtifacts(client *github.Client, workflowRun *github.WorkflowRun) {
	listOpt := github.ListOptions{PerPage: 100}
	for {
		artifactList, resp, err := client.Actions.ListWorkflowRunArtifacts(options.Context, *workflowRun.Repository.Owner.Login, *workflowRun.Repository.Name, *workflowRun.ID, &listOpt)
		if resp == nil {
			return
		}

		if resp.StatusCode == 404 {
			return
		}

		if err != nil {
			log.Error().Stack().Err(err).Msg("Failed fetching artifacts list")
			return
		}

		for _, artifact := range artifactList.Artifacts {
			log.Debug().Str("name", *artifact.Name).Str("url", *artifact.ArchiveDownloadURL).Msg("Scan")
			analyzeArtifact(client, workflowRun, artifact)
		}

		if resp.NextPage == 0 {
			break
		}
		listOpt.Page = resp.NextPage
	}
}

func analyzeArtifact(client *github.Client, workflowRun *github.WorkflowRun, artifact *github.Artifact) {

	url, resp, err := client.Actions.DownloadArtifact(options.Context, *workflowRun.Repository.Owner.Login, *workflowRun.Repository.Name, *artifact.ID, 5)

	if resp == nil {
		log.Trace().Msg("analyzeArtifact Empty response")
		return
	}

	// already deleted, skip
	if resp.StatusCode == 410 {
		log.Debug().Str("workflowRunName", *workflowRun.Name).Msg("Skipped expired artifact")
		return
	}

	if err != nil {
		log.Err(err).Msg("Failed getting artifact download URL")
		return
	}

	res, err := options.HttpClient.Get(url.String())

	if err != nil {
		log.Err(err).Str("workflow", url.String()).Msg("Failed downloading artifacts zip")
		return
	}

	if res.StatusCode == 200 {
		body, err := io.ReadAll(res.Body)
		if err != nil {
			log.Err(err).Msg("Failed reading response log body")
			return
		}
		zipListing, err := zip.NewReader(bytes.NewReader(body), int64(len(body)))
		if err != nil {
			log.Err(err).Str("url", url.String()).Msg("Failed creating zip reader")
			return
		}

		ctx := options.Context
		group := parallel.Limited(ctx, options.MaxScanGoRoutines)
		for _, file := range zipListing.File {
			group.Go(func(ctx context.Context) {
				fc, err := file.Open()
				if err != nil {
					log.Error().Stack().Err(err).Msg("Unable to open raw artifact zip file")
					return
				}

				content, err := io.ReadAll(fc)
				if err != nil {
					log.Error().Stack().Err(err).Msg("Unable to readAll artifact zip file")
					return
				}

				kind, _ := filetype.Match(content)
				// do not scan https://pkg.go.dev/github.com/h2non/filetype#readme-supported-types
				if kind == filetype.Unknown {
					scanner.DetectFileHits(content, *workflowRun.HTMLURL, *workflowRun.Name, file.Name, "", options.TruffleHogVerification)
				} else if filetype.IsArchive(content) {
					scanner.HandleArchiveArtifact(file.Name, content, *workflowRun.HTMLURL, *workflowRun.Name, options.TruffleHogVerification)
				}
				fc.Close()
			})
		}

		group.Wait()
	}
}

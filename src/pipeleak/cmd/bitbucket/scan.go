package bitbucket

import (
	"context"
	"net/url"
	"path"
	"strconv"
	"time"

	"github.com/CompassSecurity/pipeleak/pkg/format"
	"github.com/CompassSecurity/pipeleak/pkg/logging"
	"github.com/CompassSecurity/pipeleak/pkg/scan/logline"
	"github.com/CompassSecurity/pipeleak/pkg/scan/result"
	"github.com/CompassSecurity/pipeleak/pkg/scan/runner"
	"github.com/CompassSecurity/pipeleak/pkg/scanner"
	"github.com/h2non/filetype"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
)

type BitBucketScanOptions struct {
	Email                  string
	AccessToken            string
	Verbose                bool
	ConfidenceFilter       []string
	MaxScanGoRoutines      int
	TruffleHogVerification bool
	MaxPipelines           int
	Workspace              string
	Owned                  bool
	Public                 bool
	After                  string
	Artifacts              bool
	BitBucketURL           string
	Context                context.Context
	Client                 BitBucketApiClient
	BitBucketCookie        string
}

var options = BitBucketScanOptions{}

func NewScanCmd() *cobra.Command {
	scanCmd := &cobra.Command{
		Use:   "scan",
		Short: "Scan BitBucket Pipelines",
		Long: `Create a BitBucket scoped API token [here](https://id.atlassian.com/manage-profile/security/api-tokens) and pass it to the <code>--token</code> flag.
The <code>--email</code> flag expects your account's email address.
To scan artifacts (uses internal APIs) you need to extract the session cookie value <code>cloud.session.token</code> from [bitbucket.org](https://bitbucket.org) using your browser and supply it in the <code>--cookie</code> flag.
A note on artifacts: Bitbucket artifacts are only stored for a limited time and only for paid accounts. Free accounts might not have artifacts available at all.
		  `,
		Example: `
# Scan a workspace (find public ones here: https://bitbucket.org/repo/all/) without artifacts
pipeleak bb scan --token ATATTxxxxxx --email auser@example.com --workspace bitbucketpipelines

# Scan your owned repositories and their artifacts
pipeleak bb scan -t ATATTxxxxxx -c eyJxxxxxxxxxxx --artifacts -e auser@example.com --owned

# Scan all public repositories without their artifacts
> If using --after, the API becomes quite unreliable 👀
pipeleak bb scan --token ATATTxxxxxx --email auser@example.com --public --maxPipelines 5 --after 2025-03-01T15:00:00+00:00
		`,
		Run: Scan,
	}
	scanCmd.Flags().StringVarP(&options.AccessToken, "token", "t", "", "Bitbucket API token - https://id.atlassian.com/manage-profile/security/api-tokens")
	scanCmd.Flags().StringVarP(&options.Email, "email", "e", "", "Bitbucket Email")
	scanCmd.Flags().StringVarP(&options.BitBucketCookie, "cookie", "c", "", "Bitbucket Cookie [value of cloud.session.token on https://bitbucket.org]")
	scanCmd.Flags().StringVarP(&options.BitBucketURL, "bitbucket", "b", "https://api.bitbucket.org/2.0", "BitBucket API base URL")
	scanCmd.PersistentFlags().BoolVarP(&options.Artifacts, "artifacts", "a", false, "Scan workflow artifacts")
	scanCmd.MarkFlagsRequiredTogether("cookie", "artifacts")

	scanCmd.Flags().StringSliceVarP(&options.ConfidenceFilter, "confidence", "", []string{}, "Filter for confidence level, separate by comma if multiple. See readme for more info.")
	scanCmd.PersistentFlags().IntVarP(&options.MaxScanGoRoutines, "threads", "", 4, "Nr of threads used to scan")
	scanCmd.PersistentFlags().BoolVarP(&options.TruffleHogVerification, "truffleHogVerification", "", true, "Enable the TruffleHog credential verification, will actively test the found credentials and only report those. Disable with --truffleHogVerification=false")
	scanCmd.PersistentFlags().IntVarP(&options.MaxPipelines, "maxPipelines", "", -1, "Max. number of pipelines to scan per repository")

	scanCmd.Flags().StringVarP(&options.Workspace, "workspace", "w", "", "Workspace name to scan")
	scanCmd.PersistentFlags().BoolVarP(&options.Owned, "owned", "o", false, "Scan user onwed projects only")
	scanCmd.PersistentFlags().BoolVarP(&options.Public, "public", "p", false, "Scan all public repositories")
	scanCmd.PersistentFlags().StringVarP(&options.After, "after", "", "", "Filter public repos by a given date in ISO 8601 format: 2025-04-02T15:00:00+02:00 ")

	scanCmd.PersistentFlags().BoolVarP(&options.Verbose, "verbose", "v", false, "Verbose logging")

	return scanCmd
}

func Scan(cmd *cobra.Command, args []string) {
	if options.AccessToken != "" && options.Email == "" {
		log.Fatal().Msg("When using --token you must also provide --email")
	}

	logging.SetLogLevel(options.Verbose)
	go logging.ShortcutListeners(scanStatus)

	runner.InitScanner(options.ConfidenceFilter)

	options.Context = context.Background()
	options.Client = NewClient(options.Email, options.AccessToken, options.BitBucketCookie, options.BitBucketURL)

	if len(options.BitBucketCookie) > 0 {
		options.Client.GetuserInfo()
	}

	if options.Public {
		log.Info().Msg("Scanning public repos")
		scanPublic(options.Client, options.After)
	} else if options.Owned {
		log.Info().Msg("Scanning current user owned workspaces")
		scanOwned(options.Client)
	} else if options.Workspace != "" {
		log.Info().Str("name", options.Workspace).Msg("Scanning a workspace")
		scanWorkspace(options.Client, options.Workspace)
	} else {
		log.Error().Msg("Specify a scan mode --public, --owned, --workspace")
	}

	log.Info().Msg("Scan Finished, Bye Bye 🏳️‍🌈🔥")
}

func scanOwned(client BitBucketApiClient) {
	next := ""
	for {
		workspaces, nextUrl, _, err := client.ListOwnedWorkspaces(next)
		if err != nil {
			log.Error().Err(err).Msg("Failed fetching workspaces")
		}

		for _, workspace := range workspaces {
			log.Trace().Str("name", workspace.Name).Msg("Workspace")
			listWorkspaceRepos(client, workspace.Slug)
		}

		if nextUrl == "" {
			break
		}
		next = nextUrl
	}
}

func scanWorkspace(client BitBucketApiClient, workspace string) {
	next := ""
	for {
		repos, nextUrl, _, err := client.ListWorkspaceRepositoires(next, workspace)
		if err != nil {
			log.Error().Err(err).Msg("Failed fetching workspace")
		}

		for _, repo := range repos {
			log.Debug().Str("url", repo.Links.HTML.Href).Time("created", repo.CreatedOn).Time("updated", repo.UpdatedOn).Msg("Repo")
			listRepoPipelines(client, workspace, repo.Name)
		}

		if nextUrl == "" {
			break
		}
		next = nextUrl
	}
}

func scanPublic(client BitBucketApiClient, after string) {
	afterTime := time.Time{}
	if after != "" {
		afterTime = format.ParseISO8601(after)
	}
	log.Info().Time("after", afterTime).Msg("Scanning repos after")
	next := ""
	for {
		repos, nextUrl, _, err := client.ListPublicRepositories(next, afterTime)
		if err != nil {
			log.Error().Err(err).Msg("Failed fetching public repositories")
		}

		for _, repo := range repos {
			log.Debug().Str("url", repo.Links.HTML.Href).Time("created", repo.CreatedOn).Time("updated", repo.UpdatedOn).Msg("Repo")
			listRepoPipelines(client, repo.Workspace.Name, repo.Name)
		}

		if nextUrl == "" {
			break
		}
		next = nextUrl
	}
}

func listWorkspaceRepos(client BitBucketApiClient, workspaceSlug string) {
	next := ""
	for {
		repos, nextUrl, _, err := client.ListWorkspaceRepositoires(next, workspaceSlug)
		if err != nil {
			log.Error().Err(err).Msg("Failed fetching workspace repos")
		}

		for _, repo := range repos {
			log.Debug().Str("url", repo.Links.HTML.Href).Time("created", repo.CreatedOn).Time("updated", repo.UpdatedOn).Msg("Repo")
			listRepoPipelines(client, workspaceSlug, repo.Name)
		}
		if nextUrl == "" {
			break
		}
		next = nextUrl
	}
}

func listRepoPipelines(client BitBucketApiClient, workspaceSlug string, repoSlug string) {
	pipelineCount := 0
	next := ""
	for {
		pipelines, nextUrl, _, err := client.ListRepositoryPipelines(next, workspaceSlug, repoSlug)
		if err != nil {
			log.Error().Err(err).Msg("Failed fetching repo pipelines")
		}

		for _, pipeline := range pipelines {
			log.Trace().Int("buildNr", pipeline.BuildNumber).Str("uuid", pipeline.UUID).Msg("Pipeline")
			listPipelineSteps(client, workspaceSlug, repoSlug, pipeline.UUID)
			if options.Artifacts {
				log.Trace().Int("buildNr", pipeline.BuildNumber).Str("uuid", pipeline.UUID).Msg("Fetch pipeline artifacts")
				listArtifacts(client, workspaceSlug, repoSlug, pipeline.BuildNumber)
				listDownloadArtifacts(client, workspaceSlug, repoSlug)
			}

			pipelineCount = pipelineCount + 1
			if pipelineCount >= options.MaxPipelines && options.MaxPipelines > 0 {
				log.Debug().Str("workspace", workspaceSlug).Str("repo", repoSlug).Msg("Reached max pipelines runs, skip remaining")
				return
			}
		}

		if nextUrl == "" {
			break
		}
		next = nextUrl
	}
}

func listArtifacts(client BitBucketApiClient, workspaceSlug string, repoSlug string, buildId int) {
	next := ""
	for {
		artifacts, nextUrl, _, err := client.ListPipelineArtifacts(next, workspaceSlug, repoSlug, buildId)
		if err != nil {
			log.Error().Err(err).Msg("Failed fetching pipeline download artifacts")
		}

		for _, art := range artifacts {
			log.Trace().Str("name", art.Name).Msg("Pipeline Artifact")
			artifactBytes := client.GetPipelineArtifact(workspaceSlug, repoSlug, buildId, art.UUID)

			// Use artifact processor if it's an archive, otherwise use HandleArchiveArtifact
			if filetype.IsArchive(artifactBytes) {
				scanner.HandleArchiveArtifact(art.Name, artifactBytes, buildWebArtifactUrl(workspaceSlug, repoSlug, buildId, art.StepUUID), "Build "+strconv.Itoa(buildId), options.TruffleHogVerification)
			}
		}

		if nextUrl == "" {
			break
		}
		next = nextUrl
	}
}

func listDownloadArtifacts(client BitBucketApiClient, workspaceSlug string, repoSlug string) {
	next := ""
	for {
		downloadArtifacts, nextUrl, _, err := client.ListDownloadArtifacts(next, workspaceSlug, repoSlug)
		if err != nil {
			log.Error().Err(err).Msg("Failed fetching pipeline download artifacts")
		}

		for _, artifact := range downloadArtifacts {
			log.Trace().Str("name", artifact.Name).Str("creator", artifact.User.DisplayName).Msg("Download Artifact")
			getDownloadArtifact(client, artifact.Links.Self.Href, constructDownloadArtifactWebUrl(workspaceSlug, repoSlug, artifact.Name), artifact.Name)
		}

		if nextUrl == "" {
			break
		}
		next = nextUrl
	}
}

func listPipelineSteps(client BitBucketApiClient, workspaceSlug string, repoSlug string, pipelineUuid string) {
	next := ""
	for {
		steps, nextUrl, _, err := client.ListPipelineSteps(next, workspaceSlug, repoSlug, pipelineUuid)
		if err != nil {
			log.Error().Err(err).Msg("Failed fetching pipeline steps")
		}

		for _, step := range steps {
			log.Trace().Str("step", step.UUID).Msg("Step")
			getSteplog(client, workspaceSlug, repoSlug, pipelineUuid, step.UUID)
		}

		if nextUrl == "" {
			break
		}
		next = nextUrl
	}
}

func constructDownloadArtifactWebUrl(workspaceSlug string, repoSlug string, artifactName string) string {
	u, err := url.Parse("https://bitbucket.org/")
	if err != nil {
		log.Fatal().Err(err).Msg("Unable to parse ListDownloadArtifacts url")
	}
	u.Path = path.Join(u.Path, workspaceSlug, repoSlug, "downloads", artifactName)
	return u.String()
}

func getSteplog(client BitBucketApiClient, workspaceSlug string, repoSlug string, pipelineUuid string, stepUUID string) {
	logBytes, _, err := client.GetStepLog(workspaceSlug, repoSlug, pipelineUuid, stepUUID)
	if err != nil {
		log.Error().Err(err).Msg("Failed fetching pipeline steps")
	}

	// Use the new logline processor
	logResult, err := logline.ProcessLogs(logBytes, logline.ProcessOptions{
		MaxGoRoutines:     options.MaxScanGoRoutines,
		VerifyCredentials: options.TruffleHogVerification,
	})
	if err != nil {
		log.Debug().Err(err).Str("stepUUid", stepUUID).Msg("Failed detecting secrets")
		return
	}

	// Use the new result reporter
	runURL := "https://bitbucket.org/" + workspaceSlug + "/" + repoSlug + "/pipelines/results/" + pipelineUuid + "/steps/" + stepUUID
	result.ReportFindings(logResult.Findings, result.ReportOptions{
		LocationURL: runURL,
	})
}

func getDownloadArtifact(client BitBucketApiClient, downloadUrl string, webUrl string, filename string) {
	fileBytes := client.GetDownloadArtifact(downloadUrl)
	if len(fileBytes) == 0 {
		return
	}

	if filetype.IsArchive(fileBytes) {
		scanner.HandleArchiveArtifact(filename, fileBytes, webUrl, "Download Artifact", options.TruffleHogVerification)
	} else {
		scanner.DetectFileHits(fileBytes, webUrl, "Download Artifact", filename, "", options.TruffleHogVerification)
	}
}

func scanStatus() *zerolog.Event {
	return log.Info().Str("debug", "nothing to show ✔️")
}

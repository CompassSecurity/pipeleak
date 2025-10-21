package gitea

import (
	"archive/zip"
	"bytes"
	"context"
	"fmt"
	"io"

	"code.gitea.io/sdk/gitea"
	"github.com/CompassSecurity/pipeleak/scanner"
	"github.com/h2non/filetype"
	"github.com/rs/zerolog/log"
	"github.com/wandb/parallel"
)

func scanLogs(logBytes []byte, repo *gitea.Repository, run ActionWorkflowRun, jobID int64, jobName string) {
	if repo == nil {
		log.Error().Msg("Cannot scan logs: repository is nil")
		return
	}

	findings, err := scanner.DetectHits(logBytes, scanOptions.MaxScanGoRoutines, scanOptions.TruffleHogVerification)
	if err != nil {
		log.Debug().Err(err).
			Str("repo", repo.FullName).
			Int64("run_id", run.ID).
			Int64("job_id", jobID).
			Msg("Failed detecting secrets in logs")
		return
	}

	for _, finding := range findings {
		logFinding(finding, repo.FullName, run.ID, jobID, jobName, run.HTMLURL)
	}
}

func logFinding(finding scanner.Finding, repoFullName string, runID, jobID int64, jobName, url string) {
	event := log.Warn().
		Str("confidence", finding.Pattern.Pattern.Confidence).
		Str("ruleName", finding.Pattern.Pattern.Name).
		Str("value", finding.Text).
		Str("repo", repoFullName).
		Int64("run_id", runID).
		Str("url", url)

	if jobID > 0 {
		event = event.Int64("job_id", jobID)
	}

	if jobName != "" {
		event = event.Str("job_name", jobName)
	}

	event.Msg("HIT")
}

func processZipArtifact(zipBytes []byte, repo *gitea.Repository, run ActionWorkflowRun, artifactName string) {
	if repo == nil {
		log.Error().Msg("Cannot process artifact: repository is nil")
		return
	}

	zipReader, err := zip.NewReader(bytes.NewReader(zipBytes), int64(len(zipBytes)))
	if err != nil {
		log.Debug().
			Str("repo", repo.FullName).
			Int64("run_id", run.ID).
			Str("artifact", artifactName).
			Msg("Artifact is not a zip, scanning directly")
		scanArtifactContent(zipBytes, repo, run, artifactName, "")
		return
	}

	ctx := scanOptions.Context
	group := parallel.Limited(ctx, scanOptions.MaxScanGoRoutines)

	for _, file := range zipReader.File {
		f := file
		group.Go(func(ctx context.Context) {
			fc, err := f.Open()
			if err != nil {
				log.Debug().Err(err).Str("file", f.Name).Msg("Unable to open file in artifact zip")
				return
			}
			defer fc.Close()

			content, err := io.ReadAll(fc)
			if err != nil {
				log.Debug().Err(err).Str("file", f.Name).Msg("Unable to read file in artifact zip")
				return
			}

			scanArtifactContent(content, repo, run, artifactName, f.Name)
		})
	}

	group.Wait()
}

func determineFileAction(content []byte, displayName string) (action string, fileType string) {
	kind, _ := filetype.Match(content)

	if filetype.IsArchive(content) {
		return "archive", kind.MIME.Value
	}

	if kind != filetype.Unknown {
		return "skip", kind.MIME.Value
	}

	return "scan", kind.MIME.Value
}

func scanArtifactContent(content []byte, repo *gitea.Repository, run ActionWorkflowRun, artifactName string, fileName string) {
	displayName := artifactName
	if fileName != "" {
		displayName = fmt.Sprintf("%s/%s", artifactName, fileName)
	}

	action, fileType := determineFileAction(content, displayName)

	switch action {
	case "archive":
		scanner.HandleArchiveArtifact(displayName, content, run.HTMLURL, run.Name, scanOptions.TruffleHogVerification)
	case "skip":
		log.Trace().
			Str("file", displayName).
			Str("type", fileType).
			Msg("Unknown file type, scanning as text")
		scanner.DetectFileHits(content, run.HTMLURL, run.Name, displayName, repo.FullName, scanOptions.TruffleHogVerification)
	case "scan":
		log.Debug().
			Str("file", displayName).
			Str("type", fileType).
			Msg("Not an archive file type, scanning as text")
		scanner.DetectFileHits(content, run.HTMLURL, run.Name, displayName, repo.FullName, scanOptions.TruffleHogVerification)
	}
}

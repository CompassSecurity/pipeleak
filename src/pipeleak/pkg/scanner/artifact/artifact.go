package artifact

import (
	"os"
	"path"
	"strings"

	"github.com/CompassSecurity/pipeleak/pkg/format"
	"github.com/CompassSecurity/pipeleak/pkg/logging"
	"github.com/CompassSecurity/pipeleak/pkg/scanner/engine"
	"github.com/CompassSecurity/pipeleak/pkg/scanner/types"
	"github.com/h2non/filetype"
	"github.com/rs/zerolog/log"
	"golift.io/xtractr"
)

var skippableDirectoryNames = []string{"node_modules", ".yarn", ".yarn-cache", ".npm", "venv", "vendor", ".go/pkg/mod/"}

func DetectFileHits(content []byte, jobWebUrl string, jobName string, fileName string, archiveName string, enableTruffleHogVerification bool) {
	findings, err := engine.DetectHits(content, 1, enableTruffleHogVerification)
	if err != nil {
		log.Debug().Err(err).Str("job", jobWebUrl).Msg("Failed detecting secrets")
		return
	}
	for _, finding := range findings {
		baseLog := logging.Hit().Str("confidence", finding.Pattern.Pattern.Confidence).Str("ruleName", finding.Pattern.Pattern.Name).Str("value", finding.Text).Str("url", jobWebUrl).Str("jobName", jobName).Str("file", fileName)
		if len(archiveName) > 0 {
			baseLog.Str("archive", archiveName).Msg("HIT Artifact (in archive)")
		} else {
			baseLog.Msg("HIT Artifact")
		}
	}
}

func HandleArchiveArtifact(archivefileName string, content []byte, jobWebUrl string, jobName string, enableTruffleHogVerification bool) {
	HandleArchiveArtifactWithDepth(archivefileName, content, jobWebUrl, jobName, enableTruffleHogVerification, 1)
}

func HandleArchiveArtifactWithDepth(archivefileName string, content []byte, jobWebUrl string, jobName string, enableTruffleHogVerification bool, depth int) {
	if depth > 10 {
		log.Debug().Str("file", archivefileName).Int("recursionDepth", depth).Msg("Max archive recursion depth reached, skipping further extraction")
		return
	}

	for _, skipKeyword := range skippableDirectoryNames {
		if strings.Contains(archivefileName, skipKeyword) {
			log.Debug().Str("file", archivefileName).Str("keyword", skipKeyword).Msg("Skipped archive due to blocklist entry")
			return
		}
	}

	fileType, err := filetype.Get(content)
	if err != nil {
		log.Error().Stack().Err(err).Msg("Cannot determine file type")
		return
	}

	tmpArchiveFile, err := os.CreateTemp("", "pipeleak-artifact-archive-*."+fileType.Extension)
	if err != nil {
		log.Error().Stack().Err(err).Msg("Cannot create artifact archive temp file")
		return
	}

	err = os.WriteFile(tmpArchiveFile.Name(), content, format.FileUserReadWrite)
	if err != nil {
		log.Error().Stack().Err(err).Msg("Failed writing archive to disk")
		return
	}
	defer func() { _ = os.Remove(tmpArchiveFile.Name()) }()

	tmpArchiveFilesDirectory, err := os.MkdirTemp("", "pipeleak-artifact-archive-out-")
	if err != nil {
		log.Error().Stack().Err(err).Msg("Cannot create artifact archive temp directory")
		return
	}
	defer func() { _ = os.RemoveAll(tmpArchiveFilesDirectory) }()

	x := &xtractr.XFile{
		FilePath:  tmpArchiveFile.Name(),
		OutputDir: tmpArchiveFilesDirectory,
		FileMode:  0o600,
		DirMode:   0o700,
	}

	_, files, _, err := xtractr.ExtractFile(x)
	if err != nil || files == nil {
		log.Debug().Str("err", err.Error()).Msg("Unable to handle archive in artifacts")
		return
	}

	for _, fPath := range files {
		if !format.IsDirectory(fPath) {
			// #nosec G304 - Reading extracted artifact files from temp directory, path controlled by xtractr library
			fileBytes, err := os.ReadFile(fPath)
			if err != nil {
				log.Debug().Str("file", fPath).Stack().Str("err", err.Error()).Msg("Cannot read temp artifact archive file content")
				continue
			}

			currentFileName := path.Base(fPath)

			if filetype.IsArchive(fileBytes) {
				log.Trace().Str("fileName", currentFileName).Str("parentArchive", archivefileName).Int("depth", depth).Msg("Detected nested archive, recursing")
				HandleArchiveArtifactWithDepth(currentFileName, fileBytes, jobWebUrl, jobName, enableTruffleHogVerification, depth+1)
				continue
			}

			kind, _ := filetype.Match(fileBytes)
			if kind == filetype.Unknown {
				DetectFileHits(fileBytes, jobWebUrl, jobName, currentFileName, archivefileName, enableTruffleHogVerification)
			}
		}
	}
}

func ReportFinding(finding types.Finding, url string, jobName string, fileName string, archiveName string) {
	baseLog := logging.Hit().Str("confidence", finding.Pattern.Pattern.Confidence).Str("ruleName", finding.Pattern.Pattern.Name).Str("value", finding.Text).Str("url", url).Str("jobName", jobName).Str("file", fileName)
	if len(archiveName) > 0 {
		baseLog.Str("archive", archiveName).Msg("HIT Artifact (in archive)")
	} else {
		baseLog.Msg("HIT Artifact")
	}
}

package scanner

import (
	"context"
	"errors"
	"io"
	"os"
	"path"
	"regexp"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/CompassSecurity/pipeleak/helper"
	"github.com/acarl005/stripansi"
	"github.com/h2non/filetype"
	"github.com/rs/zerolog/log"
	"github.com/rxwycdh/rxhash"
	"github.com/trufflesecurity/trufflehog/v3/pkg/detectors"
	"github.com/trufflesecurity/trufflehog/v3/pkg/engine/defaults"
	"github.com/wandb/parallel"
	"golift.io/xtractr"
	"gopkg.in/yaml.v3"
)

var ruleFile = "https://raw.githubusercontent.com/mazen160/secrets-patterns-db/master/db/rules-stable.yml"

var ruleFileName = "rules.yml"

type SecretsPatterns struct {
	Patterns []PatternElement `json:"patterns"`
}

type PatternElement struct {
	Pattern PatternPattern `json:"pattern"`
}

type PatternPattern struct {
	Name       string `json:"name"`
	Regex      string `json:"regex"`
	Confidence string `json:"confidence"`
}

type Finding struct {
	Pattern PatternElement
	Text    string
}

// hold patterns in memory during runtime
var secretsPatterns = SecretsPatterns{}

// keep a nr findings in memory to check for duplicates
// prevent printing the same finding e.g. 10 times just because the same job was run several times
var findingsDeduplicationList []string
var deduplicationMutex sync.Mutex
var truffelhogRules []detectors.Detector

func DownloadRules() {
	if _, err := os.Stat(ruleFileName); errors.Is(err, os.ErrNotExist) {
		log.Debug().Msg("No rules file found, downloading")
		err := downloadFile(ruleFile, ruleFileName)
		if err != nil {
			log.Fatal().Stack().Err(err).Msg("Failed downloading rules file")
			os.Exit(1)
		}
	}
}

func downloadFile(url string, filepath string) error {
	out, err := os.Create(filepath)
	if err != nil {
		return err
	}
	defer func() { _ = out.Close() }()

	client := helper.GetPipeleakHTTPClient()
	resp, err := client.Get(url)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()

	_, err = io.Copy(out, resp.Body)
	if err != nil {
		return err
	}

	return nil
}

func InitRules(confidenceFilter []string) {
	DownloadRules()

	if len(secretsPatterns.Patterns) == 0 {
		log.Debug().Msg("Loading rules.yml from filesystem")
		yamlFile, err := os.ReadFile(ruleFileName)
		if err != nil {
			log.Fatal().Stack().Err(err).Msg("Failed opening rules file")
		}
		err = yaml.Unmarshal(yamlFile, &secretsPatterns)
		if err != nil {
			log.Fatal().Stack().Err(err).Msg("Failed unmarshalling rules file")
		}

		patterns := AppendPipeleakRules(secretsPatterns.Patterns)

		if len(confidenceFilter) > 0 {
			log.Debug().Str("filter", strings.Join(confidenceFilter, ",")).Msg("Applying confidence filter")
			filterdPatterns := []PatternElement{}
			for _, pattern := range patterns {
				if slices.Contains(confidenceFilter, pattern.Pattern.Confidence) {
					filterdPatterns = append(filterdPatterns, pattern)
				}
			}
			secretsPatterns.Patterns = filterdPatterns

			totalRules := len(secretsPatterns.Patterns)
			if totalRules == 0 {
				log.Info().Int("count", totalRules).Msg("Your confidence filter removed all rules, are you sure? TruffleHog Rules will still detect secrets. This equals --confidence high-verified")
			}

			log.Debug().Int("count", totalRules).Msg("Loaded filtered rules")
		} else {
			secretsPatterns.Patterns = patterns
			log.Debug().Int("count", len(secretsPatterns.Patterns)).Msg("Loaded rules.yml rules")
		}
	}

	truffelhogRules = defaults.DefaultDetectors()
	if len(truffelhogRules) < 1 {
		log.Fatal().Msg("No trufflehog rules have been loaded, this is a bug")
	} else {
		log.Debug().Int("count", len(truffelhogRules)).Msg("Loaded TruffleHog rules")
	}

}

// manually maintained builtin pipeleak rules
func AppendPipeleakRules(rules []PatternElement) []PatternElement {
	customRules := []PatternElement{}
	customRules = append(customRules, PatternElement{Pattern: PatternPattern{Name: "Gitlab - Predefined Environment Variable", Regex: `(GITLAB_USER_ID|KUBECONFIG|CI_SERVER_TLS_KEY_FILE|CI_REPOSITORY_URL|CI_REGISTRY_PASSWORD|DOCKER_AUTH_CONFIG)=.*`, Confidence: "medium"}})
	return slices.Concat(rules, customRules)
}

type DetectionResult struct {
	Findings []Finding
	Error    error
}

func DetectHits(text []byte, maxThreads int, enableTruffleHogVerification bool) ([]Finding, error) {
	result := make(chan DetectionResult, 1)
	go func() {
		result <- DetectHitsWithTimeout(text, maxThreads, enableTruffleHogVerification)
	}()
	select {
	// Hit detection timeout
	case <-time.After(60 * time.Second):
		return nil, errors.New("hit detection timed out")
	case result := <-result:
		return result.Findings, result.Error
	}
}

func DetectHitsWithTimeout(text []byte, maxThreads int, enableTruffleHogVerification bool) DetectionResult {
	ctx := context.Background()
	group := parallel.Collect[[]Finding](parallel.Limited(ctx, maxThreads))

	for _, pattern := range secretsPatterns.Patterns {
		group.Go(func(ctx context.Context) ([]Finding, error) {
			findingsYml := []Finding{}
			m, err := regexp.Compile(pattern.Pattern.Regex)
			if err != nil {
				log.Trace().Err(err).Str("name", pattern.Pattern.Name).Str("regex", pattern.Pattern.Regex).Msg("Failed compiling regex expression")
				return findingsYml, nil
			}

			hits := m.FindAllIndex(text, -1)

			for _, hit := range hits {
				// truncate output to max 1024 chars for output readability
				hitStr := extractHitWithSurroundingText(text, hit, 50)
				hitStr = cleanHitLine(hitStr)
				if len(hitStr) > 1024 {
					hitStr = hitStr[0:1024]
				}

				if hitStr != "" {
					findingsYml = append(findingsYml, Finding{Pattern: pattern, Text: hitStr})
				}
			}

			return findingsYml, nil
		})
	}

	resultsYml, err := group.Wait()
	if err != nil {
		log.Error().Stack().Err(err).Msg("Failed waiting for parallel hit detection")
	}

	findingsCombined := slices.Concat(resultsYml...)

	trGroup := parallel.Collect[[]Finding](parallel.Limited(ctx, maxThreads))
	for _, detector := range defaults.DefaultDetectors() {
		trGroup.Go(func(ctx context.Context) ([]Finding, error) {
			findingsTr := []Finding{}
			trHits, err := detector.FromData(ctx, enableTruffleHogVerification, text)
			if err != nil {
				log.Error().Msg("Truffelhog Detector Failed " + err.Error())
				return []Finding{}, err
			}

			for _, result := range trHits {
				secret := result.Raw
				if len(result.RawV2) > 0 {
					secret = result.RawV2
				}
				finding := Finding{Pattern: PatternElement{Pattern: PatternPattern{Name: result.DetectorType.String(), Confidence: "high-verified"}}, Text: string(secret)}

				// if trufflehog verification is enalbed ONLY verified rules are reported
				if result.Verified {
					findingsTr = append(findingsTr, finding)
				}

				// if trufflehog verification is disabled all rules are reported
				if !enableTruffleHogVerification {
					// trufflehog itself does not have confidence information
					finding.Pattern.Pattern.Confidence = "trufflehog-unverified"
					findingsTr = append(findingsTr, finding)
				}
			}
			return findingsTr, nil
		})
	}

	resultsTr, err := trGroup.Wait()
	if err != nil {
		log.Error().Stack().Err(err).Msg("Failed waiting for trufflehog parallel hit detection")
	}

	findingsTr := slices.Concat(resultsTr...)
	totalFindings := slices.Concat(findingsCombined, findingsTr)
	return DetectionResult{Findings: deduplicateFindings(totalFindings), Error: nil}
}

func deduplicateFindings(totalFindings []Finding) []Finding {
	dedupedFindings := []Finding{}
	for _, finding := range totalFindings {
		hash, _ := rxhash.HashStruct(finding)
		deduplicationMutex.Lock()
		if !slices.Contains(findingsDeduplicationList, hash) {
			dedupedFindings = append(dedupedFindings, finding)
			findingsDeduplicationList = append(findingsDeduplicationList, hash)
		}

		// keep the last 500 findings and check dupes against this list.
		if len(findingsDeduplicationList) > 500 {
			findingsDeduplicationList[0] = ""
			findingsDeduplicationList = findingsDeduplicationList[1:]
		}
		deduplicationMutex.Unlock()
	}

	return dedupedFindings
}

func DetectFileHits(content []byte, jobWebUrl string, jobName string, fileName string, archiveName string, enableTruffleHogVerification bool) {
	// 1 goroutine to prevent maxThreads^2 which trashes memory
	findings, err := DetectHits(content, 1, enableTruffleHogVerification)
	if err != nil {
		log.Debug().Err(err).Str("job", jobWebUrl).Msg("Failed detecting secrets")
		return
	}
	for _, finding := range findings {
		baseLog := log.Warn().Str("confidence", finding.Pattern.Pattern.Confidence).Str("ruleName", finding.Pattern.Pattern.Name).Str("value", finding.Text).Str("url", jobWebUrl).Str("jobName", jobName).Str("file", fileName)
		if len(archiveName) > 0 {
			baseLog.Str("archive", archiveName).Msg("HIT Artifact (in archive)")
		} else {
			baseLog.Msg("HIT Artifact")
		}
	}
}

func extractHitWithSurroundingText(text []byte, hitIndex []int, additionalBytes int) string {
	startIndex := hitIndex[0]
	endIndex := hitIndex[1]

	extendedStartIndex := startIndex - additionalBytes
	if extendedStartIndex < 0 {
		startIndex = 0
	} else {
		startIndex = extendedStartIndex
	}

	extendedEndIndex := endIndex + additionalBytes
	if extendedEndIndex > len(text) {
		endIndex = len(text)
	} else {
		endIndex = extendedEndIndex
	}

	return string(text[startIndex:endIndex])
}

func cleanHitLine(text string) string {
	text = strings.ReplaceAll(text, "\n", " ")
	return stripansi.Strip(text)
}

// https://docs.gitlab.com/ee/ci/caching/#common-use-cases-for-caches
var skippableDirectoryNames = []string{"node_modules", ".yarn", ".yarn-cache", ".npm", "venv", "vendor", ".go/pkg/mod/"}

func HandleArchiveArtifact(archivefileName string, content []byte, jobWebUrl string, jobName string, enableTruffleHogVerification bool) {
	HandleArchiveArtifactWithDepth(archivefileName, content, jobWebUrl, jobName, enableTruffleHogVerification, 1)
}

func HandleArchiveArtifactWithDepth(archivefileName string, content []byte, jobWebUrl string, jobName string, enableTruffleHogVerification bool, depth int) {
	// Prevent infinite recursion in case of nested archives, make configurable when needed.
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

	err = os.WriteFile(tmpArchiveFile.Name(), content, 0666)
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
		if !helper.IsDirectory(fPath) {
			fileBytes, err := os.ReadFile(fPath)
			if err != nil {
				log.Debug().Str("file", fPath).Stack().Str("err", err.Error()).Msg("Cannot read temp artifact archive file content")
			}

			// recursively extract archives
			if filetype.IsArchive(fileBytes) {
				log.Trace().Str("fileName", archivefileName).Msg("Detected archive, recursing")
				HandleArchiveArtifactWithDepth(archivefileName, fileBytes, jobWebUrl, jobName, enableTruffleHogVerification, depth+1)
			}

			kind, _ := filetype.Match(fileBytes)
			if kind == filetype.Unknown {
				DetectFileHits(fileBytes, jobWebUrl, jobName, path.Base(fPath), archivefileName, enableTruffleHogVerification)
			}
		}
	}
}

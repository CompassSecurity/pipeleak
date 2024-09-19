package scanner

import (
	"context"
	"errors"
	"io"
	"net/http"
	"os"
	"regexp"
	"slices"
	"strings"

	"github.com/acarl005/stripansi"
	"github.com/rs/zerolog/log"
	"github.com/trufflesecurity/trufflehog/v3/pkg/engine"
	"github.com/wandb/parallel"
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
	defer out.Close()

	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	_, err = io.Copy(out, resp.Body)
	if err != nil {
		return err
	}

	return nil
}

func GetRules(confidenceFilter []string) []PatternElement {
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
				log.Warn().Int("count", totalRules).Msg("Your confidence filter removed all rules, are you sure? TruffleHog Rules will still detect secrets")
			}

			log.Debug().Int("count", totalRules).Msg("Loaded filtered rules")
		} else {
			secretsPatterns.Patterns = patterns
			log.Debug().Int("count", len(secretsPatterns.Patterns)).Msg("Loaded rules")
		}
	}

	return secretsPatterns.Patterns
}

// manually maintained builtin pipeleak rules
func AppendPipeleakRules(rules []PatternElement) []PatternElement {
	customRules := []PatternElement{}
	customRules = append(customRules, PatternElement{Pattern: PatternPattern{Name: "Gitlab - Predefined Environment Variable", Regex: `(GITLAB_USER_ID|KUBECONFIG|CI_SERVER_TLS_KEY_FILE|CI_REPOSITORY_URL|CI_REGISTRY_PASSWORD|DOCKER_AUTH_CONFIG)=.*`, Confidence: "medium"}})
	customRules = append(customRules, PatternElement{Pattern: PatternPattern{Name: "Docker Registry Auth JSON", Regex: `{[\S\s]*"auths":.*"(?:[A-Za-z0-9+\/]{4})*(?:[A-Za-z0-9+\/]{4}|[A-Za-z0-9+\/]{3}=|[A-Za-z0-9+\/]{2}={2})`, Confidence: "medium"}})

	return slices.Concat(rules, customRules)
}

func DetectHits(text []byte) []Finding {

	ctx := context.Background()
	group := parallel.Collect[[]Finding](parallel.Unlimited(ctx))

	for _, pattern := range GetRules(nil) {
		group.Go(func(ctx context.Context) ([]Finding, error) {
			findingsYml := []Finding{}
			m := regexp.MustCompile(pattern.Pattern.Regex)
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

	trGroup := parallel.Collect[[]Finding](parallel.Unlimited(ctx))
	for _, detector := range engine.DefaultDetectors() {
		trGroup.Go(func(ctx context.Context) ([]Finding, error) {
			findingsTr := []Finding{}
			trHits, err := detector.FromData(ctx, true, text)
			if err != nil {
				log.Error().Msg("Truffelhog Detector Failed " + err.Error())
				return []Finding{}, err
			}

			for _, result := range trHits {
				// only report verified
				if result.Verified {
					secret := result.Raw
					if len(result.RawV2) > 0 {
						secret = result.Raw
					}

					findingsTr = append(findingsTr, Finding{Pattern: PatternElement{Pattern: PatternPattern{Name: result.DetectorType.String(), Confidence: "high-verified"}}, Text: string(secret)})
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
	return slices.Concat(findingsCombined, findingsTr)
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

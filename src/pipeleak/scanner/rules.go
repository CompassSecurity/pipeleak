package scanner

import (
	"errors"
	"io"
	"net/http"
	"os"
	"regexp"

	"github.com/rs/zerolog/log"
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

func DownloadRules() {
	if _, err := os.Stat(ruleFileName); errors.Is(err, os.ErrNotExist) {
		log.Debug().Msg("No rules file found, downloading")
		err := downloadFile(ruleFile, ruleFileName)
		if err != nil {
			log.Fatal().Msg("Failed downloading rules file: " + err.Error())
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

func GetRules() []PatternElement {
	// download rules if needed implicitly
	DownloadRules()

	secretsPatterns := SecretsPatterns{}

	yamlFile, err := os.ReadFile(ruleFileName)
	if err != nil {
		log.Fatal().Msg("Failed opening rules file: " + err.Error())
	}
	err = yaml.Unmarshal(yamlFile, &secretsPatterns)
	if err != nil {
		log.Fatal().Msg("Failed unmarshalling rules file: " + err.Error())
	}

	return secretsPatterns.Patterns
}

func DetectHits(target string) []Finding {
	findings := []Finding{}
	for _, pattern := range GetRules() {
		m := regexp.MustCompile(pattern.Pattern.Regex)
		res := m.FindString(target)

		if res != "" {
			findings = append(findings, Finding{Pattern: pattern, Text: res})
		}
	}

	return findings
}

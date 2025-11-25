// Package nist provides functionality to fetch vulnerability data from the NIST NVD API.
package nist

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/CompassSecurity/pipeleak/pkg/httpclient"
	"github.com/hashicorp/go-retryablehttp"
	"github.com/rs/zerolog/log"
)

const resultsPerPage = 100

type nvdResponse struct {
	ResultsPerPage  int               `json:"resultsPerPage"`
	StartIndex      int               `json:"startIndex"`
	TotalResults    int               `json:"totalResults"`
	Format          string            `json:"format"`
	Version         string            `json:"version"`
	Timestamp       string            `json:"timestamp"`
	Vulnerabilities []json.RawMessage `json:"vulnerabilities"`
}

// FetchVulns retrieves all CVE vulnerabilities for a specific GitLab version and edition from the NIST NVD API.
// It automatically handles pagination if the total results exceed the page size.
func FetchVulns(version string, enterprise bool) (string, error) {
	client := httpclient.GetPipeleakHTTPClient("", nil, nil)
	baseURL := "https://services.nvd.nist.gov/rest/json/cves/2.0"
	return fetchVulnsWithClient(client, baseURL, version, enterprise)
}

// fetchVulnsWithClient allows dependency injection for testing
func fetchVulnsWithClient(client *retryablehttp.Client, baseURL, version string, enterprise bool) (string, error) {
	edition := "community"
	if enterprise {
		edition = "enterprise"
	}

	baseCPEUrl := strings.Join([]string{
		baseURL,
		"?cpeName=cpe:2.3:a:gitlab:gitlab:",
		version,
		":*:*:*:",
		edition,
		":*:*:*",
	}, "")

	firstPageURL := fmt.Sprintf("%s&resultsPerPage=%d&startIndex=0", baseCPEUrl, resultsPerPage)
	firstPageData, err := fetchPage(client, firstPageURL)
	if err != nil {
		return "{}", err
	}

	if firstPageData.TotalResults <= resultsPerPage {
		jsonData, err := json.Marshal(firstPageData)
		if err != nil {
			log.Error().Err(err).Msg("failed to marshal response")
			return "{}", err
		}
		return string(jsonData), nil
	}

	log.Debug().Int("totalResults", firstPageData.TotalResults).Int("resultsPerPage", resultsPerPage).Msg("Fetching paginated results")

	allVulns := firstPageData.Vulnerabilities

	for startIndex := resultsPerPage; startIndex < firstPageData.TotalResults; startIndex += resultsPerPage {
		pageURL := fmt.Sprintf("%s&resultsPerPage=%d&startIndex=%d", baseCPEUrl, resultsPerPage, startIndex)
		pageData, err := fetchPage(client, pageURL)
		if err != nil {
			log.Warn().Err(err).Int("startIndex", startIndex).Msg("failed to fetch page, continuing with partial results")
			break
		}
		allVulns = append(allVulns, pageData.Vulnerabilities...)
	}

	finalResponse := firstPageData
	finalResponse.Vulnerabilities = allVulns
	finalResponse.ResultsPerPage = len(allVulns)
	finalResponse.StartIndex = 0

	jsonData, err := json.Marshal(finalResponse)
	if err != nil {
		log.Error().Err(err).Msg("failed to marshal final response")
		return "{}", err
	}

	return string(jsonData), nil
}

func fetchPage(client *retryablehttp.Client, url string) (*nvdResponse, error) {
	res, err := client.Get(url)
	if err != nil {
		return nil, err
	}
	defer func() { _ = res.Body.Close() }()

	if res.StatusCode != 200 {
		log.Error().Int("http", res.StatusCode).Str("url", url).Msg("failed fetching vulnerabilities")
		return nil, fmt.Errorf("HTTP %d", res.StatusCode)
	}

	resData, err := io.ReadAll(res.Body)
	if err != nil {
		log.Error().Int("http", res.StatusCode).Msg("unable to read HTTP response body")
		return nil, err
	}

	var nvdResp nvdResponse
	if err := json.Unmarshal(resData, &nvdResp); err != nil {
		log.Error().Err(err).Msg("failed to unmarshal NVD response")
		return nil, err
	}

	return &nvdResp, nil
}

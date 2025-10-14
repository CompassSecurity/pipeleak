package nist

import (
	"io"

	"github.com/CompassSecurity/pipeleak/helper"
	"github.com/rs/zerolog/log"
)

func FetchVulns(version string) (string, error) {
	client := helper.GetPipeleakHTTPClient()
	res, err := client.Get("https://services.nvd.nist.gov/rest/json/cves/2.0?cpeName=cpe:2.3:a:gitlab:gitlab:" + version + ":*:*:*:*:*:*:*")
	if err != nil {
		return "{}", err
	}
	defer func() { _ = res.Body.Close() }()

	if res.StatusCode == 200 {
		resData, err := io.ReadAll(res.Body)
		if err != nil {
			log.Error().Int("http", res.StatusCode).Msg("unable to read HTTP response body")
			return "{}", err
		}

		return string(resData), nil
	} else {
		log.Error().Int("http", res.StatusCode).Msg("failed fetching vulnerabilities")
		return "{}", nil
	}
}

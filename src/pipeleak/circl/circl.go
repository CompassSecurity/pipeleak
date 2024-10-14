package circl

import (
	"github.com/CompassSecurity/pipeleak/helper"
	"github.com/rs/zerolog/log"
	"io"
)

func FetchVulns() (string, error) {
	client := helper.GetNonVerifyingHTTPClient()
	res, err := client.Get("https://vulnerability.circl.lu/api/search/gitlab/gitlab")
	defer res.Body.Close()
	if err != nil {
		return "{}", err
	}

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

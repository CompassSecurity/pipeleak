package http

import (
	"context"
	"crypto/tls"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"os"

	"github.com/hashicorp/go-retryablehttp"
	"github.com/rs/zerolog/log"
)

type headerRoundTripper struct {
	headers map[string]string
	next    http.RoundTripper
}

func (hrt *headerRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	if hrt.headers == nil {
		return hrt.next.RoundTrip(req)
	}

	for k, v := range hrt.headers {
		if req.Header.Get(k) == "" {
			req.Header.Set(k, v)
		}
	}
	return hrt.next.RoundTrip(req)
}

func GetPipeleakHTTPClient(cookieUrl string, cookies []*http.Cookie, defaultHeaders map[string]string) *retryablehttp.Client {
	var jar http.CookieJar

	if len(cookies) > 0 {
		var err error
		jar, err = cookiejar.New(nil)
		if err != nil {
			log.Fatal().Err(err).Msg("Failed creating cookie jar")
		}

		urlParsed, err := url.Parse(cookieUrl)
		if err != nil {
			log.Fatal().Err(err).Msg("Failed parsing URL for cookie jar")
		}

		jar.SetCookies(urlParsed, cookies)
	}

	client := retryablehttp.NewClient()
	client.Logger = nil
	client.HTTPClient.Jar = jar

	client.CheckRetry = func(ctx context.Context, resp *http.Response, err error) (bool, error) {
		if err != nil {
			log.Error().Err(err).Msg("Retrying HTTP request, error occurred")
			return true, nil
		}

		if resp == nil {
			log.Error().Msg("Retrying HTTP request, no response")
			return false, nil
		}

		if resp.StatusCode == 429 || (resp.StatusCode >= 500 && resp.StatusCode != 501) {
			log.Trace().Str("url", resp.Request.URL.String()).Int("statusCode", resp.StatusCode).Msg("Retrying HTTP request")
			return true, nil
		}

		return false, nil
	}

	// #nosec G402 - InsecureSkipVerify required for security scanning tool to connect to untrusted targets
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}

	proxyServer, useHttpProxy := os.LookupEnv("HTTP_PROXY")
	if useHttpProxy {
		proxyUrl, err := url.Parse(proxyServer)
		if err != nil {
			log.Fatal().Err(err).Str("HTTP_PROXY", proxyServer).Msg("Invalid Proxy URL in HTTP_PROXY environment variable")
		}
		log.Info().Str("proxy", proxyUrl.String()).Msg("Using HTTP_PROXY")
		tr.Proxy = http.ProxyURL(proxyUrl)
	}

	client.HTTPClient.Transport = &headerRoundTripper{
		headers: defaultHeaders,
		next:    tr,
	}

	return client
}

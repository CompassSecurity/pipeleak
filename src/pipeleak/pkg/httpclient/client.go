package httpclient

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

type HeaderRoundTripper struct {
	Headers map[string]string
	Next    http.RoundTripper
}

// RoundTrip adds default headers when they're not present on the request
// and delegates to the next RoundTripper.
func (hrt *HeaderRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	if hrt.Headers == nil || hrt.Next == nil {
		return hrt.Next.RoundTrip(req)
	}

	for k, v := range hrt.Headers {
		if req.Header.Get(k) == "" {
			req.Header.Set(k, v)
		}
	}
	return hrt.Next.RoundTrip(req)
}

// GetPipeleakHTTPClient returns a configured retryablehttp client with optional
// cookie jar and default headers. This is a drop-in replacement for the
// helper-level function but lives in a focused package.
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
			log.Trace().Int("statusCode", resp.StatusCode).Msg("Retrying HTTP request")
			return true, nil
		}

		return false, nil
	}

	tr := &http.Transport{TLSClientConfig: &tls.Config{InsecureSkipVerify: true}}

	proxyServer, useHttpProxy := os.LookupEnv("HTTP_PROXY")
	if useHttpProxy {
		proxyUrl, err := url.Parse(proxyServer)
		if err != nil {
			log.Fatal().Err(err).Str("HTTP_PROXY", proxyServer).Msg("Invalid Proxy URL in HTTP_PROXY environment variable")
		}
		log.Info().Str("proxy", proxyUrl.String()).Msg("Using HTTP_PROXY")
		tr.Proxy = http.ProxyURL(proxyUrl)
	}

	client.HTTPClient.Transport = &HeaderRoundTripper{Headers: defaultHeaders, Next: tr}
	return client
}

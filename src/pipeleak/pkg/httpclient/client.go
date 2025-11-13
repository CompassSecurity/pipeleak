// Package httpclient provides a centralized HTTP client configuration for pipeleak.
// It offers a retryable HTTP client with cookie support, custom headers, and proxy configuration.
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

// HeaderRoundTripper is an http.RoundTripper that adds default headers to requests.
// Headers are only added if they're not already present in the request.
type HeaderRoundTripper struct {
	Headers map[string]string
	Next    http.RoundTripper
}

// RoundTrip adds default headers when they're not present on the request
// and delegates to the next RoundTripper.
func (hrt *HeaderRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	if hrt.Next == nil {
		return nil, http.ErrNotSupported
	}

	if hrt.Headers != nil {
		for k, v := range hrt.Headers {
			if req.Header.Get(k) == "" {
				req.Header.Set(k, v)
			}
		}
	}

	return hrt.Next.RoundTrip(req)
}

// GetPipeleakHTTPClient creates and configures a retryable HTTP client for pipeleak operations.
// It supports:
//   - Cookie jar configuration for session management
//   - Custom default headers
//   - Automatic retry logic for 429 and 5xx errors (except 501)
//   - HTTP proxy support via HTTP_PROXY environment variable
//   - TLS certificate verification bypass (InsecureSkipVerify)
//
// Parameters:
//   - cookieUrl: The URL to associate cookies with (required if cookies are provided)
//   - cookies: Optional cookies to add to the jar
//   - defaultHeaders: Optional headers to add to all requests
//
// Returns a configured *retryablehttp.Client ready for use.
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
			url := ""
			if resp.Request != nil && resp.Request.URL != nil {
				url = resp.Request.URL.String()
			}
			log.Trace().Str("url", url).Int("statusCode", resp.StatusCode).Msg("Retrying HTTP request")
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

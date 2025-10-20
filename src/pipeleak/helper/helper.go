package helper

import (
	"archive/zip"
	"bytes"
	"context"
	"crypto/tls"
	"encoding/base64"
	"math/rand"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"os"
	"os/signal"
	"runtime"
	"strings"
	"syscall"
	"time"

	"atomicgo.dev/keyboard"
	"atomicgo.dev/keyboard/keys"
	"github.com/hashicorp/go-retryablehttp"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"golang.org/x/net/html"
	"gopkg.in/yaml.v3"
)

func SetLogLevel(verbose bool) {
	if verbose {
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
		log.Debug().Msg("Verbose log output enabled")
	}
}

type ShortcutStatusFN func() *zerolog.Event

func ShortcutListeners(status ShortcutStatusFN) {
	err := keyboard.Listen(func(key keys.Key) (stop bool, err error) {
		switch key.Code {
		case keys.CtrlC, keys.Escape:
			return true, nil
		case keys.RuneKey:
			if key.String() == "t" {
				zerolog.SetGlobalLevel(zerolog.TraceLevel)
				log.Info().Str("logLevel", "trace").Msg("New Log level")
			}

			if key.String() == "d" {
				zerolog.SetGlobalLevel(zerolog.DebugLevel)
				log.Info().Str("logLevel", "debug").Msg("New Log level")
			}

			if key.String() == "i" {
				zerolog.SetGlobalLevel(zerolog.InfoLevel)
				log.Info().Str("logLevel", "info").Msg("New Log level")
			}

			if key.String() == "w" {
				zerolog.SetGlobalLevel(zerolog.WarnLevel)
				log.Info().Str("logLevel", "warn").Msg("New Log level")
			}

			if key.String() == "e" {
				zerolog.SetGlobalLevel(zerolog.ErrorLevel)
				log.Info().Str("logLevel", "error").Msg("New Log level")
			}

			if key.String() == "s" {
				log := status()
				log.Msg("Status")
			}
		}

		return false, nil
	})

	if err != nil {
		log.Error().Err(err).Msg("Failed hooking keyboard bindings")
	}
}

func CalculateZipFileSize(data []byte) uint64 {
	reader := bytes.NewReader(data)
	zipListing, err := zip.NewReader(reader, int64(len(data)))
	if err != nil {
		log.Error().Msg("Failed calculcatingZipFileSize")
		return 0
	}
	totalSize := uint64(0)
	for _, file := range zipListing.File {
		totalSize = totalSize + file.UncompressedSize64
	}

	return totalSize
}

type ShutdownHandler func()

func RegisterGracefulShutdownHandler(handler ShutdownHandler) {
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		sig := <-sigs
		handler()
		log.Info().Str("signal", sig.String()).Msg("Pipeleak has been terminated")
		os.Exit(1)
	}()

}

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

	jar := http.DefaultClient.Jar

	if len(cookies) > 0 {
		jar, err := cookiejar.New(nil)
		if err != nil {
			log.Fatal().Err(err).Msg("Failed creating cookie jar")
		}

		urlParsed, err := url.Parse(cookieUrl)
		if err != nil {
			log.Fatal().Err(err).Msg("Failed parsing URL for cookie jar")
		}

		jar.SetCookies(urlParsed, cookies)
		log.Debug().Str("url", urlParsed.String()).Int("cookiesCount", len(cookies)).Msg("Added cookies for HTTP client")
	}

	client := retryablehttp.NewClient()

	client.Logger = nil // Disable logging completely

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

func IsDirectory(path string) bool {
	fileInfo, err := os.Stat(path)
	if err != nil {
		return true
	}

	return fileInfo.IsDir()
}

func ParseISO8601(dateStr string) time.Time {
	t, err := time.Parse(time.RFC3339, dateStr)
	if err != nil {
		log.Fatal().Err(err).Msg("Invalid date input, not ISO8601 compatible")
	}

	return t
}

func PrettyPrintYAML(yamlStr string) (string, error) {
	var node yaml.Node

	err := yaml.Unmarshal([]byte(yamlStr), &node)
	if err != nil {
		return "", err
	}

	var buf bytes.Buffer
	encoder := yaml.NewEncoder(&buf)
	encoder.SetIndent(2)

	err = encoder.Encode(&node)
	if err != nil {
		return "", err
	}

	return buf.String(), nil
}

func ContainsI(a string, b string) bool {
	return strings.Contains(
		strings.ToLower(a),
		strings.ToLower(b),
	)
}

func GetPlatformAgnosticNewline() string {
	newline := "\n"
	if runtime.GOOS == "windows" {
		newline = "\r\n"
	}

	return newline
}

func RandomStringN(n int) string {
	letterBytes := "abcdefghijklmnopqrstuvwxyz"
	b := make([]byte, n)
	for i := range b {
		b[i] = letterBytes[rand.Intn(len(letterBytes))]
	}
	return string(b)
}

func ExtractHTMLTitleFromB64Html(body []byte) string {
	decoded, err := base64.StdEncoding.DecodeString(string(body))
	if err != nil {
		decoded = body
	}

	content := string(decoded)
	contentLower := strings.ToLower(content)

	if !strings.Contains(contentLower, "<html") {
		return ""
	}

	doc, err := html.Parse(strings.NewReader(content))
	if err != nil {
		return ""
	}

	var title string
	var f func(*html.Node)
	f = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == "title" && n.FirstChild != nil {
			title = n.FirstChild.Data
			return
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			f(c)
		}
	}
	f(doc)

	return title
}

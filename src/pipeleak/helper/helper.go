package helper

import (
	"archive/zip"
	"bytes"
	"crypto/tls"
	"encoding/base64"
	"math/rand"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"runtime"
	"strings"
	"syscall"
	"time"

	"atomicgo.dev/keyboard"
	"atomicgo.dev/keyboard/keys"
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

func GetPipeleakHTTPClient() *http.Client {
	proxyServer, useHttpProxy := os.LookupEnv("HTTP_PROXY")
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}

	if useHttpProxy {
		proxyUrl, err := url.Parse(proxyServer)
		if err != nil {
			log.Fatal().Err(err).Str("HTTP_PROXY", proxyServer).Msg("Invalid Proxy URL in HTTP_PROXY environment variable")
		}
		log.Info().Str("proxy", proxyUrl.String()).Msg("Using HTTP_PROXY")
		tr.Proxy = http.ProxyURL(proxyUrl)
	}

	return &http.Client{Transport: tr, Timeout: 60 * time.Second}
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

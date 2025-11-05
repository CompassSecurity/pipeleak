package helper

import (
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"atomicgo.dev/keyboard"
	"atomicgo.dev/keyboard/keys"
	"github.com/hashicorp/go-retryablehttp"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	"github.com/CompassSecurity/pipeleak/pkg/format"
	"github.com/CompassSecurity/pipeleak/pkg/httpclient"
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
			switch key.String() {
			case "t":
				zerolog.SetGlobalLevel(zerolog.TraceLevel)
				log.Info().Str("logLevel", "trace").Msg("New Log level")
			case "d":
				zerolog.SetGlobalLevel(zerolog.DebugLevel)
				log.Info().Str("logLevel", "debug").Msg("New Log level")
			case "i":
				zerolog.SetGlobalLevel(zerolog.InfoLevel)
				log.Info().Str("logLevel", "info").Msg("New Log level")
			case "w":
				zerolog.SetGlobalLevel(zerolog.WarnLevel)
				log.Info().Str("logLevel", "warn").Msg("New Log level")
			case "e":
				zerolog.SetGlobalLevel(zerolog.ErrorLevel)
				log.Info().Str("logLevel", "error").Msg("New Log level")
			case "s":
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
	return format.CalculateZipFileSize(data)
}

type ShutdownHandler func()

func RegisterGracefulShutdownHandler(handler ShutdownHandler) {
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigs
		handler()
		log.Info().Msg("Pipeleak has been terminated")
		os.Exit(1)
	}()
}

// GetPipeleakHTTPClient is a compatibility wrapper that delegates to pkg/httpclient
func GetPipeleakHTTPClient(cookieUrl string, cookies []*http.Cookie, defaultHeaders map[string]string) *retryablehttp.Client {
	return httpclient.GetPipeleakHTTPClient(cookieUrl, cookies, defaultHeaders)
}

// Format helpers delegated to pkg/format for better modularity
func IsDirectory(path string) bool                   { return format.IsDirectory(path) }
func ParseISO8601(dateStr string) time.Time          { return format.ParseISO8601(dateStr) }
func PrettyPrintYAML(yamlStr string) (string, error) { return format.PrettyPrintYAML(yamlStr) }
func ContainsI(a string, b string) bool              { return format.ContainsI(a, b) }
func GetPlatformAgnosticNewline() string             { return format.GetPlatformAgnosticNewline() }
func RandomStringN(n int) string                     { return format.RandomStringN(n) }
func ExtractHTMLTitleFromB64Html(body []byte) string { return format.ExtractHTMLTitleFromB64Html(body) }

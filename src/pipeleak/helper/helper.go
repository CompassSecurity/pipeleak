package helper

import (
	"archive/zip"
	"bytes"
	"crypto/tls"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"path"
	"regexp"
	"strings"
	"syscall"
	"time"

	"atomicgo.dev/keyboard"
	"atomicgo.dev/keyboard/keys"
	"github.com/PuerkitoBio/goquery"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	gitlab "gitlab.com/gitlab-org/api/client-go"
	"gopkg.in/headzoo/surf.v1"
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

func CookieSessionValid(gitlabUrl string, cookieVal string) {
	gitlabSessionsUrl, _ := url.JoinPath(gitlabUrl, "-/user_settings/active_sessions")

	req, err := http.NewRequest("GET", gitlabSessionsUrl, nil)
	if err != nil {
		log.Fatal().Stack().Err(err).Msg("Failed GitLab sessions request")
		return
	}
	req.AddCookie(&http.Cookie{Name: "_gitlab_session", Value: cookieVal})
	client := GetNonVerifyingHTTPClient()
	resp, err := client.Do(req)
	if err != nil {
		log.Fatal().Stack().Err(err).Msg("Failed GitLab session test")
	}
	defer resp.Body.Close()

	statCode := resp.StatusCode

	if statCode != 200 {
		log.Fatal().Int("http", statCode).Msg("Negative _gitlab_session test")
	} else {
		log.Info().Msg("Provided GitLab session cookie is valid")
	}
}

func DetermineVersion(gitlabUrl string, apiToken string) *gitlab.Version {
	if len(apiToken) > 0 {
		git, err := GetGitlabClient(apiToken, gitlabUrl)
		if err != nil {
			return &gitlab.Version{Version: "none", Revision: "none"}
		}

		version, _, err := git.Version.GetVersion()
		if err != nil {
			return &gitlab.Version{Version: "none", Revision: "none"}
		}
		return version
	} else {
		u, err := url.Parse(gitlabUrl)
		if err != nil {
			return &gitlab.Version{Version: "none", Revision: "none"}
		}
		u.Path = path.Join(u.Path, "/help")

		client := GetNonVerifyingHTTPClient()
		response, err := client.Get(u.String())

		if err != nil {
			log.Warn().Msg(gitlabUrl)
			return &gitlab.Version{Version: "none", Revision: "none"}
		}

		responseData, err := io.ReadAll(response.Body)
		if err != nil {
			return &gitlab.Version{Version: "none", Revision: "none"}
		}

		extractLineR := regexp.MustCompile(`instance_version":"\d*.\d*.\d*"`)
		fullLine := extractLineR.Find(responseData)
		versionR := regexp.MustCompile(`\d+.\d+.\d+`)
		versionNumber := versionR.Find(fullLine)

		if len(versionNumber) > 3 {
			return &gitlab.Version{Version: string(versionNumber), Revision: "none"}
		}
		return &gitlab.Version{Version: "none", Revision: "none"}
	}
}

func RegisterNewAccount(targetUrl string, username string, password string, email string) {

	log.Info().Msg("Best effort registration automation - not very reliable")

	gitlabUrl, err := url.Parse(targetUrl)
	if err != nil {
		log.Fatal().Msg(err.Error())
	}

	gitlabUrl.Path = "/users/sign_up"

	log.Debug().Msg("Navigate to login page")
	bow := surf.NewBrowser()
	err = bow.Open(gitlabUrl.String())
	if err != nil {
		log.Fatal().Msg(err.Error())
	}

	log.Debug().Msg("Submit registration form")
	fm, err := bow.Form("#new_new_user")

	if err != nil {
		log.Fatal().Stack().Err(err).Msg("Failed parsing sign-up form")
	}

	_ = fm.Input("new_user[name]", "Pipeleak Full Name")
	_ = fm.Input("new_user[first_name]", "Pipeleak First Name")
	_ = fm.Input("new_user[last_name]", "Automated Signup")
	_ = fm.Input("new_user[username]", username)
	_ = fm.Input("new_user[email]", email)
	_ = fm.Input("new_user[email_confirmation]", email)
	_ = fm.Input("new_user[password]", password)

	if fm.Submit() != nil {
		log.Error().Msg("Registration failed 🙀 do it manually or try with the -v flag")
		log.Fatal().Msg(err.Error())
	}

	bow.Dom().Find(".navless-container").Each(func(_ int, s *goquery.Selection) {
		log.Debug().Msg(strings.ReplaceAll(s.Text(), "\n\n", ""))
	})

	hasErrors := false
	bow.Dom().Find("#error_explanation").Each(func(_ int, s *goquery.Selection) {
		log.Error().Msg(strings.ReplaceAll(s.Text(), "\n\n", ""))
		hasErrors = true
	})

	bow.Dom().Find(".gl-alert-content").Each(func(_ int, s *goquery.Selection) {
		log.Error().Msg(strings.ReplaceAll(s.Text(), "\n\n", ""))
		hasErrors = true
	})

	if hasErrors {
		log.Error().Msg("Failed registration. Check output above or try with -v")
	} else {
		gitlabUrl.Path = "/users/sign_in"
		log.Info().Str("url", gitlabUrl.String()).Msg("Done! Check your inbox to confirm the account if needed or login directly")
	}
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

func GetNonVerifyingHTTPClient() *http.Client {
	proxyServer, isSet := os.LookupEnv("HTTP_PROXY")

	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}

	if isSet {
		proxyUrl, err := url.Parse(proxyServer)
		if err != nil {
			log.Fatal().Err(err).Str("HTTP_PROXY", proxyServer).Msg("Invalid Proxy URL in HTTP_PROXY environment variable")
		}
		log.Debug().Str("proxy", proxyUrl.String()).Msg("Auto detected proxy")
		tr.Proxy = http.ProxyURL(proxyUrl)
	}

	return &http.Client{Transport: tr, Timeout: 60 * time.Second}
}

func GetGitlabClient(token string, url string) (*gitlab.Client, error) {
	client, err := gitlab.NewClient(token, gitlab.WithBaseURL(url), gitlab.WithHTTPClient(GetNonVerifyingHTTPClient()))
	return client, err
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

package util

import (
	"io"
	"net/http"
	"net/url"
	"path"
	"regexp"
	"strings"

	"github.com/CompassSecurity/pipeleak/helper"
	"github.com/PuerkitoBio/goquery"
	"github.com/headzoo/surf"
	"github.com/rs/zerolog/log"
	gitlab "gitlab.com/gitlab-org/api/client-go"
)

func GetGitlabClient(token string, url string) (*gitlab.Client, error) {
	return gitlab.NewClient(token, gitlab.WithBaseURL(url), gitlab.WithHTTPClient(helper.GetPipeleakHTTPClient()))
}

func CookieSessionValid(gitlabUrl string, cookieVal string) {
	gitlabSessionsUrl, _ := url.JoinPath(gitlabUrl, "-/user_settings/active_sessions")

	req, err := http.NewRequest("GET", gitlabSessionsUrl, nil)
	if err != nil {
		log.Fatal().Stack().Err(err).Msg("Failed GitLab sessions request")
		return
	}
	req.AddCookie(&http.Cookie{Name: "_gitlab_session", Value: cookieVal})
	client := helper.GetPipeleakHTTPClient()
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

		client := helper.GetPipeleakHTTPClient()
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

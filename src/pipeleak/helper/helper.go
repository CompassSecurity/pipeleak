package helper

import (
	"net/url"
	"strings"

	"github.com/PuerkitoBio/goquery"
	"github.com/rs/zerolog/log"
	"gopkg.in/headzoo/surf.v1"
)

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
		log.Fatal().Msg("Failed parsing sign-up form: " + err.Error())
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
		log.Info().Msg("Done! Check your inbox to confirm the account if needed or login directly at " + gitlabUrl.String())
	}
}
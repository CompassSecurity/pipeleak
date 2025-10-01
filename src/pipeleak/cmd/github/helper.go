package github

import (
	"context"
	"net/http"
	"time"

	"github.com/gofri/go-github-ratelimit/v2/github_ratelimit"
	"github.com/gofri/go-github-ratelimit/v2/github_ratelimit/github_primary_ratelimit"
	"github.com/gofri/go-github-ratelimit/v2/github_ratelimit/github_secondary_ratelimit"
	"github.com/google/go-github/v69/github"
	"github.com/rs/zerolog/log"
)

type GitHubScanOptions struct {
	AccessToken            string
	Verbose                bool
	ConfidenceFilter       []string
	MaxScanGoRoutines      int
	TruffleHogVerification bool
	MaxWorkflows           int
	Organization           string
	Owned                  bool
	User                   string
	Public                 bool
	SearchQuery            string
	Artifacts              bool
	Context                context.Context
	Client                 *github.Client
	HttpClient             *http.Client
}

var options = GitHubScanOptions{}

func setupClient(accessToken string) *github.Client {
	rateLimiter := github_ratelimit.New(nil,
		github_primary_ratelimit.WithLimitDetectedCallback(func(ctx *github_primary_ratelimit.CallbackContext) {
			resetTime := ctx.ResetTime.Add(time.Duration(time.Second * 30))
			log.Info().Str("category", string(ctx.Category)).Time("reset", resetTime).Msg("Primary rate limit detected, will resume automatically")
			time.Sleep(time.Until(resetTime))
			log.Info().Str("category", string(ctx.Category)).Msg("Resuming")
		}),
		github_secondary_ratelimit.WithLimitDetectedCallback(func(ctx *github_secondary_ratelimit.CallbackContext) {
			resetTime := ctx.ResetTime.Add(time.Duration(time.Second * 30))
			log.Info().Time("reset", *ctx.ResetTime).Dur("totalSleep", *ctx.TotalSleepTime).Msg("Secondary rate limit detected, will resume automatically")
			time.Sleep(time.Until(resetTime))
			log.Info().Msg("Resuming")
		}),
	)

	return github.NewClient(&http.Client{Transport: rateLimiter}).WithAuthToken(accessToken)
}
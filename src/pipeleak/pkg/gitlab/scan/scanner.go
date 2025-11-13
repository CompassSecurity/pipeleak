package scan

import (
	"net/url"

	"github.com/CompassSecurity/pipeleak/cmd/gitlab/util"
	"github.com/CompassSecurity/pipeleak/pkg/format"
	"github.com/CompassSecurity/pipeleak/pkg/scanner"
	"github.com/rs/zerolog/log"
)

// Scanner provides methods for scanning GitLab projects for secrets.
// It extends scanner.BaseScanner with GitLab-specific functionality.
type Scanner interface {
	scanner.BaseScanner
	// GetQueueStatus returns the current queue status
	GetQueueStatus() int
}

// gitlabScanner implements the Scanner interface.
type gitlabScanner struct {
	options *ScanOptions
}

// Ensure gitlabScanner implements scanner.BaseScanner
var _ scanner.BaseScanner = (*gitlabScanner)(nil)

// NewScanner creates a new GitLab scanner with the provided options.
func NewScanner(opts *ScanOptions) Scanner {
	return &gitlabScanner{
		options: opts,
	}
}

// Scan performs the GitLab scanning operation.
func (s *gitlabScanner) Scan() error {
	version := util.DetermineVersion(s.options.GitlabUrl, s.options.GitlabApiToken)
	log.Info().Str("version", version.Version).Str("revision", version.Revision).Msg("Gitlab Version Check")

	ScanGitLabPipelines(s.options)
	log.Info().Msg("Scan Finished, Bye Bye 🏳️‍🌈🔥")
	return nil
}

// GetQueueStatus returns the current queue status.
func (s *gitlabScanner) GetQueueStatus() int {
	return GetQueueStatus()
}

// InitializeOptions prepares scan options from CLI parameters.
func InitializeOptions(gitlabUrl, gitlabApiToken, gitlabCookie, projectSearchQuery, repository, namespace, queueFolder, maxArtifactSizeStr string,
	artifacts, owned, member, truffleHogVerification bool,
	jobLimit, maxScanGoRoutines int, confidenceFilter []string) (*ScanOptions, error) {

	_, err := url.ParseRequestURI(gitlabUrl)
	if err != nil {
		return nil, err
	}

	byteSize, err := format.ParseHumanSize(maxArtifactSizeStr)
	if err != nil {
		return nil, err
	}

	return &ScanOptions{
		GitlabUrl:              gitlabUrl,
		GitlabApiToken:         gitlabApiToken,
		GitlabCookie:           gitlabCookie,
		ProjectSearchQuery:     projectSearchQuery,
		Artifacts:              artifacts,
		Owned:                  owned,
		Member:                 member,
		Repository:             repository,
		Namespace:              namespace,
		JobLimit:               jobLimit,
		ConfidenceFilter:       confidenceFilter,
		MaxArtifactSize:        byteSize,
		MaxScanGoRoutines:      maxScanGoRoutines,
		QueueFolder:            queueFolder,
		TruffleHogVerification: truffleHogVerification,
	}, nil
}

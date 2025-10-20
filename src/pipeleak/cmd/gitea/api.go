package gitea

import (
	"context"
	"net/http"

	"code.gitea.io/sdk/gitea"
	"github.com/hashicorp/go-retryablehttp"
)

type GiteaScanOptions struct {
	Token                  string
	GiteaURL               string
	Artifacts              bool
	Verbose                bool
	ConfidenceFilter       []string
	MaxScanGoRoutines      int
	TruffleHogVerification bool
	Owned                  bool
	Organization           string
	Repository             string
	Cookie                 string
	RunsLimit              int
	Context                context.Context
	Client                 *gitea.Client
	HttpClient             *retryablehttp.Client
}

type AuthTransport struct {
	Base  http.RoundTripper
	Token string
}

func (t *AuthTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	req2 := req.Clone(req.Context())
	req2.Header.Set("Authorization", "token "+t.Token)
	return t.Base.RoundTrip(req2)
}

type ActionWorkflowRun struct {
	ID         int64  `json:"id"`
	Name       string `json:"name"`
	Title      string `json:"title"`
	Status     string `json:"status"`
	Event      string `json:"event"`
	CreatedAt  string `json:"created_at"`
	UpdatedAt  string `json:"updated_at"`
	RunNumber  int64  `json:"run_number"`
	WorkflowID string `json:"workflow_id"`
	HeadBranch string `json:"head_branch"`
	HeadSha    string `json:"head_sha"`
	URL        string `json:"url"`
	HTMLURL    string `json:"html_url"`
}

type ActionWorkflowRunsResponse struct {
	TotalCount   int64               `json:"total_count"`
	WorkflowRuns []ActionWorkflowRun `json:"workflow_runs"`
}

type ActionArtifact struct {
	ID                 int64  `json:"id"`
	Name               string `json:"name"`
	Size               int64  `json:"size"`
	CreatedAt          string `json:"created_at"`
	ExpiredAt          string `json:"expired_at"`
	WorkflowRunID      int64  `json:"workflow_run_id"`
	ArchiveDownloadURL string `json:"archive_download_url"`
}

type ActionArtifactsResponse struct {
	TotalCount int64            `json:"total_count"`
	Artifacts  []ActionArtifact `json:"artifacts"`
}

type ActionJob struct {
	ID          int64  `json:"id"`
	Name        string `json:"name"`
	Status      string `json:"status"`
	Conclusion  string `json:"conclusion"`
	StartedAt   string `json:"started_at"`
	CompletedAt string `json:"completed_at"`
	RunID       int64  `json:"run_id"`
	TaskID      int64  `json:"task_id"`
}

type ActionJobsResponse struct {
	TotalCount int64       `json:"total_count"`
	Jobs       []ActionJob `json:"jobs"`
}

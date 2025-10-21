package gitea

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"net/url"

	"code.gitea.io/sdk/gitea"
	"github.com/rs/zerolog/log"
)

// httpResponse represents a generic HTTP response with common fields
type httpResponse struct {
	Body       []byte
	StatusCode int
}

// makeHTTPRequest performs a GET request and returns the response
func makeHTTPRequest(url string) (*httpResponse, error) {
	resp, err := scanOptions.HttpClient.Get(url)
	if err != nil {
		return nil, fmt.Errorf("HTTP request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	return &httpResponse{
		Body:       body,
		StatusCode: resp.StatusCode,
	}, nil
}

// makeHTTPPostRequest performs a POST request with custom headers
func makeHTTPPostRequest(urlStr string, body []byte, headers map[string]string) (*httpResponse, error) {
	client := scanOptions.HttpClient.StandardClient()

	var bodyReader io.Reader
	if body != nil {
		bodyReader = bytes.NewReader(body)
	}

	req, err := http.NewRequest("POST", urlStr, bodyReader)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	for key, value := range headers {
		req.Header.Set(key, value)
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("HTTP POST request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	return &httpResponse{
		Body:       respBody,
		StatusCode: resp.StatusCode,
	}, nil
}

// buildGiteaURL constructs a URL for Gitea API endpoints
func buildGiteaURL(pathFormat string, args ...interface{}) (string, error) {
	link, err := url.Parse(scanOptions.GiteaURL)
	if err != nil {
		return "", fmt.Errorf("failed to parse Gitea URL: %w", err)
	}
	link.Path = fmt.Sprintf(pathFormat, args...)
	return link.String(), nil
}

// buildAPIURL constructs a URL for Gitea API with query parameters
func buildAPIURL(repo *gitea.Repository, pathFormat string, pathArgs ...interface{}) (string, error) {
	link, err := url.Parse(scanOptions.GiteaURL)
	if err != nil {
		return "", err
	}

	// Build the path with repository info
	basePath := fmt.Sprintf("/api/v1/repos/%s/%s", repo.Owner.UserName, repo.Name)
	link.Path = basePath + fmt.Sprintf(pathFormat, pathArgs...)

	return link.String(), nil
}

// logContext holds common logging fields
type logContext struct {
	Repo  string
	RunID int64
	JobID int64
}

// logHTTPError logs HTTP errors with consistent formatting
func logHTTPError(statusCode int, operation string, ctx logContext) {
	event := log.Error().Int("status", statusCode)

	if ctx.Repo != "" {
		event = event.Str("repo", ctx.Repo)
	}
	if ctx.RunID > 0 {
		event = event.Int64("run_id", ctx.RunID)
	}
	if ctx.JobID > 0 {
		event = event.Int64("job_id", ctx.JobID)
	}

	event.Msgf("failed to %s", operation)
}

// checkHTTPStatus checks HTTP status codes and returns appropriate errors
func checkHTTPStatus(statusCode int, operation string) error {
	switch statusCode {
	case 200:
		return nil
	case 404:
		return fmt.Errorf("resource not found (404)")
	case 403:
		return fmt.Errorf("access forbidden (403)")
	case 410:
		return fmt.Errorf("resource gone (410)")
	default:
		if statusCode >= 400 {
			return fmt.Errorf("HTTP error: %d", statusCode)
		}
		return nil
	}
}

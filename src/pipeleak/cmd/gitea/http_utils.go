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

type httpResponse struct {
	Body       []byte
	StatusCode int
}

func makeHTTPRequest(url string) (*httpResponse, error) {
	if scanOptions.HttpClient == nil {
		return nil, fmt.Errorf("HTTP client is not initialized")
	}

	resp, err := scanOptions.HttpClient.Get(url)
	if err != nil {
		return nil, fmt.Errorf("HTTP request failed: %w", err)
	}

	if resp == nil {
		return nil, fmt.Errorf("HTTP response is nil")
	}

	defer func() {
		if err := resp.Body.Close(); err != nil {
			log.Debug().Err(err).Msg("Failed to close HTTP response body")
		}
	}()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	return &httpResponse{
		Body:       body,
		StatusCode: resp.StatusCode,
	}, nil
}

func makeHTTPPostRequest(urlStr string, body []byte, headers map[string]string) (*httpResponse, error) {
	if scanOptions.HttpClient == nil {
		return nil, fmt.Errorf("HTTP client is not initialized")
	}

	client := scanOptions.HttpClient.StandardClient()
	if client == nil {
		return nil, fmt.Errorf("standard HTTP client is not initialized")
	}

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

	if resp == nil {
		return nil, fmt.Errorf("HTTP response is nil")
	}

	defer func() {
		if err := resp.Body.Close(); err != nil {
			log.Debug().Err(err).Msg("Failed to close HTTP POST response body")
		}
	}()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	return &httpResponse{
		Body:       respBody,
		StatusCode: resp.StatusCode,
	}, nil
}

func buildGiteaURL(pathFormat string, args ...interface{}) (string, error) {
	link, err := url.Parse(scanOptions.GiteaURL)
	if err != nil {
		return "", fmt.Errorf("failed to parse Gitea URL: %w", err)
	}
	link.Path = fmt.Sprintf(pathFormat, args...)
	return link.String(), nil
}

func buildAPIURL(repo *gitea.Repository, pathFormat string, pathArgs ...interface{}) (string, error) {
	if repo == nil {
		return "", fmt.Errorf("repository is nil")
	}

	if repo.Owner == nil {
		return "", fmt.Errorf("repository owner is nil")
	}

	link, err := url.Parse(scanOptions.GiteaURL)
	if err != nil {
		return "", err
	}

	basePath := fmt.Sprintf("/api/v1/repos/%s/%s", repo.Owner.UserName, repo.Name)
	link.Path = basePath + fmt.Sprintf(pathFormat, pathArgs...)

	return link.String(), nil
}

type logContext struct {
	Repo  string
	RunID int64
	JobID int64
}

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

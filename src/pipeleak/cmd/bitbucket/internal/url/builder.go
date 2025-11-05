package url

import (
	"net/url"
	"path"
	"strings"
)

// GetWebBaseURL derives the web base URL from the API base URL
// For example: "https://api.bitbucket.org/2.0" -> "https://bitbucket.org"
func GetWebBaseURL(apiBaseURL string) string {
	// Remove /2.0 or any path suffix from API URL
	webURL := strings.TrimSuffix(apiBaseURL, "/2.0")
	// Remove "api." prefix if present
	webURL = strings.Replace(webURL, "://api.", "://", 1)
	return webURL
}

func BuildDownloadArtifactWebURL(baseWebURL, workspaceSlug, repoSlug, artifactName string) (string, error) {
	u, err := url.Parse(baseWebURL)
	if err != nil {
		return "", err
	}
	u.Path = path.Join(u.Path, workspaceSlug, repoSlug, "downloads", artifactName)
	return u.String(), nil
}

func BuildPipelineStepURL(baseWebURL, workspaceSlug, repoSlug, pipelineUUID, stepUUID string) string {
	// Use simple string concatenation to preserve UUID format (including curly braces)
	// and avoid URL encoding by url.URL
	if baseWebURL == "" {
		baseWebURL = "https://bitbucket.org"
	}
	// Remove trailing slash from base URL if present
	baseWebURL = strings.TrimSuffix(baseWebURL, "/")
	
	return baseWebURL + "/" + workspaceSlug + "/" + repoSlug + "/pipelines/results/" + pipelineUUID + "/steps/" + stepUUID
}

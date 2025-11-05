package url

import (
	"net/url"
	"path"
)

func BuildDownloadArtifactWebURL(workspaceSlug, repoSlug, artifactName string) (string, error) {
	u, err := url.Parse("https://bitbucket.org/")
	if err != nil {
		return "", err
	}
	u.Path = path.Join(u.Path, workspaceSlug, repoSlug, "downloads", artifactName)
	return u.String(), nil
}

func BuildPipelineStepURL(workspaceSlug, repoSlug, pipelineUUID, stepUUID string) string {
	return "https://bitbucket.org/" + workspaceSlug + "/" + repoSlug + "/pipelines/results/" + pipelineUUID + "/steps/" + stepUUID
}

package bitbucket

import (
	"net/url"
	"path"
	"strconv"

	"github.com/rs/zerolog/log"
)

func buildWebArtifactUrl(workspaceSlug string, repoSlug string, buildNumber int, stepUUID string) string {
	u, err := url.Parse("https://bitbucket.org/repositories/")
	if err != nil {
		log.Error().Err(err).Msg("Unable to parse web artifact url")
		return "failed building url"
	}
	u.Path = path.Join(u.Path, workspaceSlug, repoSlug, "pipelines", "results", strconv.Itoa(buildNumber), "steps", stepUUID, "artifacts")

	return u.String()
}

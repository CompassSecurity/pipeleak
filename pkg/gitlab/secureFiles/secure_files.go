package securefiles

import (
	"github.com/CompassSecurity/pipeleak/pkg/gitlab/util"
	"github.com/rs/zerolog/log"
	gitlab "gitlab.com/gitlab-org/api/client-go"
)

// RunFetchSecureFiles fetches and prints all CI/CD secure files
func RunFetchSecureFiles(gitlabUrl, gitlabApiToken string) {
	log.Info().Msg("Fetching secure files")

	git, err := util.GetGitlabClient(gitlabApiToken, gitlabUrl)
	if err != nil {
		log.Fatal().Stack().Err(err).Msg("Failed creating gitlab client")
	}

	projectOpts := &gitlab.ListProjectsOptions{
		ListOptions: gitlab.ListOptions{
			PerPage: 100,
			Page:    1,
		},
		MinAccessLevel: gitlab.Ptr(gitlab.MaintainerPermissions),
		OrderBy:        gitlab.Ptr("last_activity_at"),
	}

	err = util.IterateProjects(git, projectOpts, func(project *gitlab.Project) error {
		log.Debug().Str("project", project.WebURL).Msg("Fetch project secure files")
		fileIds, err := GetSecureFiles(project.ID, gitlabUrl, gitlabApiToken)
		if err != nil {
			log.Error().Stack().Err(err).Str("project", project.WebURL).Msg("Failed fetching secure files list")
			return nil // Continue to next project
		}

		for _, id := range fileIds {
			secureFile, downloadUrl, err := DownloadSecureFile(project.ID, id, gitlabUrl, gitlabApiToken)
			if err != nil {
				log.Error().Stack().Err(err).Str("project", project.WebURL).Int64("fileId", id).Msg("Failed fetching secure file")
				continue
			}

			if len(secureFile) > 100 {
				secureFile = secureFile[:100]
			}

			log.Warn().Str("downloadUrl", downloadUrl).Bytes("content", secureFile).Msg("Secure file")
		}
		return nil
	})
	if err != nil {
		log.Fatal().Stack().Err(err).Msg("Failed iterating projects")
	}

	log.Info().Msg("Fetched all secure files")
}

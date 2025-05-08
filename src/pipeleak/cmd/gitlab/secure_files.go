package gitlab

import (
	pgitlab "github.com/CompassSecurity/pipeleak/gitlab"
	"github.com/CompassSecurity/pipeleak/helper"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"gitlab.com/gitlab-org/api/client-go"
)

func NewSecureFilesCmd() *cobra.Command {
	secureFilesCmd := &cobra.Command{
		Use:   "secureFiles [no options!]",
		Short: "Print CI/CD secure files",
		Run:   FetchSecureFiles,
	}
	secureFilesCmd.Flags().StringVarP(&gitlabUrl, "gitlab", "g", "", "GitLab instance URL")
	err := secureFilesCmd.MarkFlagRequired("gitlab")
	if err != nil {
		log.Fatal().Stack().Err(err).Msg("Unable to require gitlab flag")
	}

	secureFilesCmd.Flags().StringVarP(&gitlabApiToken, "token", "t", "", "GitLab API Token")
	err = secureFilesCmd.MarkFlagRequired("token")
	if err != nil {
		log.Fatal().Msg("Unable to require token flag")
	}
	secureFilesCmd.MarkFlagsRequiredTogether("gitlab", "token")

	secureFilesCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "Verbose logging")
	return secureFilesCmd
}

func FetchSecureFiles(cmd *cobra.Command, args []string) {
	if verbose {
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
		log.Debug().Msg("Verbose log output enabled")
	}

	log.Info().Msg("Fetching secure files")

	git, err := helper.GetGitlabClient(gitlabApiToken, gitlabUrl)
	if err != nil {
		log.Fatal().Stack().Err(err).Msg("failed creating gitlab client")
	}

	projectOpts := &gitlab.ListProjectsOptions{
		ListOptions: gitlab.ListOptions{
			PerPage: 100,
			Page:    1,
		},
		MinAccessLevel: gitlab.Ptr(gitlab.MaintainerPermissions),
		OrderBy:        gitlab.Ptr("last_activity_at"),
	}

	for {
		projects, resp, err := git.Projects.ListProjects(projectOpts)
		if err != nil {
			log.Error().Stack().Err(err).Msg("Failed fetching projects")
			break
		}

		for _, project := range projects {
			log.Debug().Str("project", project.WebURL).Msg("Fetch project secure files")
			err, fileIds := pgitlab.GetSecureFiles(project.ID, gitlabUrl, gitlabApiToken)
			if err != nil {
				log.Error().Stack().Err(err).Str("project", project.WebURL).Msg("Failed fetching secure files list")
				continue
			}

			for _, id := range fileIds {
				err, secureFile, downloadUrl := pgitlab.DownloadSecureFile(project.ID, id, gitlabUrl, gitlabApiToken)
				if err != nil {
					log.Error().Stack().Err(err).Str("project", project.WebURL).Int64("fileId", id).Msg("Failed fetching secure file")
					continue
				}

				if len(secureFile) > 100 {
					secureFile = secureFile[:100]
				}

				log.Warn().Str("downloadUrl", downloadUrl).Bytes("content", secureFile).Msg("Secure file")
			}
		}

		if resp.NextPage == 0 {
			break
		}
		projectOpts.Page = resp.NextPage
	}

	log.Info().Msg("Fetched all secure files")
}

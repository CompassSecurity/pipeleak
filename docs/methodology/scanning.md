---
title: Credentials Scanning in GitLab Pipelines
description: Learn how to scan GitLab CI/CD pipelines for exposed secrets and credentials using Pipeleek.
keywords:
  - GitLab pipeline scanning
  - credential scanning
  - secrets detection
  - pipeline security
  - CI/CD security
---

# Credentials Scanning in GitLab Pipelines

> This example focuses on GitLab, but Pipeleek also supports other platforms. Refer to the documentation for details on additional integrations.

Suppose you're conducting a penetration test and have access to a GitLab instance with a user account. Your goal is to scan the pipelines for exposed secrets and credentials.

Start by creating a personal access token (Menu → Preferences → Access Tokens) and grant it read access scopes. Additionally, use your browser's developer tools to extract the session cookie (`_gitlab_session`).

For an initial scan, target all repositories you can access, including public ones. To keep the scan fast and broad, limit it to the latest 15 jobs per project:

```bash
pipeleek gl scan -g https://gitlab.com -t glpat-[redacted] --cookie [redacted] --artifacts --job-limit 15
2025-09-30T09:53:30Z info Gitlab Version Check revision=f0455ea9f90 version=18.5.0-pre
2025-09-30T09:53:30Z info Fetching projects
2025-09-30T09:53:30Z info Provided GitLab session cookie is valid
2025-09-30T09:53:33Z hit SECRET confidence=low type=log jobName=archives-job ruleName=api_key url=gitlab.com/testgroup/project/-/jobs/11484162851 value="m$ mkdir archive_data $ echo \"datadog_api_key=secrets.txt file hit\" > archive_data/secrets_in_ar"
2025-09-30T09:53:36Z hit SECRET confidence=high type=log ruleName="Generic - 1719" url=gitlab.com/testgroup/project/-/jobs/11484162842 value="datadog_api_key=dotenv ONLY file hit, no other artifacts "
2025-09-30T09:53:37Z hit SECRET confidence=high type=artifact file=an_artifact.txt jobName=artifact-job ruleName="Generic - 1719" url=gitlab.com/testgroup/project/-/jobs/11484162833 value="datadog_api_key=secret_artifact_value "
```

As shown, Pipeleek can detect secrets in job logs, dotenv files, and build artifacts. Security findings are logged at the custom `hit` level to distinguish them from regular warnings. Manually review the hits to verify if they're valid credentials. If you see `confidence=high-verified`, it's very likely a real credential, as Pipeleek has tested it against the respective service.

If you find a repository that looks particularly interesting e.g. `secret-pipelines`, you can scan all its job logs, not just the most recent ones:

```bash
pipeleek gl scan -g https://gitlab.com -t glpat-[redacted] --cookie [redacted] --artifacts --repo mygroup/my-secret-pipelines-project
```

## Quiet(er) Scanning

Sometimes you need to scan through a proxy and avoid making too much noise. Disable TruffleHog verification, configure your proxy using environment variables, save logs to disk, and turn off colored logs for easier grepping. Make sure to scan slowly by only using one thread and limit traffic by settting an artifact size limit.

```bash
HTTP_PROXY=http://127.0.0.1:8080 pipeleek gl scan -g https://gitlab.internal-company.com -t glpat-[redacted] --threads 1 --max-artifact-size 5mb --truffle-hog-verification=false --verbose --logfile pipeleek_out --job-limit 10
```

## Custom Rules

To scan for a specific pattern, edit the `rules.yml` file Pipeleek creates on the first run. It looks like:

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

You can add custom rules, remove unnecessary ones, and set your own confidence levels. Test your regexes at [regex101.com](https://regex101.com/) (select Golang flavor).

For example, a custom rule:

```yaml
patterns:
  - pattern:
      name: Pipeleek Custom Rule
      regex: PIPELEAK_.*
      confidence: custom-confidence
```

When you run Pipeleek, you'll see results for your custom rule and any built-in rules:

```bash
pipeleek gl scan -g https://gitlab.com -t glpat-[redacted] --truffle-hog-verification=false --verbose
2025-09-30T11:39:08Z hit SECRET confidence=custom-confidence type=log jobName=build-job-hidden ruleName="Pipeleek Custom Rule" url=gitlab.com/testgroup/project/-/jobs/11547853360 value="PIPELEAK_HIT=secret"
```

## Log Levels

Pipeleek uses structured JSON logging with the following levels:

- `trace`: Most detailed logging, includes all operations
- `debug`: Detailed information for debugging
- `info`: General informational messages
- `warn`: Warning messages for potential issues
- `hit`: **Custom level for security findings** - indicates a potential secret or credential has been detected
- `error`: Error messages for failures

## Interactive Log Level

In the scan mode you can change interactively between log levels by pressing `t`: Trace, `d`: Debug, `i`: info, `w`: Warn, `e`: Error. Pressing `s` will output the current status.

```bash
pipeleek gl scan -g https://gitlab.com -t glpat-[redacted] --truffle-hog-verification=false --verbose
[Pressed d]
2025-09-30T11:42:58Z info New Log level logLevel=debug
```

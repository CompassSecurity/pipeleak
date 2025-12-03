---
title: Secret Verification with TruffleHog
description: Learn how Pipeleek uses TruffleHog to automatically verify detected secrets, understand confidence levels, and how to disable verification for operational security.
keywords:
  - secret verification
  - TruffleHog
  - credential validation
  - confidence levels
  - high-verified
  - secret detection
  - credential testing
  - opsec
---

Pipeleek integrates [TruffleHog v3](https://github.com/trufflesecurity/trufflehog) to automatically detect and verify secrets in CI/CD logs and artifacts. TruffleHog provides many detectors for various services and platforms, each with built-in verification capabilities.

### How It Works

When Pipeleek scans logs or artifacts, it uses two detection engines in parallel:

1. **Pattern-based detection**: Custom YAML rules from `rules.yml` (regex patterns)
2. **TruffleHog detectors**: Specialized detectors with active verification

The TruffleHog engine:

1. **Scans** text for secrets using pattern matching
2. **Extracts** potential credentials (API keys, tokens, passwords)
3. **Verifies** credentials by attempting authentication with the target service
4. **Reports** only verified secrets (by default)

### Verification Process

When verification is **enabled** (default), TruffleHog:

- Makes **live authentication attempts** to validate credentials
- Tests against the actual service (GitHub, AWS, GitLab, etc.)
- Marks secrets as `high-verified` only if authentication succeeds
- Filters out false positives automatically

### Confidence Levels

Pipeleek assigns confidence levels to all detected secrets:

| Level                     | Source     | Description                                       | Verified |
| ------------------------- | ---------- | ------------------------------------------------- | -------- |
| **high-verified**         | TruffleHog | Actively verified and confirmed working           | ✅ Yes   |
| **trufflehog-unverified** | TruffleHog | Detected but not verified (verification disabled) | ❌ No    |
| **high**                  | rules.yml  | High confidence pattern match                     | ❌ No    |
| **medium**                | rules.yml  | Medium confidence pattern match                   | ❌ No    |
| **low**                   | rules.yml  | Low confidence pattern match                      | ❌ No    |
| **custom**                | rules.yml  | User-defined confidence level                     | ❌ No    |

By default, TruffleHog verification is **enabled**. This means:

- Creates **live authentication attempts** against target services
- May trigger security alerts or rate limits
- Authentication attempts are logged by target services
- **Not stealthy** - visible in service audit logs

**Example output with verification enabled:**

```bash
pipeleek gl scan -g https://gitlab.com -t glpat-xxxxx --verbose

2025-12-03T10:15:32Z hit SECRET confidence=high-verified type=log \
  jobName=build-job \
  ruleName="GitLab Personal Access Token" \
  url=gitlab.com/myorg/myproject/-/jobs/123456 \
  value="glpat-xxxxxxxxxxxxxxxxxxxx"
```

### Disabling Verification

For operational security (OpSec) or simply due to privacy concerns, you should disable verification.

Use the `--truffle-hog-verification=false` flag:

```bash
pipeleek gl scan -g https://gitlab.com -t glpat-xxxxx --truffle-hog-verification=false
```

### Confidence Filtering

Results can be filtered by confidence, using the `--confidence` flag.

```bash
pipeleek gl scan -g https://gitlab.com -t glpat-xxxxx --confidence=high-verified,highj
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
      regex: PIPELEEK_.*
      confidence: custom-confidence
```

When you run Pipeleek, you'll see results for your custom rule and any built-in rules:

```bash
pipeleek gl scan -g https://gitlab.com -t glpat-[redacted] --truffle-hog-verification=false --verbose
2025-09-30T11:39:08Z hit SECRET confidence=custom-confidence type=log jobName=build-job-hidden ruleName="Pipeleek Custom Rule" url=gitlab.com/testgroup/project/-/jobs/11547853360 value="PIPELEEK_HIT=secret"
```

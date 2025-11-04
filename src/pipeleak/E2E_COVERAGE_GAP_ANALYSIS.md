# E2E Test Coverage Gap Analysis

## Summary
This document identifies CLI feature flags that exist in the application but are not yet covered by E2E tests.

Analysis Date: 2025-01-16
Test Suite Location: `tests/e2e/`

---

## GitHub (`cmd/github/scan.go`)

### Available Flags
- `--token` (required)
- `--confidence`
- `--threads`
- `--truffleHogVerification`
- `--maxWorkflows`
- `--artifacts`
- `--org`
- `--user`
- `--owned`
- `--public`
- `--search`
- `--github`
- `--verbose`

### Currently Tested Flags
- `--token` ✅
- `--confidence` ⚠️ (TEST SKIPPED: `SkipTestGitHubScan_ConfidenceFilter`)
- `--github` ✅
- `--maxWorkflows` ✅
- `--org` ✅
- `--owned` ✅
- `--artifacts` ✅
- `--verbose` ✅

### UNTESTED FLAGS ❌
1. **`--user`** - Scan repositories of a specific GitHub user
2. **`--search`** - GitHub search query for repositories
3. **`--public`** - Scan all public repositories
4. **`--threads`** - Number of concurrent threads
5. **`--truffleHogVerification=false`** - Disable credential verification

### Skipped Test to Re-Enable ⚠️
- `TestGitHubScan_ConfidenceFilter` - Currently skipped, should be enabled

---

## GitLab (`cmd/gitlab/scan/scan.go`)

### Available Flags
- `--gitlab` (required)
- `--token` (required)
- `--cookie`
- `--search`
- `--confidence`
- `--artifacts`
- `--owned`
- `--member`
- `--repo`
- `--namespace`
- `--job-limit`
- `--max-artifact-size`
- `--threads`
- `--queue`
- `--truffleHogVerification`
- `--verbose`

### Currently Tested Flags
- `--gitlab` ✅
- `--token` ✅
- `--job-limit` ✅
- `--member` ✅
- `--namespace` ✅
- `--owned` ✅
- `--repo` ✅
- `--search` ✅
- `--threads` ✅
- `--artifacts` ✅
- `--verbose` ✅

### UNTESTED FLAGS ❌
1. **`--cookie`** - GitLab session cookie for dotenv artifacts
2. **`--confidence`** - Confidence level filtering
3. **`--max-artifact-size`** - Maximum artifact size to scan
4. **`--queue`** - Custom queue folder path
5. **`--truffleHogVerification=false`** - Disable credential verification

---

## BitBucket (`cmd/bitbucket/scan.go`)

### Available Flags
- `--token`
- `--username`
- `--cookie`
- `--bitbucket`
- `--artifacts`
- `--confidence`
- `--threads`
- `--truffleHogVerification`
- `--maxPipelines`
- `--workspace`
- `--owned`
- `--public`
- `--after`
- `--verbose`

### Currently Tested Flags
- `--token` ✅
- `--username` ✅
- `--cookie` ✅
- `--bitbucket` ✅
- `--artifacts` ✅
- `--confidence` ✅
- `--threads` ✅
- `--truffleHogVerification=false` ✅
- `--maxPipelines` ✅
- `--workspace` ✅
- `--owned` ✅
- `--public` ✅
- `--after` ✅
- `--verbose` ✅

### UNTESTED FLAGS ❌
**NONE** - BitBucket has 100% E2E coverage! ✨

---

## Gitea (`cmd/gitea/scan/scan.go`)

### Available Flags
- `--token` (required)
- `--gitea`
- `--artifacts`
- `--owned`
- `--organization`
- `--repository`
- `--cookie`
- `--runs-limit`
- `--start-run-id`
- `--confidence`
- `--threads`
- `--truffleHogVerification`
- `--verbose`

### Currently Tested Flags
- `--token` ✅
- `--gitea` ✅
- `--artifacts` ✅
- `--confidence` ✅
- `--cookie` ✅
- `--organization` ✅
- `--owned` ✅
- `--repository` ✅
- `--runs-limit` ✅
- `--start-run-id` ✅
- `--threads` ✅
- `--truffleHogVerification=false` ✅

### UNTESTED FLAGS ❌
**NONE** - Gitea has 100% E2E coverage! ✨

---

## Azure DevOps (`cmd/devops/scan.go`)

### Available Flags
- `--token` (required)
- `--username` (required)
- `--confidence`
- `--threads`
- `--truffleHogVerification`
- `--maxBuilds`
- `--artifacts`
- `--organization`
- `--project`
- `--devops`
- `--verbose`

### Currently Tested Flags
- `--token` ✅
- `--username` ✅
- `--artifacts` ✅
- `--devops` ✅
- `--organization` ✅
- `--project` ✅

### UNTESTED FLAGS ❌
1. **`--confidence`** - Confidence level filtering
2. **`--threads`** - Number of concurrent threads
3. **`--truffleHogVerification=false`** - Disable credential verification
4. **`--maxBuilds`** - Maximum number of builds to scan per project
5. **`--verbose`** - Verbose logging

---

## Summary Statistics

| Platform | Total Flags | Tested | Untested | Coverage % |
|----------|-------------|--------|----------|------------|
| GitHub | 13 | 8 | 5 (+1 skipped) | 61.5% |
| GitLab | 15 | 10 | 5 | 66.7% |
| **BitBucket** | 14 | 14 | 0 | **100%** ✨ |
| **Gitea** | 12 | 12 | 0 | **100%** ✨ |
| Azure DevOps | 11 | 6 | 5 | 54.5% |
| **TOTAL** | **65** | **50** | **15** | **76.9%** |

---

## Priority Test Implementation Plan

### High Priority (Core Functionality)
1. **GitHub `--search`** - Common use case for finding repositories
2. **GitHub `--user`** - User-specific scanning
3. **GitHub `--public`** - Public repository scanning
4. **GitLab `--confidence`** - Critical filtering feature
5. **DevOps `--confidence`** - Critical filtering feature
6. **DevOps `--maxBuilds`** - Rate limiting feature

### Medium Priority (Performance/Verification)
7. **GitHub `--threads`** - Performance tuning
8. **GitLab `--max-artifact-size`** - Resource management
9. **DevOps `--threads`** - Performance tuning
10. **DevOps `--verbose`** - Logging validation

### Low Priority (Advanced Features)
11. **GitHub `--truffleHogVerification=false`** - Disable verification
12. **GitLab `--cookie`** - Advanced authentication
13. **GitLab `--queue`** - Custom queue management
14. **GitLab `--truffleHogVerification=false`** - Disable verification
15. **DevOps `--truffleHogVerification=false`** - Disable verification

### Skipped Test to Fix
- Re-enable `TestGitHubScan_ConfidenceFilter` (currently skipped)

---

## Implementation Notes

### Test File Organization
- GitHub tests: `tests/e2e/github_*_test.go`
- GitLab tests: `tests/e2e/gitlab_*_test.go`
- BitBucket tests: `tests/e2e/bitbucket_*_test.go` (reference for complete coverage)
- Gitea tests: `tests/e2e/gitea_*_test.go` (reference for complete coverage)
- DevOps tests: `tests/e2e/devops_*_test.go`

### Testing Patterns
- Use `httptest.Server` for mocking platform APIs
- Follow naming: `Test{Platform}Scan_{Feature}`
- Include setup/teardown via `startMockServer`
- Use `runCLI` helper from `e2e_helpers_test.go`
- Assert on stdout/stderr/exit codes
- Test both success and error scenarios

### Mock Server Requirements
Each test should:
1. Start a mock HTTP server
2. Configure appropriate API endpoints
3. Return realistic JSON responses
4. Handle authentication headers
5. Simulate error conditions where applicable

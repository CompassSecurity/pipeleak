# Refactoring Plan: Extract Testable Business Logic

## Overview
This document outlines the refactoring plan to extract business logic from cmd/* packages to improve testability. The goal is to separate pure business logic from I/O operations, API calls, and CLI handling.

## Refactoring Strategy

### Principles
1. **Extract pure functions**: Separate data processing logic from I/O
2. **Create internal packages**: Move extracted logic to `internal/` subpackages within each cmd
3. **Maintain backwards compatibility**: Keep existing CLI behavior unchanged
4. **Add comprehensive tests**: Write unit tests for all extracted functions

### Target Packages

---

## 1. cmd/devops/scan.go

### Current Issues
- `scanLogLines()` mixes scanning logic with logging
- `analyzeArtifact()` combines zip extraction, file type detection, and scanning
- No separation between orchestration and business logic

### Refactoring Plan

#### Extract to: `cmd/devops/internal/processor/`

**Function: `ProcessLogContent(logs []byte, verifyCredentials bool) ([]scanner.Finding, error)`**
- Pure function that takes log bytes and returns findings
- Remove direct log.Warn() calls, return findings instead
- Testable with mock log content

**Function: `ProcessArtifactZip(zipBytes []byte, artifactName string, verifyCredentials bool) ([]scanner.Finding, error)`**
- Extract zip processing logic from analyzeArtifact
- Handle zip extraction, file iteration, and scanning
- Return structured findings instead of logging directly

**Function: `ScanFileContent(content []byte, filename string, verifyCredentials bool) ([]scanner.Finding, error)`**
- Handle individual file scanning with context
- Separate file type detection from scanning

#### Tests to Add
- `processor/logs_test.go`: Test log content scanning with various inputs
- `processor/artifact_test.go`: Test zip processing with mock archives
- `processor/file_test.go`: Test file content scanning with different file types

---

## 2. cmd/bitbucket/scan.go

### Current Issues
- Pipeline/step scanning logic mixed with API pagination
- Artifact processing embedded in scan flow
- Build URL construction logic not testable

### Refactoring Plan

#### Extract to: `cmd/bitbucket/internal/processor/`

**Function: `ProcessPipelineStepLogs(logContent []byte, stepInfo StepInfo, verifyCredentials bool) ([]scanner.Finding, error)`**
- Extract log scanning logic
- StepInfo struct: workspace, repo, pipeline UUID, step UUID
- Return findings without side effects

**Function: `ProcessArtifactContent(artifactBytes []byte, artifactName string, buildInfo BuildInfo, verifyCredentials bool) ([]scanner.Finding, error)`**
- Handle artifact zip extraction and scanning
- BuildInfo struct: workspace, repo, buildNumber, stepUUID
- Separate file processing from logging

**Function: `ShouldContinueScanning(pipelineCount int, maxPipelines int) bool`**
- Extract pipeline limit checking logic
- Simple, pure function for flow control

#### Extract to: `cmd/bitbucket/internal/url/`

**Function: `BuildWebArtifactURL(workspace, repo string, buildNumber int, stepUUID string) string`**
- Already exists but could be tested separately
- Move to internal/url package

#### Tests to Add
- `processor/pipeline_test.go`: Test pipeline log processing
- `processor/artifact_test.go`: Test artifact processing with various zip structures
- `url/builder_test.go`: Test URL construction (buildWebArtifactUrl already exists, add tests)

---

## 3. cmd/github/scan.go

### Current Issues
- `deleteHighestXKeys()` already extracted (has tests ✓)
- `readZipFile()` already extracted (has tests ✓)
- Log scanning logic mixed with GitHub API calls
- Artifact processing embedded in main flow

### Refactoring Plan

#### Extract to: `cmd/github/internal/processor/`

**Function: `ProcessWorkflowLogs(logContent []byte, jobName string, verifyCredentials bool) ([]scanner.Finding, error)`**
- Extract log scanning from scanWorkflowRunLogs
- Takes raw log content, returns findings
- Separate from GitHub API interaction

**Function: `ProcessArtifactZip(zipReader *zip.Reader, artifactName string, workflowURL string, verifyCredentials bool) ([]scanner.Finding, error)`**
- Extract from analyzeArtifact
- Handle zip file iteration and scanning
- Return structured findings

**Function: `ExtractWorkflowRunsToScan(allRuns []*github.WorkflowRun, newestId, oldestId int64, maxRuns int) []*github.WorkflowRun`**
- Extract filtering logic from iterateWorkflowRuns
- Pure function for run selection
- Testable with mock workflow runs

**Function: `DetermineRunIDRange(newestId, oldestId, currentId int64) bool`**
- Extract range checking logic
- Returns whether a run should be scanned

#### Tests to Add
- `processor/workflow_test.go`: Test workflow log processing
- `processor/artifact_test.go`: Test artifact zip processing
- `processor/filter_test.go`: Test run filtering logic with various scenarios

---

## 4. cmd/gitlab/scan/queue.go

### Current Issues
- Queue item processing mixed with GitLab API calls
- Artifact analysis logic embedded in queue consumer
- No separation of concerns between queue management and processing

### Refactoring Plan

#### Extract to: `cmd/gitlab/scan/internal/processor/`

**Function: `ProcessJobTrace(traceContent []byte, jobInfo JobInfo, verifyCredentials bool) ([]scanner.Finding, error)`**
- Extract from getJobTrace flow
- JobInfo struct: projectId, jobId, jobName, jobWebUrl
- Pure function for trace processing

**Function: `ProcessJobArtifacts(artifactBytes []byte, artifactName string, jobInfo JobInfo, verifyCredentials bool) ([]scanner.Finding, error)`**
- Extract artifact processing logic
- Handle zip/gzip extraction
- Return findings without side effects

**Function: `ProcessDotenvArtifact(content []byte, projectPath string, jobId int, verifyCredentials bool) ([]scanner.Finding, error)`**
- Extract .env file processing
- Testable with mock .env content

**Function: `DetermineArtifactType(content []byte) (string, error)`**
- Extract file type detection logic
- Use filetype library, return structured result

#### Tests to Add
- `processor/trace_test.go`: Test job trace processing
- `processor/artifacts_test.go`: Test artifact processing with various formats
- `processor/dotenv_test.go`: Test .env artifact handling
- `processor/filetype_test.go`: Test file type detection

---

## 5. cmd/gitlab/runners/list.go

### Current Issues
- Runner listing mixes API calls with data transformation
- Output formatting embedded in list logic

### Refactoring Plan

#### Extract to: `cmd/gitlab/runners/internal/formatter/`

**Function: `FormatRunnerOutput(runners map[int]runnerResult) []byte`**
- Extract JSON formatting logic
- Pure function for output transformation
- Testable with mock runner data

**Function: `MergeRunnerMaps(projectRunners, groupRunners map[int]runnerResult) map[int]runnerResult`**
- Extract map merging logic
- Currently implicit, make explicit
- Test with various merge scenarios

#### Tests to Add
- `formatter/output_test.go`: Test runner output formatting
- `formatter/merge_test.go`: Test runner map merging

---

## 6. cmd/gitlab/renovate/enum.go

### Current Issues
- `extractSelfHostedOptions()` already extracted (has tests ✓)
- `validateOrderBy()` already extracted (has tests ✓)
- API validation logic could be extracted

### Refactoring Plan

#### Extract to: `cmd/gitlab/renovate/internal/validator/`

**Function: `ValidateRenovateConfig(configContent []byte) (bool, []string, error)`**
- Extract config validation logic from validateRenovateConfigService
- Return validation status and errors
- Testable with mock configs

**Function: `ParseRenovateResponse(responseBody []byte) (*RenovateValidationResult, error)`**
- Extract JSON parsing logic
- Return structured result

#### Tests to Add
- `validator/config_test.go`: Test renovate config validation
- `validator/response_test.go`: Test response parsing

---

## 7. cmd/gitea/scan/*.go

### Current Issues
- Already has good test coverage (62.1% ✓)
- HTTP utilities well separated
- Scan utilities could use more extraction

### Refactoring Plan

#### Minimal changes needed
- Current structure is relatively clean
- Add more tests for edge cases
- Extract HTML parsing to separate functions if needed

---

## Implementation Order

### Phase 1: Low-hanging fruit (Start here)
1. ✅ cmd/github/scan_test.go (already done)
2. ✅ cmd/gitlab/renovate/enum_test.go (already done)
3. ✅ cmd/gitlab/runners/exploit_test.go (already done)
4. **cmd/devops**: Simple log processing logic
5. **cmd/bitbucket**: Similar patterns to devops

### Phase 2: More complex refactoring
6. **cmd/github**: Multiple interconnected functions
7. **cmd/gitlab/scan**: Complex queue processing

### Phase 3: Formatting and utilities
8. **cmd/gitlab/runners**: Output formatting
9. **cmd/gitlab/renovate**: Additional validation logic

---

## Success Metrics

### Coverage Goals
- **cmd/devops**: 0% → 40%+
- **cmd/bitbucket**: 1.2% → 30%+
- **cmd/github**: 5.0% → 25%+
- **cmd/gitlab/scan**: Currently untested → 20%+
- **cmd/gitlab/runners**: 12.7% → 35%+
- **Overall cmd/ coverage**: 19.4% → 35%+

### Quality Metrics
- All new code passes golangci-lint
- All tests use table-driven test pattern
- 100% of extracted functions have unit tests
- No regressions in E2E tests

---

## Risk Mitigation

### Approach
1. **Extract, don't rewrite**: Copy logic first, refactor incrementally
2. **Test before refactoring**: Ensure E2E tests pass before starting
3. **One package at a time**: Complete refactoring and tests per package before moving on
4. **Verify no behavior changes**: Run E2E tests after each package refactoring

### Rollback Plan
- Each package refactored in separate commit
- Easy to revert individual changes if needed
- Keep original functions initially, mark as deprecated

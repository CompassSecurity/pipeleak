# End-to-End Tests for Pipeleak CLI

This directory contains comprehensive end-to-end tests for the Pipeleak CLI tool.

## Overview

The e2e tests validate the CLI by:
- Running commands programmatically (in-process via `cmd.Execute()`)
- Using mock HTTP servers (`httptest.Server`) for all external API calls
- Capturing and asserting on stdout, stderr, and exit codes
- Validating request/response behavior, flags, and error handling

## Running Tests

**IMPORTANT**: By default, the e2e tests run in "framework demonstration mode" and will skip actual CLI execution. To run tests against the real CLI:

1. Edit `tests/e2e/cli_integration.go` 
2. Change `const useLiveExecution = false` to `const useLiveExecution = true`
3. Run the tests

### Run All E2E Tests

```bash
cd /workspaces/pipeleak/src/pipeleak
go test ./tests/e2e/... -v
```

### Enable Live CLI Execution

```bash
# Edit the file
sed -i 's/useLiveExecution = false/useLiveExecution = true/' tests/e2e/cli_integration.go

# Run tests with actual CLI
go test ./tests/e2e/... -v
```

### Run a Specific Test

```bash
go test ./tests/e2e/... -v -run TestGitLabScan
```

### Run Tests with Coverage

```bash
go test ./tests/e2e/... -v -cover -coverprofile=coverage.out
go tool cover -html=coverage.out
```

### Run Tests in Parallel (carefully)

Most tests use `t.Parallel()` where safe. To control parallelism:

```bash
go test ./tests/e2e/... -v -parallel 4
```

## Test Structure

- `e2e_helpers_test.go` - Common test utilities and mock server helpers
- `gitlab_test.go` - GitLab command tests (scan, enum, variables, etc.)
- `gitea_test.go` - Gitea command tests (scan, enum)
- `github_test.go` - GitHub command tests (scan)
- `bitbucket_test.go` - BitBucket command tests (scan)
- `devops_test.go` - Azure DevOps command tests (scan)
- `root_test.go` - Root command and global flag tests

## Writing New Tests

### Basic Test Pattern

```go
func TestMyCommand(t *testing.T) {
    t.Parallel() // Only if test is independent
    
    // Setup mock server
    server, requests, cleanup := startMockServer(t, func(w http.ResponseWriter, r *http.Request) {
        // Handle requests and return responses
        w.WriteHeader(http.StatusOK)
        json.NewEncoder(w).Encode(myResponse)
    })
    defer cleanup()
    
    // Run CLI command
    stdout, stderr, exitErr := runCLI(t, []string{
        "mycommand",
        "--flag", "value",
        "--server", server.URL,
    }, nil, 5*time.Second)
    
    // Assert results
    assert.Nil(t, exitErr, "Command should succeed")
    assertLogContains(t, stdout, []string{"expected", "output"})
}
```

### Mock Server Best Practices

1. **Always use `httptest.Server`** - Never use fixed ports
2. **Record requests** - Use the `RecordedRequest` slice returned by `startMockServer`
3. **Use timeouts** - Pass reasonable timeout to `runCLI` to prevent hangs
4. **Close servers** - Always `defer cleanup()` after creating mock server

### Debugging Failed Tests

When a test fails, the output will include:
- Captured stdout and stderr from the CLI
- All HTTP requests received by the mock server
- Detailed diffs between expected and actual output

To enable verbose logging in tests:

```bash
go test ./tests/e2e/... -v -run TestMyCommand
```

## Test Coverage

The e2e tests cover:

### GitLab Commands
- ✅ `gl scan` - Pipeline and artifact scanning
- ✅ `gl enum` - User and group enumeration
- ✅ `gl variables` - CI/CD variable extraction
- ✅ `gl vuln` - Vulnerability scanning
- ✅ `gl runners list` - Runner enumeration
- ✅ `gl cicd yaml` - CI/CD configuration fetching
- ✅ `gl schedule` - Scheduled pipeline enumeration
- ✅ `gl secureFiles` - Secure files extraction
- ✅ `gluna register` - Unauthenticated runner registration

### Gitea Commands
- ✅ `gitea scan` - Actions workflow scanning
- ✅ `gitea enum` - Repository enumeration

### GitHub Commands
- ✅ `gh scan` - Actions workflow scanning

### BitBucket Commands
- ✅ `bb scan` - Pipeline scanning

### Azure DevOps Commands
- ✅ `ad scan` - Pipeline scanning

### Global Tests
- ✅ `--help` flag
- ✅ `--version` flag
- ✅ JSON log output (`--json`)
- ✅ Log file output (`--logfile`)
- ✅ Error handling (invalid URLs, missing flags, API errors)

## Updating Tests

### When Adding New CLI Commands

1. Add command test to appropriate `*_test.go` file
2. Create mock server handler for API endpoints
3. Test happy path + error scenarios
4. Test all flags (required and optional)
5. Verify request assertions (method, headers, body)

### When Changing API Behavior

1. Update mock server responses in affected tests
2. Update assertions to match new behavior
3. Add regression tests if fixing a bug

## CI/CD Integration

These tests are designed to run in CI/CD with:
- No external dependencies (all APIs mocked)
- Deterministic output (no random ports, timestamps handled)
- Fast execution (in-process CLI calls)
- Clear failure diagnostics

## Common Issues

### Tests Hanging

- Check timeout values in `runCLI` calls
- Ensure mock servers are properly closed with `defer cleanup()`
- Verify no goroutines are leaking (use `go test -race`)

### Flaky Tests

- Avoid `t.Parallel()` if tests share global state
- Use `httptest.Server` not fixed ports
- Clean up temp files/dirs properly

### Port Already in Use

- If you see "address already in use", you're not using `httptest.Server`
- Fix by using the `startMockServer` helper which auto-assigns ports

## Additional Resources

- [Go Testing Guide](https://golang.org/pkg/testing/)
- [httptest Package](https://golang.org/pkg/net/http/httptest/)
- [Cobra Testing](https://github.com/spf13/cobra/blob/main/command_test.go)

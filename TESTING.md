# Testing Guide for Pipeleak

This document describes how to run tests and contribute test coverage to the Pipeleak project.

## Prerequisites

- Go 1.24.0 or later
- golangci-lint (latest version recommended)

## Test Structure

Pipeleak uses two types of tests:

### Unit Tests
Unit tests are located alongside the source code in `*_test.go` files. They test individual functions and components in isolation using mock objects and interfaces.

**Location:** `src/pipeleak/cmd/*/` and `src/pipeleak/pkg/*/`

### E2E (End-to-End) Tests
E2E tests validate complete user workflows by executing the CLI binary with mock HTTP servers simulating external APIs.

**Location:** `src/pipeleak/tests/e2e/`

**Important:** All E2E tests use mock servers (`httptest.Server`) and do NOT require real external services.

## Running Tests

### Run All Unit Tests

```bash
cd src/pipeleak
go test $(go list ./... | grep -v /tests/e2e) -v
```

### Run Specific Unit Test Package

```bash
cd src/pipeleak
go test ./cmd/github/... -v
go test ./pkg/scanner/... -v
```

### Run All E2E Tests

```bash
cd src/pipeleak
go test ./tests/e2e/... -v -timeout 15m
```

The E2E tests will automatically build a test binary. You can also specify a pre-built binary:

```bash
cd src/pipeleak
go build -o pipeleak .
PIPELEAK_BINARY=./pipeleak go test ./tests/e2e/... -v -timeout 15m
```

### Run Specific Test

```bash
cd src/pipeleak
go test ./cmd/github/... -run TestDeleteHighestXKeys -v
go test ./tests/e2e/... -run TestGitLabScan -v
```

### Run Tests with Race Detector

```bash
cd src/pipeleak
go test $(go list ./... | grep -v /tests/e2e) -v -race
```

### Run Tests with Coverage

```bash
cd src/pipeleak
go test ./... -coverprofile=coverage.out
go tool cover -html=coverage.out
```

## Linting

### Run golangci-lint

```bash
cd src/pipeleak
golangci-lint run --timeout=10m
```

### Install golangci-lint

```bash
# Binary will be $(go env GOPATH)/bin/golangci-lint
curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(go env GOPATH)/bin latest
```

## Writing Tests

### Test Guidelines

1. **Explicit Assertions**: Every test must have explicit assertions that verify real behavior, not just setup or error conditions.

2. **Table-Driven Tests**: Use table-driven tests when testing multiple scenarios:

```go
func TestMyFunction(t *testing.T) {
    tests := []struct {
        name     string
        input    string
        expected string
    }{
        {name: "empty input", input: "", expected: ""},
        {name: "normal case", input: "test", expected: "TEST"},
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            result := MyFunction(tt.input)
            assert.Equal(t, tt.expected, result)
        })
    }
}
```

3. **Error Path Testing**: Always test both success and error paths:

```go
func TestWithError(t *testing.T) {
    t.Run("success case", func(t *testing.T) {
        result, err := FunctionThatCanFail(validInput)
        assert.NoError(t, err)
        assert.NotNil(t, result)
    })
    
    t.Run("error case", func(t *testing.T) {
        result, err := FunctionThatCanFail(invalidInput)
        assert.Error(t, err)
        assert.Nil(t, result)
    })
}
```

4. **Mock Servers for E2E**: E2E tests must always use mock HTTP servers:

```go
server, getRequests, cleanup := startMockServer(t, func(w http.ResponseWriter, r *http.Request) {
    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(http.StatusOK)
    json.NewEncoder(w).Encode(mockResponse)
})
defer cleanup()
```

5. **Avoid Useless Tests**: Do not write tests that only check `NotPanics` without verifying actual behavior. Tests should validate:
   - Return values are correct
   - Side effects occur as expected
   - Error conditions are properly handled

### Example: Good vs Bad Tests

**Bad Test** (only checks it doesn't panic):
```go
func TestBad(t *testing.T) {
    assert.NotPanics(t, func() {
        ProcessData(input)
    })
}
```

**Good Test** (verifies actual behavior):
```go
func TestGood(t *testing.T) {
    result := ProcessData(input)
    assert.NotNil(t, result)
    assert.Equal(t, expected, result.Value)
    assert.NoError(t, result.Error)
}
```

## CI/CD Integration

### GitHub Actions Workflows

All tests are automatically run on:
- Push to `main` branch
- Pull requests

Workflows include:
- **Unit Tests** (`.github/workflows/test.yml`): Runs on Linux and Windows
- **E2E Tests** (`.github/workflows/test.yml`): Runs on Linux and Windows
- **Linting** (`.github/workflows/golangci-lint.yml`): Runs golangci-lint

### Required Checks

Before merging code, ensure:
1. ✅ All unit tests pass
2. ✅ All E2E tests pass
3. ✅ golangci-lint passes with 0 issues
4. ✅ No race conditions detected

## Debugging Failed Tests

### View Detailed Test Output

```bash
go test ./... -v -run FailingTest
```

### Run with Debug Logging

```bash
go test ./... -v -run FailingTest 2>&1 | tee test.log
```

### E2E Test Debugging

E2E tests capture CLI output. Use `t.Logf()` to inspect:

```go
stdout, stderr, err := runCLI(t, args, nil, 30*time.Second)
t.Logf("STDOUT:\n%s", stdout)
t.Logf("STDERR:\n%s", stderr)
```

## Common Issues

### Tests Timing Out

If tests timeout, increase the timeout:
```bash
go test ./tests/e2e/... -timeout 30m -v
```

### E2E Binary Not Found

E2E tests build the binary automatically. If you see errors, ensure:
- You're running from `src/pipeleak` directory
- Go modules are properly initialized (`go mod download`)

### Race Conditions

If race detector finds issues:
```bash
go test -race ./... -v
```

Fix all race conditions before submitting code.

## Contributing Tests

When adding new features:

1. **Add unit tests** for all new functions
2. **Add E2E tests** for new CLI commands or workflows
3. **Update this guide** if introducing new test patterns
4. **Ensure 100% of new code** has test coverage for critical paths
5. **Run all tests locally** before creating a pull request

## Performance Testing

For performance-sensitive code:

```bash
go test -bench=. -benchmem ./pkg/scanner/...
```

## Test Dependencies

Tests use the following libraries:
- `github.com/stretchr/testify/assert` - Assertions
- `github.com/stretchr/testify/require` - Required assertions (fail fast)
- `net/http/httptest` - HTTP mock servers
- Standard library `testing` package

## Questions?

For questions about testing, please:
1. Review existing tests as examples
2. Check this guide
3. Open an issue if something is unclear

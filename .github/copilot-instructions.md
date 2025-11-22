# GitHub Copilot Instructions for Pipeleak

## Project Overview

Pipeleak is a CLI tool designed to scan CI/CD logs and artifacts for secrets across multiple platforms including GitLab, GitHub, BitBucket, Azure DevOps, and Gitea. The tool uses TruffleHog for secret detection and provides additional helper commands for exploitation workflows.

## Technology Stack

- **Language**: Go 1.24+
- **CLI Framework**: Cobra (github.com/spf13/cobra)
- **Logging**: Zerolog (github.com/rs/zerolog)
- **Secret Detection**: TruffleHog v3
- **Testing**: Go testing framework with testify
- **Build Tool**: Go build system
- **Release**: GoReleaser

## Project Structure

```
pipeleak/
├── src/pipeleak/           # Main Go module
│   ├── cmd/                # CLI commands (using Cobra)
│   │   ├── bitbucket/      # BitBucket-specific commands
│   │   ├── devops/         # Azure DevOps commands
│   │   ├── docs/           # Documentation command
│   │   ├── gitea/          # Gitea commands
│   │   ├── github/         # GitHub-specific commands
│   │   ├── gitlab/         # GitLab-specific commands
│   ├── pkg/                # Core business logic packages
│   │   ├── archive/        # Archive handling
│   │   ├── bitbucket/      # BitBucket business logic
│   │   ├── config/         # Configuration management
│   │   ├── devops/         # Azure DevOps business logic
│   │   ├── docs/           # Documentation generation
│   │   ├── format/         # Formatting helpers
│   │   ├── gitea/          # Gitea business logic
│   │   ├── github/         # GitHub business logic
│   │   ├── gitlab/         # GitLab business logic
│   │   ├── httpclient/     # HTTP client helpers
│   │   ├── logging/        # Logging helpers
│   │   ├── scan/           # Scan logic
│   │   ├── scanner/        # Scanner engine
│   │   ├── system/         # System helpers
│   ├── tests/              # Test files
│   │   └── e2e/            # End-to-end tests
│   ├── main.go             # Application entry point
│   ├── go.mod              # Go module definition
│   └── go.sum              # Dependency checksums
├── docs/                   # Documentation (MkDocs)
├── .github/                # GitHub workflows and configs
│   └── workflows/          # CI/CD pipelines
└── goreleaser.yaml         # Release configuration
```

## Building and Testing

### Building the Project

```bash
cd src/pipeleak
go build -o pipeleak .
```

### Running Tests

**Unit tests (excluding e2e):**
```bash
cd src/pipeleak
go test $(go list ./... | grep -v /tests/e2e) -v -race
```

**End-to-end tests:**
```bash
cd src/pipeleak
go build -o pipeleak .
PIPELEAK_BINARY=./pipeleak go test ./tests/e2e/... -v -timeout 10m
```

### Linting

The project uses golangci-lint:
```bash
cd src/pipeleak
golangci-lint run --timeout=10m
```

## Code Style and Conventions

### General Guidelines

1. **Follow Go idioms**: Use standard Go conventions and patterns
2. **Error handling**: Always check and handle errors appropriately
3. **Logging**: Use zerolog for structured logging with appropriate levels (trace, debug, info, warn, error, fatal)
4. **Testing**: Write tests for new functionality; maintain existing test coverage
5. **Documentation**: Update documentation when adding or modifying features
6. **Comments**: Only add comments that provide useful context and additional understanding; avoid obvious or redundant comments
7. **File Moves/Copies**: When moving or copying files, always delete any resulting unused or vestigial files to keep the codebase clean and maintainable.

### Command Structure

- Commands follow the Cobra pattern with `NewXCommand()` functions
- Each command should have a corresponding test file
- Commands are organized by platform (gitlab, github, bitbucket, devops, gitea)
- Use consistent flag naming across commands

### Package Organization

- Keep business logic in `pkg/` packages
- Keep CLI interface code in `cmd/` packages
- Separate concerns: commands orchestrate, packages implement

### Testing Conventions

- Test files should be named `*_test.go`
- Use table-driven tests where appropriate
- Use testify/assert for assertions
- E2E tests go in `tests/e2e/`
- Mock external dependencies in unit tests

### Logging Best Practices

- Use appropriate log levels:
  - `trace`: Very detailed diagnostic information
  - `debug`: Detailed information for debugging
  - `info`: General informational messages (default)
  - `warn`: Warning messages
  - `error`: Error conditions
  - `fatal`: Fatal errors that require program termination
  - `hit`: Special log level used exclusively for logging detected secrets
- Use structured logging with fields: `log.Info().Str("key", "value").Msg("message")`
- Log context-relevant information to aid debugging

## Dependencies

### Adding New Dependencies

1. Use `go get` to add dependencies:
   ```bash
   cd src/pipeleak
   go get github.com/example/package@version
   ```

2. Run `go mod tidy` to clean up:
   ```bash
   go mod tidy
   ```

3. Update go.sum by running tests or building:
   ```bash
   go build ./...
   ```

### Key Dependencies

- `github.com/spf13/cobra`: CLI framework
- `github.com/rs/zerolog`: Structured logging
- `github.com/trufflesecurity/trufflehog/v3`: Secret detection
- `github.com/google/go-github/v69`: GitHub API client
- `gitlab.com/gitlab-org/api/client-go`: GitLab API client
- `code.gitea.io/sdk/gitea`: Gitea API client

## Common Development Tasks

### Adding a New Command

1. Create command file in appropriate `cmd/<platform>/` directory
2. Implement command using Cobra patterns
3. Add corresponding business logic in `pkg/<platform>/`
4. Write tests for both command and business logic
5. Update documentation if needed

### Adding a New Platform

1. Create new directory under `cmd/<platform>/`
2. Create corresponding package under `pkg/<platform>/`
3. Implement scan and other relevant commands
4. Add tests
5. Update documentation

### Modifying Secret Detection

- Secret detection is handled by TruffleHog
- Custom rules can be defined in `rules.yml` (user-generated)
- Confidence levels: low, medium, high, high-verified
- Verification can be disabled with `--truffle-hog-verification=false`

## CI/CD

The project uses GitHub Actions for CI/CD:

- **test.yml**: Runs unit and e2e tests on Linux and Windows
- **golangci-lint.yml**: Runs linting checks
- **release.yml**: Builds and publishes releases using GoReleaser
- **docs.yml**: Builds and deploys documentation

## Important Notes

1. **Working Directory**: The Go module is in `src/pipeleak/`, not the repository root
2. **Binary Names**: 
   - Linux/macOS: `pipeleak`
   - Windows: `pipeleak.exe`
3. **Test Exclusions**: E2E tests are excluded from regular test runs
4. **Terminal State**: The application manages terminal state for interactive features
5. **Cross-Platform**: Code should work on Linux, macOS, and Windows

## Additional Resources

- [Getting Started Guide](https://compasssecurity.github.io/pipeleak/introduction/getting_started/)
- [GitHub Repository](https://github.com/CompassSecurity/pipeleak)
- [TruffleHog Documentation](https://github.com/trufflesecurity/trufflehog)

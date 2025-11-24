.PHONY: help build test test-unit test-e2e lint clean

# Default target
help:
	@echo "Pipeleak Makefile"
	@echo ""
	@echo "Available targets:"
	@echo "  make build        - Build the pipeleak binary"
	@echo "  make test         - Run all tests (unit + e2e)"
	@echo "  make test-unit    - Run unit tests only"
	@echo "  make test-e2e     - Run e2e tests (builds binary first)"
	@echo "  make lint         - Run golangci-lint"
	@echo "  make clean        - Remove built artifacts"

# Build the pipeleak binary
build:
	@echo "Building pipeleak..."
	go build -o pipeleak .

# Run all tests
test: test-unit test-e2e

# Run unit tests (excluding e2e)
test-unit:
	@echo "Running unit tests..."
	go test $$(go list ./... | grep -v /tests/e2e) -v -race

# Run e2e tests (builds binary first)
test-e2e: build
	@echo "Running e2e tests..."
	PIPELEAK_BINARY=$$(pwd)/pipeleak go test ./tests/e2e/... -tags=e2e -v -timeout 10m

# Run e2e tests for specific platform
test-e2e-gitlab: build
	@echo "Running GitLab e2e tests..."
	PIPELEAK_BINARY=$$(pwd)/pipeleak go test ./tests/e2e/gitlab/... -tags=e2e -v

test-e2e-github: build
	@echo "Running GitHub e2e tests..."
	PIPELEAK_BINARY=$$(pwd)/pipeleak go test ./tests/e2e/github/... -tags=e2e -v

test-e2e-bitbucket: build
	@echo "Running BitBucket e2e tests..."
	PIPELEAK_BINARY=$$(pwd)/pipeleak go test ./tests/e2e/bitbucket/... -tags=e2e -v

test-e2e-devops: build
	@echo "Running Azure DevOps e2e tests..."
	PIPELEAK_BINARY=$$(pwd)/pipeleak go test ./tests/e2e/devops/... -tags=e2e -v

test-e2e-gitea: build
	@echo "Running Gitea e2e tests..."
	PIPELEAK_BINARY=$$(pwd)/pipeleak go test ./tests/e2e/gitea/... -tags=e2e -v

# Run golangci-lint
lint:
	@echo "Running golangci-lint..."
	golangci-lint run --timeout=10m

# Clean built artifacts
clean:
	@echo "Cleaning..."
	rm -f pipeleak pipeleak.exe
	go clean -cache -testcache

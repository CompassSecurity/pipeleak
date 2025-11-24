.PHONY: help build test test-unit test-e2e lint clean coverage coverage-html

# Default target
help:
	@echo "Pipeleak Makefile"
	@echo ""
	@echo "Available targets:"
	@echo "  make build        - Build the pipeleak binary"
	@echo "  make test         - Run all tests (unit + e2e)"
	@echo "  make test-unit    - Run unit tests only"
	@echo "  make test-e2e     - Run e2e tests (builds binary first)"
	@echo "  make coverage     - Generate test coverage report"
	@echo "  make coverage-html - Generate and open HTML coverage report"
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

# Generate test coverage report
coverage:
	@echo "Generating coverage report..."
	@go test $$(go list ./... | grep -v /tests/e2e) -coverprofile=coverage.out -covermode=atomic
	@echo ""
	@echo "Coverage Summary:"
	@go tool cover -func=coverage.out | grep total | awk '{print "Total Coverage: " $$3}'
	@echo ""
	@echo "Coverage report saved to coverage.out"
	@echo "Run 'make coverage-html' to view detailed HTML report"

# Generate and open HTML coverage report
coverage-html: coverage
	@echo "Generating HTML coverage report..."
	@go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report saved to coverage.html"
	@if command -v xdg-open > /dev/null; then \
		xdg-open coverage.html; \
	elif command -v open > /dev/null; then \
		open coverage.html; \
	else \
		echo "Open coverage.html in your browser to view the report"; \
	fi

# Run golangci-lint
lint:
	@echo "Running golangci-lint..."
	golangci-lint run --timeout=10m

# Clean up built artifacts
clean:
	@echo "Cleaning up..."
	rm -f pipeleak pipeleak.exe coverage.out coverage.html
	go clean -cache -testcache

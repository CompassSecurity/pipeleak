.PHONY: help build test test-unit test-e2e lint clean install-tools

# Default target
help:
@echo "Available targets:"
@echo "  build         - Build the pipeleak binary"
@echo "  test          - Run all tests (unit + e2e)"
@echo "  test-unit     - Run unit tests only"
@echo "  test-e2e      - Run end-to-end tests only"
@echo "  lint          - Run golangci-lint"
@echo "  clean         - Remove build artifacts"
@echo "  install-tools - Install development tools"

# Build the binary
build:
@echo "Building pipeleak..."
@go build -o pipeleak .

# Run all tests
test:
@echo "Running all tests..."
@go test ./... -v -race

# Run unit tests only (exclude e2e tests)
test-unit:
@echo "Running unit tests..."
@go test $$(go list ./... | grep -v /tests/e2e) -v -race

# Run e2e tests only
test-e2e: build
@echo "Running e2e tests..."
@PIPELEAK_BINARY=./pipeleak go test ./tests/e2e/... -v -timeout 10m

# Run linter
lint:
@echo "Running golangci-lint..."
@golangci-lint run --timeout=10m

# Clean build artifacts
clean:
@echo "Cleaning build artifacts..."
@rm -f pipeleak pipeleak.exe
@go clean

# Install development tools
install-tools:
@echo "Installing golangci-lint..."
@go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest

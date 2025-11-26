#!/bin/bash
set -e

echo "=== Pipeleak Development Environment Setup ==="

# Verify Docker is available
echo "Checking Docker availability..."
if command -v docker &> /dev/null; then
    echo "Docker is available: $(docker --version)"
    # Wait for Docker daemon to be ready
    timeout=30
    while ! docker info &> /dev/null && [ $timeout -gt 0 ]; do
        echo "Waiting for Docker daemon to start... ($timeout seconds remaining)"
        sleep 1
        timeout=$((timeout - 1))
    done
    if docker info &> /dev/null; then
        echo "Docker daemon is running"
    else
        echo "Warning: Docker daemon is not responding"
    fi
else
    echo "Warning: Docker is not installed"
fi

# Install golangci-lint
echo "Installing golangci-lint..."
go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest

# Install Python dependencies for documentation
echo "Installing MkDocs and dependencies..."
pip install --user mkdocs mkdocs-material mkdocs-minify-plugin

# Download Go dependencies
echo "Downloading Go dependencies..."
cd src/pipeleak
go mod download

# Build the binary
echo "Building pipeleak binary..."
go build -o pipeleak .

# Setup bash aliases
echo "Setting up bash aliases..."
cat >> ~/.bashrc << 'EOF'

# Custom aliases
alias ll='ls -alh'
alias la='ls -A'
alias l='ls -CF'

# Git shortcuts
alias gs='git status'
alias ga='git add'
alias gc='git commit'
alias gp='git push'
alias gl='git pull'
alias gd='git diff'
alias gco='git checkout'
alias gb='git branch'
alias glog='git log --oneline --graph --decorate'
EOF

echo ""
echo "=== Setup Complete ==="
echo ""
echo "Quick start commands:"
echo "  cd src/pipeleak"
echo "  make build       - Build the binary"
echo "  make test-unit   - Run unit tests"
echo "  make lint        - Run linter"
echo "  make serve-docs  - Generate and serve documentation"
echo ""
echo "Docker is available for testing containerized workflows."
echo "Run './pipeleak --help' to see available commands."

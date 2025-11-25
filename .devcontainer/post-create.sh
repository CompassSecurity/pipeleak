#!/bin/bash
set -e

echo "=== Pipeleak Development Environment Setup ==="

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
echo "Run './pipeleak --help' to see available commands."

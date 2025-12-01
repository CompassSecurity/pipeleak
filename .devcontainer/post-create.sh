#!/bin/bash
set -e

echo "=== Pipeleak Development Environment Setup ==="

# Verify Docker is available
echo "Checking Docker availability..."
if command -v docker &> /dev/null; then
    echo "Docker is available: $(docker --version)"
    if docker info &> /dev/null 2>&1; then
        echo "Docker daemon is ready"
    else
        echo "Warning: Docker daemon may not be responding"
    fi
else
    echo "Warning: Docker is not installed"
fi

# Create .bash_profile to source .bashrc for login shells
echo "Setting up .bash_profile..."
if [ ! -f ~/.bash_profile ]; then
    cat > ~/.bash_profile << 'EOF'
# Source .bashrc for login shells
if [ -f ~/.bashrc ]; then
    . ~/.bashrc
fi
EOF
    echo "Created ~/.bash_profile"
fi

# Install golangci-lint
echo "Installing golangci-lint..."
go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest

# Install Python dependencies for documentation
echo "Installing MkDocs and dependencies..."
pip install --user mkdocs mkdocs-material mkdocs-minify-plugin

# Download Go dependencies
echo "Downloading Go dependencies..."
go mod download

# Setup bash aliases
echo "Setting up bash aliases..."
if ! grep -q "# Pipeleak custom aliases" ~/.bashrc; then
    cat >> ~/.bashrc << 'EOF'

# Pipeleak custom aliases
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
fi

# Source bashrc to make aliases available immediately
source ~/.bashrc

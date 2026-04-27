#!/bin/bash

# OpenCode Goal Tracker Installation Script
# Installs the goal tracking system for OpenCode

set -e

SCRIPT_NAME="OpenCode Goal Tracker"
REPO_URL="https://gitlab.com/sig/opengoal"  # Update with your GitLab URL
VERSION="1.0.0"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Detect OS and architecture
OS=$(uname -s | tr '[:upper:]' '[:lower:]')
ARCH=$(uname -m)

case "$ARCH" in
    x86_64)
        ARCH="amd64"
        ;;
    aarch64|arm64)
        ARCH="arm64"
        ;;
    *)
        echo -e "${RED}Unsupported architecture: $ARCH${NC}"
        exit 1
        ;;
esac

case "$OS" in
    darwin)
        OS="darwin"
        ;;
    linux)
        OS="linux"
        ;;
    mingw*|msys*|cygwin*)
        OS="windows"
        ;;
    *)
        echo -e "${RED}Unsupported OS: $OS${NC}"
        exit 1
        ;;
esac

echo -e "${BLUE}╔════════════════════════════════════════════════╗${NC}"
echo -e "${BLUE}║  Installing ${SCRIPT_NAME} v${VERSION}${NC}"
echo -e "${BLUE}╚════════════════════════════════════════════════╝${NC}"
echo ""
echo -e "Detected: ${GREEN}${OS}/${ARCH}${NC}"
echo ""

# Check if OpenCode config directory exists
if [ ! -d "$HOME/.config/opencode" ]; then
    echo -e "${YELLOW}⚠️  OpenCode config directory not found.${NC}"
    echo "This tool requires OpenCode to be installed."
    echo ""
    read -p "Create OpenCode config directory? (y/n) " -n 1 -r
    echo
    if [[ $REPLY =~ ^[Yy]$ ]]; then
        mkdir -p "$HOME/.config/opencode"
        echo -e "${GREEN}✓ Created OpenCode config directory${NC}"
    else
        echo -e "${RED}Installation cancelled.${NC}"
        exit 1
    fi
fi

# Create directories
echo "Creating directories..."
mkdir -p "$HOME/.config/opencode/skills/goal-tracker"
mkdir -p "$HOME/.config/opencode/commands"
mkdir -p "$HOME/.local/share/opencode"

# Install from local directory or download from GitHub
if [ -f "main.go" ]; then
    echo -e "${BLUE}Installing from local directory...${NC}"
    
    # Check if Go is installed
    if ! command -v go &> /dev/null; then
        echo -e "${RED}Error: Go is not installed. Please install Go first.${NC}"
        echo "Visit: https://golang.org/doc/install"
        exit 1
    fi
    
    # Build the binary
    echo "Building Go binary..."
    go build -o "$HOME/.config/opencode/skills/goal-tracker/goals" main.go
    echo -e "${GREEN}✓ Built binary${NC}"
    
    # Copy files
    cp main.go "$HOME/.config/opencode/skills/goal-tracker/"
    cp go.mod "$HOME/.config/opencode/skills/goal-tracker/"
    cp SKILL.md "$HOME/.config/opencode/skills/goal-tracker/"
    [ -f README_GO.md ] && cp README_GO.md "$HOME/.config/opencode/skills/goal-tracker/"
    [ -f GETTING_STARTED.md ] && cp GETTING_STARTED.md "$HOME/.config/opencode/skills/goal-tracker/"
    
    # Copy command files
    cp commands/*.md "$HOME/.config/opencode/commands/" 2>/dev/null || true
    
else
    echo -e "${BLUE}Downloading from GitHub...${NC}"
    
    # Download binary for platform
    BINARY_URL="${REPO_URL}/releases/download/v${VERSION}/goals-${OS}-${ARCH}"
    
    if command -v curl &> /dev/null; then
        curl -L -o "$HOME/.config/opencode/skills/goal-tracker/goals" "$BINARY_URL"
    elif command -v wget &> /dev/null; then
        wget -O "$HOME/.config/opencode/skills/goal-tracker/goals" "$BINARY_URL"
    else
        echo -e "${RED}Error: Neither curl nor wget is available.${NC}"
        exit 1
    fi
    
    # Download other files
    echo "Downloading configuration files..."
    # TODO: Download skill file, commands, etc.
fi

# Make binary executable
chmod +x "$HOME/.config/opencode/skills/goal-tracker/goals"
echo -e "${GREEN}✓ Made binary executable${NC}"

# Initialize goals.json if it doesn't exist
if [ ! -f "$HOME/.local/share/opencode/goals.json" ]; then
    echo "Initializing goals database..."
    "$HOME/.config/opencode/skills/goal-tracker/goals" list > /dev/null 2>&1 || true
    echo -e "${GREEN}✓ Initialized goals database${NC}"
fi

# Test installation
echo ""
echo "Testing installation..."
if "$HOME/.config/opencode/skills/goal-tracker/goals" today > /dev/null 2>&1; then
    echo -e "${GREEN}✓ Installation successful!${NC}"
else
    echo -e "${RED}✗ Installation test failed${NC}"
    exit 1
fi

echo ""
echo -e "${GREEN}╔════════════════════════════════════════════════╗${NC}"
echo -e "${GREEN}║  🎉 Installation Complete!${NC}"
echo -e "${GREEN}╚════════════════════════════════════════════════╝${NC}"
echo ""
echo -e "${BLUE}Quick Start:${NC}"
echo "  1. Restart OpenCode (or reload config)"
echo "  2. Run: ${GREEN}/today${NC} to see your dashboard"
echo "  3. Run: ${GREEN}/goals-main <title>${NC} to add a goal"
echo "  4. Run: ${GREEN}/task-add <title>${NC} to add a task"
echo ""
echo -e "${BLUE}Documentation:${NC}"
echo "  ~/.config/opencode/skills/goal-tracker/GETTING_STARTED.md"
echo ""
echo -e "${BLUE}Available Commands:${NC}"
echo "  /today           - Show dashboard"
echo "  /goals-main      - Add main goal"
echo "  /goals-list      - List goals"
echo "  /task-add        - Add task"
echo "  /task-list       - List tasks"
echo "  /goals-summary   - Daily summary"
echo ""
echo -e "Visit ${BLUE}${REPO_URL}${NC} for more info"
echo ""

#!/usr/bin/env bash
set -e

echo "Setting up test environment with project p command..."

# Build the project first
make build

# Create a temporary test environment
TEST_DIR=$(mktemp -d)
echo "Test environment: $TEST_DIR"

# Create test project structure
mkdir -p "$TEST_DIR/code/user1/project1"
mkdir -p "$TEST_DIR/code/user1/project2" 
mkdir -p "$TEST_DIR/code/user2/project3"

# Initialize git repos
(cd "$TEST_DIR/code/user1/project1" && git init --quiet)
(cd "$TEST_DIR/code/user1/project2" && git init --quiet)
(cd "$TEST_DIR/code/user2/project3" && git init --quiet)

# Create config file
cat > "$TEST_DIR/.projectrc" <<EOF
root = "$TEST_DIR/code"
user = "testuser"
EOF

# Generate project init script
./build/proj init zsh > "$TEST_DIR/project.zsh"

# Create zshrc
cat > "$TEST_DIR/.zshrc" <<EOF
# Test zshrc for project
export PROJECT_CONFIG_FILE="$TEST_DIR/.projectrc"
source "$TEST_DIR/project.zsh"

# Welcome message
echo "🚀 Project test environment loaded!"
echo "Available test projects:"
echo "  - user1/project1"
echo "  - user1/project2" 
echo "  - user2/project3"
echo ""
echo "Try: p project1, p proj2, p user2/proj3"
echo "Tab completion should work!"
echo ""
EOF

# Export environment and start zsh
export ZDOTDIR="$TEST_DIR"
export PROJECT_CONFIG_FILE="$TEST_DIR/.projectrc"

echo "Starting zsh with project p command loaded..."
echo "Type 'exit' when done to cleanup."

# Start zsh
zsh

# Cleanup
echo "Cleaning up test environment..."
rm -rf "$TEST_DIR"
echo "Done!"
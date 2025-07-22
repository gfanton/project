#!/usr/bin/env bash
set -euo pipefail

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Check if we're in nix-shell
if [ -z "${IN_NIX_SHELL:-}" ]; then
    echo -e "${YELLOW}Warning: Not running in nix-shell. Some tests may fail.${NC}"
    echo "Run 'nix-shell' first or use 'make test-nix'"
fi

# Build the project first
echo "Building project..."
make build

# Run unit tests
echo -e "\n${GREEN}Running Go unit tests...${NC}"
make test

# Run integration tests
echo -e "\n${GREEN}Running shell integration tests...${NC}"

# BATS unit tests for tmux integration
if command -v bats >/dev/null 2>&1; then
    echo "Running BATS unit tests..."
    if [[ -d "tests/unit" && $(find tests/unit -name "*.bats" | wc -l) -gt 0 ]]; then
        bats tests/unit/
    else
        echo "No BATS unit tests found in tests/unit/"
    fi
    
    echo "Running BATS integration tests..."
    if [[ -f "tests/integration/zsh_test.sh" ]]; then
        bats tests/integration/zsh_test.sh
    else
        echo "zsh_test.sh not found, skipping"
    fi
else
    echo -e "${RED}BATS not found. Skipping BATS tests.${NC}"
fi

# Expect tests (interactive)
if command -v expect >/dev/null 2>&1; then
    echo -e "\n${GREEN}Running interactive shell tests...${NC}"
    expect tests/integration/zsh_interactive_test.exp
else
    echo -e "${RED}Expect not found. Skipping interactive tests.${NC}"
fi

echo -e "\n${GREEN}All tests completed!${NC}"
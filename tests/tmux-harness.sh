#!/usr/bin/env bash
# Tmux test harness - Pure bash implementation for tmux testing
# No external dependencies required beyond tmux and bash

set -euo pipefail

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Test state
TEST_COUNT=0
FAILED_COUNT=0
CURRENT_TEST=""

# Initialize test environment
init_test_env() {
    # Set up isolated tmux environment
    export TEST_DIR="${TEST_DIR:-$(mktemp -d -t tmux-proj-test-XXXXXX)}"
    export TEST_TMUX_SOCKET="$TEST_DIR/tmux-socket"
    export TEST_PROJECT_ROOT="$TEST_DIR/projects"
    export TMUX_TMPDIR="$TEST_DIR"
    
    # Create directories
    mkdir -p "$TEST_PROJECT_ROOT"
    
    # Build proj binary if needed
    if [[ -z "${PROJ_BINARY:-}" ]]; then
        if [[ -f "./build/proj" ]]; then
            export PROJ_BINARY="$(pwd)/build/proj"
        else
            echo -e "${YELLOW}Building proj binary...${NC}" >&2
            make build >/dev/null 2>&1 || {
                echo -e "${RED}Failed to build proj binary${NC}" >&2
                exit 1
            }
            export PROJ_BINARY="$(pwd)/build/proj"
        fi
    fi
    
    # Kill any existing tmux server on our socket
    tmux -S "$TEST_TMUX_SOCKET" kill-server 2>/dev/null || true
    
    echo -e "${GREEN}Test environment initialized${NC}" >&2
    echo "  TEST_DIR: $TEST_DIR" >&2
    echo "  TEST_TMUX_SOCKET: $TEST_TMUX_SOCKET" >&2
    echo "  PROJ_BINARY: $PROJ_BINARY" >&2
}

# Cleanup test environment
cleanup_test_env() {
    if [[ -n "${TEST_TMUX_SOCKET:-}" ]]; then
        tmux -S "$TEST_TMUX_SOCKET" kill-server 2>/dev/null || true
    fi
    
    if [[ -n "${TEST_DIR:-}" ]] && [[ -d "$TEST_DIR" ]]; then
        rm -rf "$TEST_DIR"
    fi
}

# Test runner functions
run_test() {
    local test_name="$1"
    local test_function="$2"
    
    CURRENT_TEST="$test_name"
    ((TEST_COUNT++))
    
    echo -e "${BLUE}Running test:${NC} $test_name ..."
    
    # Run test in subshell to isolate failures
    if (
        set -e
        echo "[DEBUG] Entering subshell for test: $test_name" >&2
        # Clear any existing sessions before test
        echo "[DEBUG] Killing tmux server..." >&2
        tmux -S "$TEST_TMUX_SOCKET" kill-server 2>/dev/null || true
        echo "[DEBUG] Running test function: $test_function" >&2
        # Run the test function
        "$test_function"
        echo "[DEBUG] Test function completed" >&2
    ); then
        echo -e "  ${GREEN}✓ PASS${NC}"
    else
        echo -e "  ${RED}✗ FAIL${NC}"
        ((FAILED_COUNT++))
    fi
}

# Assertion functions
assert_equals() {
    local expected="$1"
    local actual="$2"
    local message="${3:-Values should be equal}"
    
    if [[ "$expected" != "$actual" ]]; then
        echo -e "${RED}Assertion failed in $CURRENT_TEST: $message${NC}" >&2
        echo -e "${RED}  Expected: '$expected'${NC}" >&2
        echo -e "${RED}  Actual:   '$actual'${NC}" >&2
        return 1
    fi
}

assert_contains() {
    local haystack="$1"
    local needle="$2"
    local message="${3:-String should contain substring}"
    
    if [[ "$haystack" != *"$needle"* ]]; then
        echo -e "${RED}Assertion failed in $CURRENT_TEST: $message${NC}" >&2
        echo -e "${RED}  String: '$haystack'${NC}" >&2
        echo -e "${RED}  Should contain: '$needle'${NC}" >&2
        return 1
    fi
}

assert_true() {
    local condition="$1"
    local message="${2:-Condition should be true}"
    
    if ! eval "$condition"; then
        echo -e "${RED}Assertion failed in $CURRENT_TEST: $message${NC}" >&2
        echo -e "${RED}  Condition: '$condition'${NC}" >&2
        return 1
    fi
}

assert_false() {
    local condition="$1"
    local message="${2:-Condition should be false}"
    
    if eval "$condition"; then
        echo -e "${RED}Assertion failed in $CURRENT_TEST: $message${NC}" >&2
        echo -e "${RED}  Condition should be false: '$condition'${NC}" >&2
        return 1
    fi
}

# Tmux helper functions
tmux_cmd() {
    tmux -S "$TEST_TMUX_SOCKET" "$@"
}

create_session() {
    local session_name="$1"
    local start_dir="${2:-$TEST_PROJECT_ROOT}"
    tmux_cmd new-session -d -s "$session_name" -c "$start_dir"
}

session_exists() {
    local session_name="$1"
    tmux_cmd list-sessions -F "#{session_name}" 2>/dev/null | grep -q "^${session_name}$"
}

get_session_count() {
    # Use word count without newlines and trim any whitespace
    tmux_cmd list-sessions 2>/dev/null | wc -l | tr -d '[:space:]' || echo "0"
}

get_session_dir() {
    local session_name="$1"
    tmux_cmd display-message -t "$session_name" -p "#{session_path}" 2>/dev/null
}

window_exists() {
    local session_name="$1"
    local window_name="$2"
    tmux_cmd list-windows -t "$session_name" -F "#{window_name}" 2>/dev/null | grep -q "^${window_name}$"
}

get_window_count() {
    local session_name="$1"
    # Use word count without newlines and trim any whitespace
    tmux_cmd list-windows -t "$session_name" 2>/dev/null | wc -l | tr -d '[:space:]' || echo "0"
}

# Project helper functions
create_test_project() {
    local org="$1"
    local name="$2"
    local project_dir="$TEST_PROJECT_ROOT/$org/$name"
    
    mkdir -p "$project_dir"
    cd "$project_dir"
    
    git init --quiet
    git config user.name "Test User"
    git config user.email "test@example.com"
    
    echo "# $name" > README.md
    git add README.md
    git commit --quiet -m "Initial commit"
    
    echo "$project_dir"
}

proj_cmd() {
    env PROJECT_ROOT="$TEST_PROJECT_ROOT" "$PROJ_BINARY" "$@"
}

# Report test results
report_results() {
    echo
    echo "================================"
    echo "Test Results:"
    echo "  Total tests: $TEST_COUNT"
    echo -e "  Passed: ${GREEN}$((TEST_COUNT - FAILED_COUNT))${NC}"
    echo -e "  Failed: ${RED}$FAILED_COUNT${NC}"
    echo "================================"
    
    if [[ $FAILED_COUNT -gt 0 ]]; then
        return 1
    fi
    return 0
}

# Note: Functions are available when this file is sourced
# No need to export them as we're sourcing directly
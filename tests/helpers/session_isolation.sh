#!/usr/bin/env bash
# Session isolation helpers for tmux testing
# Provides utilities to ensure tests don't interfere with each other or user sessions

set -euo pipefail

# Session isolation configuration
ISOLATION_PREFIX="test-$$"
ISOLATION_COUNTER=0
ISOLATED_SOCKETS=()

# Simple isolated environment creation - returns socket path directly
create_isolated_env() {
    local test_name="${1:-generic}"
    local socket_dir socket_path
    
    ((ISOLATION_COUNTER++))
    socket_dir=$(mktemp -d -t "tmux-${ISOLATION_PREFIX}-${test_name}-${ISOLATION_COUNTER}-XXXXXX")
    socket_path="$socket_dir/tmux-socket"
    
    # Track this socket for cleanup
    ISOLATED_SOCKETS+=("$socket_path")
    
    echo "$socket_path"
}

# Get tmux command for isolated environment - takes socket path directly
isolated_tmux() {
    local socket_path="$1"
    shift # Remove socket_path from args
    
    if [[ -z "$socket_path" ]]; then
        echo "Error: Socket path required" >&2
        return 1
    fi
    
    tmux -S "$socket_path" "$@"
}

# Create session in isolated environment - takes socket path directly
create_isolated_session() {
    local socket_path="$1"
    local session_name="$2"
    local start_dir="${3:-$(pwd)}"
    
    isolated_tmux "$socket_path" new-session -d -s "$session_name" -c "$start_dir"
    echo "Created isolated session '$session_name' (socket: $socket_path)" >&2
}

# Kill session in isolated environment
kill_isolated_session() {
    local socket_path="$1"
    local session_name="$2"
    
    isolated_tmux "$socket_path" kill-session -t "$session_name" 2>/dev/null || true
    echo "Killed isolated session '$session_name' (socket: $socket_path)" >&2
}

# List sessions in isolated environment
list_isolated_sessions() {
    local socket_path="$1"
    
    isolated_tmux "$socket_path" list-sessions -F "#{session_name}" 2>/dev/null || true
}

# Check if session exists in isolated environment
isolated_session_exists() {
    local socket_path="$1"
    local session_name="$2"
    
    list_isolated_sessions "$socket_path" | grep -q "^${session_name}$"
}

# Get session count in isolated environment
get_isolated_session_count() {
    local socket_path="$1"
    
    list_isolated_sessions "$socket_path" | wc -l | tr -d '[:space:]'
}

# Kill all sessions in isolated environment
kill_all_isolated_sessions() {
    local socket_path="$1"
    
    isolated_tmux "$socket_path" kill-server 2>/dev/null || true
    echo "Killed all sessions in isolated environment (socket: $socket_path)" >&2
}

# Cleanup isolated environment - takes socket path directly
cleanup_isolated_env() {
    local socket_path="$1"
    
    if [[ -n "$socket_path" ]]; then
        # Kill tmux server on this socket
        tmux -S "$socket_path" kill-server 2>/dev/null || true
        
        # Remove socket directory
        local tmpdir_path
        tmpdir_path=$(dirname "$socket_path")
        if [[ -d "$tmpdir_path" ]]; then
            rm -rf "$tmpdir_path"
        fi
        
        echo "Cleaned up isolated environment (socket: $socket_path)" >&2
    fi
}

# Cleanup all tracked isolated environments
cleanup_all_isolated_envs() {
    local socket_path
    local tmpdir_path
    
    # Kill all tracked sockets
    for socket_path in "${ISOLATED_SOCKETS[@]:-}"; do
        if [[ -S "$socket_path" ]]; then
            tmux -S "$socket_path" kill-server 2>/dev/null || true
            tmpdir_path=$(dirname "$socket_path")
            if [[ -d "$tmpdir_path" ]]; then
                rm -rf "$tmpdir_path"
            fi
        fi
    done
    
    # Clear the array
    ISOLATED_SOCKETS=()
    
    # Clean up any remaining environment variables
    local var_name
    for var_name in $(env | grep "^TMUX_SOCKET_" | cut -d= -f1); do
        unset "$var_name"
    done
    for var_name in $(env | grep "^TMUX_TMPDIR_" | cut -d= -f1); do
        unset "$var_name"
    done
    
    echo "Cleaned up all isolated environments" >&2
}

# Ensure cleanup happens on script exit
trap cleanup_all_isolated_envs EXIT

# Test runner that uses isolation
run_isolated_test() {
    local test_name="$1"
    local test_function="$2"
    
    echo "Running isolated test: $test_name"
    
    # Create isolated environment
    local socket_path
    socket_path=$(create_isolated_env "$test_name")
    
    # Run test in subshell with isolated environment
    local result=0
    if (
        export TEST_SOCKET="$socket_path"
        set -e
        "$test_function"
    ); then
        echo "  ✓ PASS: $test_name"
    else
        echo "  ✗ FAIL: $test_name"
        result=1
    fi
    
    # Cleanup this isolated environment
    cleanup_isolated_env "$socket_path"
    
    return $result
}

# Verify isolation is working
test_isolation_works() {
    local socket1 socket2
    
    socket1=$(create_isolated_env "test1")
    socket2=$(create_isolated_env "test2")
    
    # Create sessions with same name in different environments
    create_isolated_session "$socket1" "same-name"
    create_isolated_session "$socket2" "same-name"
    
    # Both should exist in their respective environments
    if isolated_session_exists "$socket1" "same-name" && 
       isolated_session_exists "$socket2" "same-name"; then
        echo "✓ Session isolation working correctly"
        cleanup_isolated_env "$socket1"
        cleanup_isolated_env "$socket2"
        return 0
    else
        echo "✗ Session isolation failed"
        cleanup_isolated_env "$socket1"
        cleanup_isolated_env "$socket2"
        return 1
    fi
}

# Helper to run multiple isolated tests
run_isolated_test_suite() {
    local suite_name="$1"
    shift
    local tests=("$@")
    
    echo "Running isolated test suite: $suite_name"
    echo "================================"
    
    local passed=0
    local failed=0
    local test_name test_function
    
    for test_spec in "${tests[@]}"; do
        test_name=$(echo "$test_spec" | cut -d: -f1)
        test_function=$(echo "$test_spec" | cut -d: -f2)
        
        if run_isolated_test "$test_name" "$test_function"; then
            ((passed++))
        else
            ((failed++))
        fi
    done
    
    echo "================================"
    echo "Test suite results:"
    echo "  Passed: $passed"
    echo "  Failed: $failed"
    echo "  Total:  $((passed + failed))"
    
    return $failed
}

echo "Session isolation helpers loaded (PID: $$, PREFIX: $ISOLATION_PREFIX)" >&2
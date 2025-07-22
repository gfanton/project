#!/usr/bin/env bash
# Test cleanup and teardown procedures
# Provides comprehensive cleanup for tmux integration tests

set -euo pipefail

# Cleanup state tracking
CLEANUP_REGISTERED=()
CLEANUP_TEMP_DIRS=()
CLEANUP_PROCESSES=()
CLEANUP_TMUX_SOCKETS=()

# Register cleanup action
register_cleanup() {
    local action="$1"
    CLEANUP_REGISTERED+=("$action")
    echo "[CLEANUP] Registered cleanup action: $action" >&2
}

# Register temporary directory for cleanup
register_temp_dir() {
    local temp_dir="$1"
    CLEANUP_TEMP_DIRS+=("$temp_dir")
    echo "[CLEANUP] Registered temp directory: $temp_dir" >&2
}

# Register process for cleanup
register_process() {
    local pid="$1"
    CLEANUP_PROCESSES+=("$pid")
    echo "[CLEANUP] Registered process: $pid" >&2
}

# Register tmux socket for cleanup
register_tmux_socket() {
    local socket_path="$1"
    CLEANUP_TMUX_SOCKETS+=("$socket_path")
    echo "[CLEANUP] Registered tmux socket: $socket_path" >&2
}

# Kill specific tmux socket
cleanup_tmux_socket() {
    local socket_path="$1"
    
    if [[ -S "$socket_path" ]]; then
        echo "[CLEANUP] Killing tmux server on socket: $socket_path" >&2
        tmux -S "$socket_path" kill-server 2>/dev/null || true
        
        # Wait a moment for server to shut down
        local timeout=5
        while [[ $timeout -gt 0 ]] && [[ -S "$socket_path" ]]; do
            sleep 0.1
            ((timeout--))
        done
        
        if [[ -S "$socket_path" ]]; then
            echo "[CLEANUP] Warning: socket still exists after cleanup: $socket_path" >&2
        else
            echo "[CLEANUP] Successfully killed tmux server on: $socket_path" >&2
        fi
    fi
}

# Kill all registered tmux sockets
cleanup_all_tmux_sockets() {
    local socket_path
    for socket_path in "${CLEANUP_TMUX_SOCKETS[@]:-}"; do
        cleanup_tmux_socket "$socket_path"
    done
    CLEANUP_TMUX_SOCKETS=()
}

# Kill specific process
cleanup_process() {
    local pid="$1"
    
    if kill -0 "$pid" 2>/dev/null; then
        echo "[CLEANUP] Killing process: $pid" >&2
        kill "$pid" 2>/dev/null || true
        
        # Wait for process to die
        local timeout=10
        while [[ $timeout -gt 0 ]] && kill -0 "$pid" 2>/dev/null; do
            sleep 0.1
            ((timeout--))
        done
        
        # Force kill if still running
        if kill -0 "$pid" 2>/dev/null; then
            echo "[CLEANUP] Force killing process: $pid" >&2
            kill -9 "$pid" 2>/dev/null || true
        fi
    fi
}

# Kill all registered processes
cleanup_all_processes() {
    local pid
    for pid in "${CLEANUP_PROCESSES[@]:-}"; do
        cleanup_process "$pid"
    done
    CLEANUP_PROCESSES=()
}

# Remove temporary directory
cleanup_temp_dir() {
    local temp_dir="$1"
    
    if [[ -d "$temp_dir" ]]; then
        echo "[CLEANUP] Removing temp directory: $temp_dir" >&2
        rm -rf "$temp_dir" || {
            echo "[CLEANUP] Warning: failed to remove temp directory: $temp_dir" >&2
        }
    fi
}

# Remove all registered temporary directories
cleanup_all_temp_dirs() {
    local temp_dir
    for temp_dir in "${CLEANUP_TEMP_DIRS[@]:-}"; do
        cleanup_temp_dir "$temp_dir"
    done
    CLEANUP_TEMP_DIRS=()
}

# Execute custom cleanup action
execute_cleanup_action() {
    local action="$1"
    
    echo "[CLEANUP] Executing custom cleanup: $action" >&2
    eval "$action" || {
        echo "[CLEANUP] Warning: cleanup action failed: $action" >&2
    }
}

# Execute all registered cleanup actions
execute_all_cleanup_actions() {
    local action
    for action in "${CLEANUP_REGISTERED[@]:-}"; do
        execute_cleanup_action "$action"
    done
    CLEANUP_REGISTERED=()
}

# Comprehensive cleanup function
comprehensive_cleanup() {
    echo "[CLEANUP] Starting comprehensive cleanup..." >&2
    
    # Execute custom cleanup actions first
    execute_all_cleanup_actions
    
    # Kill tmux sockets
    cleanup_all_tmux_sockets
    
    # Kill processes
    cleanup_all_processes
    
    # Remove temp directories
    cleanup_all_temp_dirs
    
    # Kill any lingering tmux processes from this session
    cleanup_lingering_tmux_processes
    
    echo "[CLEANUP] Comprehensive cleanup completed" >&2
}

# Kill any lingering tmux processes that might be from our tests
cleanup_lingering_tmux_processes() {
    # Find tmux processes with our test prefix in socket names
    local pids
    if command -v pgrep >/dev/null 2>&1; then
        # Look for tmux processes with test-related sockets
        pids=$(pgrep -f "tmux.*test-$$" 2>/dev/null || true)
        if [[ -n "$pids" ]]; then
            echo "[CLEANUP] Killing lingering tmux test processes: $pids" >&2
            echo "$pids" | xargs kill 2>/dev/null || true
        fi
    fi
}

# Cleanup on script exit
cleanup_on_exit() {
    local exit_code=$?
    echo "[CLEANUP] Script exiting with code $exit_code, running cleanup..." >&2
    comprehensive_cleanup
    exit $exit_code
}

# Safe directory removal that handles mount points and special files
safe_remove_directory() {
    local dir="$1"
    
    if [[ ! -d "$dir" ]]; then
        return 0
    fi
    
    # Check if it's a mount point
    if mountpoint -q "$dir" 2>/dev/null; then
        echo "[CLEANUP] Warning: Not removing mount point: $dir" >&2
        return 1
    fi
    
    # Check if it contains any mount points
    if find "$dir" -type d -exec mountpoint -q {} \; 2>/dev/null | grep -q .; then
        echo "[CLEANUP] Warning: Directory contains mount points: $dir" >&2
        return 1
    fi
    
    # Safe removal
    rm -rf "$dir"
    echo "[CLEANUP] Safely removed directory: $dir" >&2
}

# Test environment factory with automatic cleanup
create_test_env() {
    local test_name="$1"
    local base_dir="${2:-$(mktemp -d -t "tmux-test-${test_name}-XXXXXX")}"
    
    # Create test environment structure
    mkdir -p "$base_dir"/{sockets,tmp,projects,config}
    
    # Register for cleanup
    register_temp_dir "$base_dir"
    
    # Export environment variables
    export "TEST_ENV_${test_name^^}_DIR=$base_dir"
    export "TEST_ENV_${test_name^^}_SOCKET=$base_dir/sockets/tmux-socket"
    export "TEST_ENV_${test_name^^}_TMP=$base_dir/tmp"
    export "TEST_ENV_${test_name^^}_PROJECTS=$base_dir/projects"
    export "TEST_ENV_${test_name^^}_CONFIG=$base_dir/config"
    
    echo "[CLEANUP] Created test environment '$test_name' at: $base_dir" >&2
    echo "$base_dir"
}

# Teardown specific test environment
teardown_test_env() {
    local test_name="$1"
    local dir_var="TEST_ENV_${test_name^^}_DIR"
    local socket_var="TEST_ENV_${test_name^^}_SOCKET"
    local base_dir="${!dir_var:-}"
    local socket_path="${!socket_var:-}"
    
    if [[ -n "$socket_path" ]]; then
        cleanup_tmux_socket "$socket_path"
    fi
    
    if [[ -n "$base_dir" ]]; then
        safe_remove_directory "$base_dir"
        
        # Clean up environment variables
        unset "$dir_var"
        unset "TEST_ENV_${test_name^^}_SOCKET"
        unset "TEST_ENV_${test_name^^}_TMP"
        unset "TEST_ENV_${test_name^^}_PROJECTS"
        unset "TEST_ENV_${test_name^^}_CONFIG"
    fi
    
    echo "[CLEANUP] Tore down test environment: $test_name" >&2
}

# Validate cleanup completeness
validate_cleanup() {
    local issues=0
    
    # Check for remaining test sockets
    if find /tmp -name "*test-$$*" -type s 2>/dev/null | grep -q .; then
        echo "[CLEANUP] Warning: Found remaining test sockets" >&2
        find /tmp -name "*test-$$*" -type s 2>/dev/null | head -5
        ((issues++))
    fi
    
    # Check for remaining test directories
    if find /tmp -name "*tmux-test-*" -type d 2>/dev/null | grep -q .; then
        echo "[CLEANUP] Warning: Found remaining test directories" >&2
        find /tmp -name "*tmux-test-*" -type d 2>/dev/null | head -5
        ((issues++))
    fi
    
    # Check for remaining tmux processes
    if pgrep -f "tmux.*test-$$" >/dev/null 2>&1; then
        echo "[CLEANUP] Warning: Found remaining tmux test processes" >&2
        pgrep -f "tmux.*test-$$" | head -5
        ((issues++))
    fi
    
    if [[ $issues -eq 0 ]]; then
        echo "[CLEANUP] ✓ Cleanup validation passed" >&2
        return 0
    else
        echo "[CLEANUP] ✗ Cleanup validation found $issues issues" >&2
        return 1
    fi
}

# Register signal handlers for cleanup
trap cleanup_on_exit EXIT
trap cleanup_on_exit INT
trap cleanup_on_exit TERM

echo "[CLEANUP] Test cleanup helpers loaded (PID: $$)" >&2
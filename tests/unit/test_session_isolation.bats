#!/usr/bin/env bats
# Session isolation testing

# Load test helpers
load '../helpers/session_isolation.sh'
load '../helpers/test_cleanup.sh'

# Global test state
ISOLATION_IDS=()

# Setup for each test
setup() {
    # Clear any previous isolation state
    cleanup_all_isolated_envs
    ISOLATION_IDS=()
}

# Teardown for each test
teardown() {
    # Cleanup all isolations created in this test
    for id in "${ISOLATION_IDS[@]:-}"; do
        cleanup_isolated_env "$id" 2>/dev/null || true
    done
    cleanup_all_isolated_envs
}

@test "isolated environment creation works" {
    # Create isolated environment
    local socket_path
    socket_path=$(create_isolated_env "test1")
    ISOLATION_IDS+=("$socket_path")
    
    # Verify socket path is returned
    [[ -n "$socket_path" ]]
    
    # Verify socket directory exists
    [[ -d "$(dirname "$socket_path")" ]]
}

@test "isolated tmux commands work" {
    # Create isolated environment
    local socket_path
    socket_path=$(create_isolated_env "tmux-test")
    ISOLATION_IDS+=("$socket_path")
    
    # Create session using isolated tmux
    run create_isolated_session "$socket_path" "test-session" 
    [[ "$status" -eq 0 ]]
    
    # Verify session exists in isolated environment
    run isolated_session_exists "$socket_path" "test-session"
    [[ "$status" -eq 0 ]]
    
    # Verify session count
    local count
    count=$(get_isolated_session_count "$socket_path")
    [[ "$count" -eq 1 ]]
}

@test "session isolation prevents interference" {
    # Create two isolated environments
    local env1 env2
    env1=$(create_isolated_env "env1")
    env2=$(create_isolated_env "env2") 
    ISOLATION_IDS+=("$env1" "$env2")
    
    # Create sessions with same name in both environments
    create_isolated_session "$env1" "same-name"
    create_isolated_session "$env2" "same-name"
    
    # Both should exist independently
    run isolated_session_exists "$env1" "same-name"
    [[ "$status" -eq 0 ]]
    
    run isolated_session_exists "$env2" "same-name"
    [[ "$status" -eq 0 ]]
    
    # Each environment should have exactly one session
    local count1 count2
    count1=$(get_isolated_session_count "$env1")
    count2=$(get_isolated_session_count "$env2")
    [[ "$count1" -eq 1 ]]
    [[ "$count2" -eq 1 ]]
}

@test "isolated session management works" {
    # Create isolated environment
    local env_id
    env_id=$(create_isolated_env "session-mgmt")
    ISOLATION_IDS+=("$env_id")
    
    # Create multiple sessions
    create_isolated_session "$env_id" "session1"
    create_isolated_session "$env_id" "session2"
    create_isolated_session "$env_id" "session3"
    
    # Verify all sessions exist
    local count
    count=$(get_isolated_session_count "$env_id")
    [[ "$count" -eq 3 ]]
    
    # List sessions and verify names
    local sessions
    sessions=$(list_isolated_sessions "$env_id")
    [[ "$sessions" =~ "session1" ]]
    [[ "$sessions" =~ "session2" ]]
    [[ "$sessions" =~ "session3" ]]
    
    # Kill one session
    kill_isolated_session "$env_id" "session2"
    
    # Verify session is gone
    count=$(get_isolated_session_count "$env_id")
    [[ "$count" -eq 2 ]]
    
    run isolated_session_exists "$env_id" "session2"
    [[ "$status" -eq 1 ]]
    
    # Verify other sessions still exist
    run isolated_session_exists "$env_id" "session1"
    [[ "$status" -eq 0 ]]
    
    run isolated_session_exists "$env_id" "session3"  
    [[ "$status" -eq 0 ]]
}

@test "kill all sessions works" {
    # Create isolated environment
    local env_id
    env_id=$(create_isolated_env "kill-all")
    ISOLATION_IDS+=("$env_id")
    
    # Create multiple sessions
    create_isolated_session "$env_id" "session1"
    create_isolated_session "$env_id" "session2"
    create_isolated_session "$env_id" "session3"
    
    # Verify sessions exist
    local count
    count=$(get_isolated_session_count "$env_id")
    [[ "$count" -eq 3 ]]
    
    # Kill all sessions
    kill_all_isolated_sessions "$env_id"
    
    # Verify no sessions remain
    count=$(get_isolated_session_count "$env_id")
    [[ "$count" -eq 0 ]]
}

@test "isolated environment cleanup works" {
    # Create isolated environment (returns socket path directly)
    local socket_path
    socket_path=$(create_isolated_env "cleanup-test")
    
    # Create session
    create_isolated_session "$socket_path" "test-session"
    
    # Verify session exists
    run isolated_session_exists "$socket_path" "test-session"
    [[ "$status" -eq 0 ]]
    
    # Store socket directory for verification
    local socket_dir
    socket_dir=$(dirname "$socket_path")
    
    # Cleanup environment
    cleanup_isolated_env "$socket_path"
    
    # Verify socket directory is cleaned up
    [[ ! -d "$socket_dir" ]]
}

@test "run_isolated_test function works" {
    # Define a simple test function
    simple_test() {
        local socket_path="$TEST_SOCKET"
        [[ -n "$socket_path" ]]
        
        # Create and verify session
        create_isolated_session "$socket_path" "isolated-test-session"
        isolated_session_exists "$socket_path" "isolated-test-session"
    }
    
    # Run the test in isolation
    run run_isolated_test "simple-test" "simple_test"
    [[ "$status" -eq 0 ]]
    [[ "$output" =~ "✓ PASS: simple-test" ]]
}

@test "isolation test suite works" {
    # Define test functions
    test1() {
        create_isolated_session "$TEST_SOCKET" "suite-session1"
        isolated_session_exists "$TEST_SOCKET" "suite-session1"
    }
    
    test2() {
        create_isolated_session "$TEST_SOCKET" "suite-session2"
        isolated_session_exists "$TEST_SOCKET" "suite-session2" 
    }
    
    # Run test suite
    run run_isolated_test_suite "test-suite" "test1:test1" "test2:test2"
    [[ "$status" -eq 0 ]]
    [[ "$output" =~ "Passed: 2" ]]
    [[ "$output" =~ "Failed: 0" ]]
}

@test "isolation verification works" {
    # Run the built-in isolation test
    run test_isolation_works
    [[ "$status" -eq 0 ]]
    [[ "$output" =~ "✓ Session isolation working correctly" ]]
}
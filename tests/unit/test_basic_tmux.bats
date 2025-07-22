#!/usr/bin/env bats
# Basic tmux testing functionality

# Source the tmux test harness
source "$(dirname "$BATS_TEST_DIRNAME")/tmux-harness.sh"

# Setup and teardown
setup() {
    # Initialize test environment
    init_test_env
}

teardown() {
    # Clean up test environment
    cleanup_test_env
}

@test "test environment is properly set up" {
    # Check required environment variables
    [[ -n "$TEST_DIR" ]]
    [[ -n "$TEST_TMUX_SOCKET" ]]
    [[ -n "$TEST_PROJECT_ROOT" ]]
    [[ -n "$PROJ_BINARY" ]]
    
    # Check directories exist
    [[ -d "$TEST_DIR" ]]
    [[ -d "$TEST_PROJECT_ROOT" ]]
    
    # Check proj binary exists and is executable
    [[ -f "$PROJ_BINARY" ]]
    [[ -x "$PROJ_BINARY" ]]
}

@test "can start and stop isolated tmux server" {
    # Initially no sessions should exist
    [[ $(get_session_count) -eq 0 ]]
    
    # Create a test session
    create_session "test-basic"
    
    # Session should now exist
    assert_true "session_exists 'test-basic'"
    [[ $(get_session_count) -eq 1 ]]
    
    # Kill the session
    tmux_cmd kill-session -t "test-basic"
    
    # Should be no sessions again
    [[ $(get_session_count) -eq 0 ]]
}

@test "can create test projects" {
    # Create test project
    local project_path
    project_path=$(create_test_project "testorg" "testproj")
    
    # Project should exist
    [[ -d "$project_path" ]]
    [[ -d "$project_path/.git" ]]
    
    # Should have initial commit
    cd "$project_path"
    local log_output
    log_output=$(git log --oneline)
    assert_contains "$log_output" "Initial commit"
}

@test "proj command works in test environment" {
    # Create test project
    create_test_project "testorg" "testproj"
    
    # Test proj list command
    local list_output
    list_output=$(proj_cmd list)
    assert_contains "$list_output" "testorg/testproj"
}

@test "tmux session isolation works" {
    # Create multiple sessions
    create_session "session1"
    create_session "session2"
    
    # Both should exist
    assert_true "session_exists 'session1'"
    assert_true "session_exists 'session2'"
    [[ $(get_session_count) -eq 2 ]]
    
    # Kill one session
    tmux_cmd kill-session -t "session1"
    
    # Only session2 should remain
    assert_false "session_exists 'session1'"
    assert_true "session_exists 'session2'"
    [[ $(get_session_count) -eq 1 ]]
    
    # Kill server completely
    tmux_cmd kill-server 2>/dev/null || true
    
    # No sessions should exist
    [[ $(get_session_count) -eq 0 ]]
}

@test "helper functions work correctly" {
    # Test project creation helper
    local project_path
    project_path=$(create_test_project "myorg" "myproject")
    [[ -n "$project_path" ]]
    [[ -d "$project_path" ]]
    
    # Test session creation helper
    create_session "helper-test" "$project_path"
    assert_true "session_exists 'helper-test'"
    
    # Test working directory is correct
    local session_dir
    session_dir=$(get_session_dir "helper-test")
    assert_equals "$project_path" "$session_dir" "Session working directory should match project path"
}
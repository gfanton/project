#!/usr/bin/env bash
# Test tmux session management

# Source the test harness if not already loaded
if ! type run_test >/dev/null 2>&1; then
    source "$(dirname "${BASH_SOURCE[0]}")/../tmux-harness.sh"
fi

# Test basic session creation
test_session_creation() {
    # Create a session
    create_session "test-proj-session"
    
    # Verify it exists
    assert_true "session_exists 'test-proj-session'" \
        "Session should exist after creation"
    
    # Verify session count
    assert_equals "1" "$(get_session_count)" \
        "Should have exactly one session"
}

# Test session naming conventions
test_session_naming() {
    # Create sessions with project-style names
    create_session "proj-gfanton-project"
    create_session "proj-testorg-webapp"
    
    # Both should exist
    assert_true "session_exists 'proj-gfanton-project'"
    assert_true "session_exists 'proj-testorg-webapp'"
    
    # List sessions and check naming
    local sessions
    sessions=$(tmux_cmd list-sessions -F "#{session_name}")
    assert_contains "$sessions" "proj-gfanton-project"
    assert_contains "$sessions" "proj-testorg-webapp"
}

# Test session working directory
test_session_working_directory() {
    # Create a test project
    local project_path
    project_path=$(create_test_project "myorg" "myapp")
    
    # Create session in project directory
    create_session "proj-myorg-myapp" "$project_path"
    
    # Verify working directory
    local session_dir
    session_dir=$(get_session_dir "proj-myorg-myapp")
    assert_equals "$project_path" "$session_dir" \
        "Session should start in project directory"
}

# Test multiple sessions
test_multiple_sessions() {
    # Create multiple sessions
    create_session "session-1"
    create_session "session-2"
    create_session "session-3"
    
    # Verify count
    assert_equals "3" "$(get_session_count)" \
        "Should have three sessions"
    
    # Kill middle session
    tmux_cmd kill-session -t "session-2"
    
    # Verify count and remaining sessions
    assert_equals "2" "$(get_session_count)" \
        "Should have two sessions after killing one"
    assert_true "session_exists 'session-1'"
    assert_false "session_exists 'session-2'"
    assert_true "session_exists 'session-3'"
}

# Test session windows
test_session_windows() {
    # Create session
    create_session "proj-test"
    
    # Default should have one window
    assert_equals "1" "$(get_window_count 'proj-test')" \
        "New session should have one window"
    
    # Create additional windows
    tmux_cmd new-window -t "proj-test" -n "feature"
    tmux_cmd new-window -t "proj-test" -n "bugfix"
    
    # Verify window count
    assert_equals "3" "$(get_window_count 'proj-test')" \
        "Should have three windows"
    
    # Verify specific windows exist
    assert_true "window_exists 'proj-test' 'feature'"
    assert_true "window_exists 'proj-test' 'bugfix'"
}

# Check if we have the functions we need
if ! type run_test >/dev/null 2>&1; then
    echo "ERROR: run_test function not found!" >&2
    exit 1
fi

# Run all tests
echo "Starting tmux session tests..."
run_test "Session creation" test_session_creation
run_test "Session naming conventions" test_session_naming  
run_test "Session working directory" test_session_working_directory
run_test "Multiple sessions" test_multiple_sessions
run_test "Session windows" test_session_windows
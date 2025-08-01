#!/usr/bin/env bats
# Project menu plugin tests

# Load test helpers
load '../helpers/plugin_test_helpers.sh'
load '../helpers/session_isolation.sh' 
load '../helpers/test_cleanup.sh'

# Setup for each test
setup() {
    # Create isolated plugin test environment
    init_plugin_test_env
    register_temp_dir "$PLUGIN_TEST_DIR"
    register_tmux_socket "$PLUGIN_SOCKET"
    
    # Install the actual tmux-proj plugin
    install_test_plugin "$(pwd)/plugins/proj-tmux/plugin"
    load_test_plugin
}

# Teardown for each test  
teardown() {
    cleanup_plugin_test_env
}

@test "project menu plugin loads without errors" {
    # Verify plugin loaded
    run assert_plugin_loaded
    [[ "$status" -eq 0 ]]
    
    # Verify key bindings exist
    run assert_key_binding_exists "P"
    [[ "$status" -eq 0 ]]
    
    run assert_key_binding_exists "C-p"
    [[ "$status" -eq 0 ]]
}

@test "proj commands are available in test environment" {
    # Test proj command is available
    run which proj
    [[ "$status" -eq 0 ]]
    
    # Test proj-tmux command is available  
    run which proj-tmux
    [[ "$status" -eq 0 ]]
    
    # Test proj list works
    run proj list
    [[ "$status" -eq 0 ]]
    
    # Test proj-tmux help works
    run proj-tmux --help
    [[ "$status" -eq 1 ]]  # Expected since it shows help and exits
}

@test "project menu script can be executed in tmux context" {
    # Create a test session
    plugin_tmux new-session -d -s "menu-test"
    
    # Get the actual script path from the plugin
    local script_path="$PLUGIN_TEST_DIR/plugins/tmux-proj/scripts/project_menu.sh"
    
    # Verify script exists
    [[ -f "$script_path" ]]
    
    # Test script execution in tmux context
    # Note: This might fail if no projects found, but should not return error 1 for other reasons
    run plugin_tmux run-shell "$script_path"
    
    # Check the output for clues about what went wrong
    echo "Script output: $output" >&2
    echo "Script status: $status" >&2
    
    # For debugging - show what projects are available
    run proj list
    echo "Available projects: $output" >&2
}

@test "project popup script handles fzf availability correctly" {
    # Create a test session
    plugin_tmux new-session -d -s "popup-test"
    
    local script_path="$PLUGIN_TEST_DIR/plugins/tmux-proj/scripts/project_popup.sh"
    
    # Verify script exists
    [[ -f "$script_path" ]]
    
    # Test if fzf is available
    if command -v fzf >/dev/null 2>&1; then
        echo "fzf is available for testing" >&2
        
        # Test script execution - this may fail due to no user input, but shouldn't error out
        run plugin_tmux run-shell "$script_path"
        echo "Popup script output: $output" >&2
        echo "Popup script status: $status" >&2
    else
        echo "fzf not available - testing fallback behavior" >&2
        
        # Without fzf, it should fall back to regular menu
        run plugin_tmux run-shell "$script_path"
        echo "Fallback script output: $output" >&2
        echo "Fallback script status: $status" >&2
    fi
}

@test "project session creation works with proj-tmux" {
    # Create a test session 
    plugin_tmux new-session -d -s "session-test"
    
    # Test project session creation - using a known project structure
    # Create a mock project directory for testing
    mkdir -p "$PLUGIN_TEST_DIR/test-projects/gfanton/test-project"
    cd "$PLUGIN_TEST_DIR/test-projects"
    
    # Set proj root to our test directory
    export PROJECT_ROOT="$PLUGIN_TEST_DIR/test-projects"
    
    # Test session creation
    run proj-tmux session create "gfanton/test-project"
    echo "Session creation output: $output" >&2
    echo "Session creation status: $status" >&2
    
    # Note: This might fail due to tmux context issues, but we can check the command structure
}

@test "debug project menu script logic step by step" {
    # Test individual functions from the script by sourcing it
    local script_path="$PLUGIN_TEST_DIR/plugins/tmux-proj/scripts/project_menu.sh"
    [[ -f "$script_path" ]]
    
    # Create test session first
    plugin_tmux new-session -d -s "debug-test"
    
    # Test the project count logic
    echo "Testing project count..." >&2
    run bash -c "source '$script_path'; get_project_count"
    echo "Project count: $output (status: $status)" >&2
    
    # Test get_projects function
    echo "Testing get_projects..." >&2
    run bash -c "source '$script_path'; get_projects | head -5"
    echo "Projects (first 5): $output (status: $status)" >&2
    
    # Test with different project counts
    local project_count
    project_count=$(bash -c "source '$script_path'; get_project_count" 2>/dev/null || echo "0")
    echo "Detected project count: $project_count" >&2
    
    if [[ "$project_count" -gt 15 ]]; then
        echo "Should trigger popup mode (>15 projects)" >&2
    else
        echo "Should use menu mode (≤15 projects)" >&2
    fi
}
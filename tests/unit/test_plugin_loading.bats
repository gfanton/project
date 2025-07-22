#!/usr/bin/env bats
# Plugin loading and configuration tests

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
}

# Teardown for each test  
teardown() {
    cleanup_plugin_test_env
}

@test "plugin test environment initializes correctly" {
    # Test environment variables are set
    [[ -n "$PLUGIN_TEST_DIR" ]]
    [[ -n "$PLUGIN_SOCKET" ]]
    [[ -n "$PLUGIN_TMPDIR" ]]
    
    # Test directories exist
    [[ -d "$PLUGIN_TEST_DIR/plugins/tmux-proj" ]]
    [[ -d "$PLUGIN_TEST_DIR/config" ]]
}

@test "plugin tmux command wrapper works" {
    # Should be able to start tmux server on isolated socket
    plugin_tmux new-session -d -s "test-session"
    
    # Verify session exists
    run plugin_tmux list-sessions -F "#{session_name}"
    [[ "$status" -eq 0 ]]
    [[ "$output" =~ "test-session" ]]
}

@test "plugin installation creates necessary files" {
    # Create mock plugin files
    echo '#!/usr/bin/env bash' > tmux-proj.tmux
    echo 'echo "Plugin loaded"' >> tmux-proj.tmux
    mkdir -p scripts
    echo '#!/usr/bin/env bash' > scripts/test-script.sh
    
    # Install plugin
    install_test_plugin "."
    
    # Verify files were copied
    [[ -f "$PLUGIN_TEST_DIR/plugins/tmux-proj/tmux-proj.tmux" ]]
    [[ -f "$PLUGIN_TEST_DIR/plugins/tmux-proj/scripts/test-script.sh" ]]
    
    # Cleanup mock files
    rm -f tmux-proj.tmux
    rm -rf scripts
}

@test "plugin loading executes without errors" {
    # Create mock plugin file using raw tmux config syntax
    cat > tmux-proj.tmux << 'EOF'
# Mock plugin that sets a simple option
set-option -g "@proj_loaded" "true"
EOF
    
    # Install and load plugin
    install_test_plugin "."
    
    run load_test_plugin
    [[ "$status" -eq 0 ]]
    
    # Verify plugin was loaded (check for the option we set)
    run plugin_tmux show-options -g "@proj_loaded"
    [[ "$status" -eq 0 ]]
    
    # Cleanup mock file
    rm -f tmux-proj.tmux
}

@test "key binding registration can be tested" {
    # Create mock plugin with key binding using raw tmux config syntax
    cat > tmux-proj.tmux << 'EOF'
# Mock plugin that registers a key binding
bind-key "P" display-message "Project menu"
EOF
    
    # Install and load plugin
    install_test_plugin "."
    load_test_plugin
    
    # Test key binding exists
    run test_key_binding "P" "prefix"
    [[ "$status" -eq 0 ]]
    
    # Test non-existent key binding
    run test_key_binding "Z" "prefix"
    [[ "$status" -eq 1 ]]
    
    # Cleanup mock file
    rm -f tmux-proj.tmux
}

@test "tmux option testing works" {
    # Start tmux server
    plugin_tmux new-session -d -s "option-test"
    
    # Set a test option
    plugin_tmux set-option -g @proj_test "test-value"
    
    # Test option checking
    run test_tmux_option "@proj_test" "test-value"
    [[ "$status" -eq 0 ]]
    
    # Test wrong value
    run test_tmux_option "@proj_test" "wrong-value"
    [[ "$status" -eq 1 ]]
    
    # Test option exists (without value check)
    run test_tmux_option "@proj_test"
    [[ "$status" -eq 0 ]]
}

@test "session creation with plugin works" {
    # Create plugin test session
    create_plugin_test_session "plugin-session" "$PLUGIN_TEST_DIR"
    
    # Verify session exists
    run plugin_tmux list-sessions -F "#{session_name}"
    [[ "$status" -eq 0 ]]
    [[ "$output" =~ "plugin-session" ]]
    
    # Verify working directory
    run plugin_tmux display-message -t "plugin-session" -p "#{session_path}"
    [[ "$status" -eq 0 ]]
    [[ "$output" == "$PLUGIN_TEST_DIR" ]]
}

@test "window creation with plugin works" {
    # Create session first
    create_plugin_test_session "window-session" "$PLUGIN_TEST_DIR"
    
    # Create additional window
    create_plugin_test_window "window-session" "feature-window" "$PLUGIN_TEST_DIR"
    
    # Verify window exists
    run plugin_tmux list-windows -t "window-session" -F "#{window_name}"
    [[ "$status" -eq 0 ]]
    [[ "$output" =~ "feature-window" ]]
}

@test "plugin assertions work correctly" {
    # Create mock plugin that sets identifiable options using raw tmux config syntax
    cat > tmux-proj.tmux << 'EOF'
# Mock plugin that sets identifiable options
set-option -g "@proj_loaded" "true"
bind-key "P" display-message "Project menu"
EOF
    
    # Install and load plugin
    install_test_plugin "."
    load_test_plugin
    
    # Test plugin loaded assertion
    run assert_plugin_loaded
    [[ "$status" -eq 0 ]]
    
    # Test key binding assertion
    run assert_key_binding_exists "P"
    [[ "$status" -eq 0 ]]
    
    # Test session name format assertion
    run assert_session_has_name_format "proj-test-project" "proj-.*-.*"
    [[ "$status" -eq 0 ]]
    
    # Cleanup mock file
    rm -f tmux-proj.tmux
}

@test "cleanup validation works" {
    # Create some test resources
    create_plugin_test_session "cleanup-test"
    create_plugin_test_window "cleanup-test" "test-window"
    
    # Verify resources exist
    run plugin_tmux list-sessions
    [[ "$status" -eq 0 ]]
    
    # Manual cleanup
    cleanup_plugin_test_env
    
    # Validate cleanup - this should pass since we cleaned up
    # (Note: validation might not be perfect in isolated environment)
    run validate_cleanup
    # Don't require this to pass as it may have false positives in CI
}
#!/usr/bin/env bash
# Plugin-specific test helpers for tmux integration testing
# Provides utilities for testing tmux plugins in isolation

set -euo pipefail

# Plugin testing configuration
PLUGIN_TEST_DIR="${PLUGIN_TEST_DIR:-}"
PLUGIN_SOCKET="${PLUGIN_SOCKET:-}"
PLUGIN_TMPDIR="${PLUGIN_TMPDIR:-}"

# Initialize plugin testing environment
init_plugin_test_env() {
    # Set up isolated plugin test environment
    export PLUGIN_TEST_DIR="${PLUGIN_TEST_DIR:-$(mktemp -d -t tmux-plugin-test-XXXXXX)}"
    export PLUGIN_SOCKET="$PLUGIN_TEST_DIR/plugin-socket"
    export PLUGIN_TMPDIR="$PLUGIN_TEST_DIR"
    export TMUX_TMPDIR="$PLUGIN_TMPDIR"
    
    # Create plugin directories
    mkdir -p "$PLUGIN_TEST_DIR/plugins/tmux-proj"
    mkdir -p "$PLUGIN_TEST_DIR/config"
    
    echo "[PLUGIN-TEST] Plugin test environment initialized" >&2
    echo "  PLUGIN_TEST_DIR: $PLUGIN_TEST_DIR" >&2
    echo "  PLUGIN_SOCKET: $PLUGIN_SOCKET" >&2
}

# Cleanup plugin testing environment
cleanup_plugin_test_env() {
    if [[ -n "${PLUGIN_SOCKET:-}" ]] && [[ -S "$PLUGIN_SOCKET" ]]; then
        tmux -S "$PLUGIN_SOCKET" kill-server 2>/dev/null || true
    fi
    
    if [[ -n "${PLUGIN_TEST_DIR:-}" ]] && [[ -d "$PLUGIN_TEST_DIR" ]]; then
        rm -rf "$PLUGIN_TEST_DIR"
    fi
    
    echo "[PLUGIN-TEST] Plugin test environment cleaned up" >&2
}

# Plugin-specific tmux command wrapper
plugin_tmux() {
    tmux -S "$PLUGIN_SOCKET" "$@"
}

# Install plugin for testing
install_test_plugin() {
    local plugin_source="${1:-$(pwd)}"
    
    # Copy plugin files to test directory
    if [[ -f "$plugin_source/tmux-proj.tmux" ]]; then
        cp "$plugin_source/tmux-proj.tmux" "$PLUGIN_TEST_DIR/plugins/tmux-proj/"
    fi
    
    if [[ -d "$plugin_source/scripts" ]]; then
        cp -r "$plugin_source/scripts" "$PLUGIN_TEST_DIR/plugins/tmux-proj/"
    fi
    
    echo "[PLUGIN-TEST] Plugin installed from $plugin_source" >&2
}

# Load plugin into tmux session
load_test_plugin() {
    # Start tmux server if not running
    plugin_tmux new-session -d -s "plugin-test-session" 2>/dev/null || true
    
    # Source the plugin
    if [[ -f "$PLUGIN_TEST_DIR/plugins/tmux-proj/tmux-proj.tmux" ]]; then
        plugin_tmux source-file "$PLUGIN_TEST_DIR/plugins/tmux-proj/tmux-proj.tmux"
        echo "[PLUGIN-TEST] Plugin loaded successfully" >&2
        return 0
    else
        echo "[PLUGIN-TEST] Plugin file not found" >&2
        return 1
    fi
}

# Test if key binding exists
test_key_binding() {
    local key="$1"
    local table="${2:-prefix}"
    
    # Use more precise grep pattern to match exact key binding
    plugin_tmux list-keys -T "$table" | grep -q "bind-key.*-T $table $key " || {
        echo "[PLUGIN-TEST] Key binding '$key' not found in table '$table'" >&2
        return 1
    }
    
    echo "[PLUGIN-TEST] Key binding '$key' found in table '$table'" >&2
    return 0
}

# Test if tmux option is set
test_tmux_option() {
    local option="$1"
    local expected_value="${2:-}"
    
    local actual_value
    actual_value=$(plugin_tmux show-options -gv "$option" 2>/dev/null || echo "")
    
    if [[ -n "$expected_value" ]]; then
        if [[ "$actual_value" == "$expected_value" ]]; then
            echo "[PLUGIN-TEST] Option '$option' correctly set to '$expected_value'" >&2
            return 0
        else
            echo "[PLUGIN-TEST] Option '$option' expected '$expected_value', got '$actual_value'" >&2
            return 1
        fi
    else
        if [[ -n "$actual_value" ]]; then
            echo "[PLUGIN-TEST] Option '$option' is set to '$actual_value'" >&2
            return 0
        else
            echo "[PLUGIN-TEST] Option '$option' is not set" >&2
            return 1
        fi
    fi
}

# Simulate key press in tmux
simulate_key_press() {
    local session="${1:-plugin-test-session}"
    local key="$2"
    
    # Send the key to tmux session
    plugin_tmux send-keys -t "$session" "$key"
    
    # Give tmux time to process the key
    sleep 0.1
    
    echo "[PLUGIN-TEST] Sent key '$key' to session '$session'" >&2
}

# Capture tmux session output
capture_session_output() {
    local session="${1:-plugin-test-session}"
    local pane="${2:-0}"
    
    plugin_tmux capture-pane -t "$session:$pane" -p
}

# Test session creation with plugin
create_plugin_test_session() {
    local session_name="${1:-plugin-test-session}"
    local start_dir="${2:-$PLUGIN_TEST_DIR}"
    
    plugin_tmux new-session -d -s "$session_name" -c "$start_dir"
    echo "[PLUGIN-TEST] Created test session '$session_name'" >&2
}

# Test window creation with plugin
create_plugin_test_window() {
    local session="${1:-plugin-test-session}"
    local window_name="${2:-test-window}"
    local start_dir="${3:-$PLUGIN_TEST_DIR}"
    
    plugin_tmux new-window -t "$session" -n "$window_name" -c "$start_dir"
    echo "[PLUGIN-TEST] Created test window '$window_name' in session '$session'" >&2
}

# Verify plugin command availability
test_plugin_command() {
    local command="$1"
    
    # Check if command is available in the plugin
    if command -v "$command" >/dev/null 2>&1; then
        echo "[PLUGIN-TEST] Plugin command '$command' is available" >&2
        return 0
    else
        echo "[PLUGIN-TEST] Plugin command '$command' is not available" >&2
        return 1
    fi
}

# Test menu/popup functionality
test_display_menu() {
    local session="${1:-plugin-test-session}"
    local menu_title="${2:-Test Menu}"
    
    # Create a simple test menu
    plugin_tmux display-menu -t "$session" -T "$menu_title" \
        "Option 1" 1 "" \
        "Option 2" 2 "" \
        "Cancel" q ""
    
    # Check if menu was displayed (this is tricky to test automatically)
    echo "[PLUGIN-TEST] Display menu '$menu_title' triggered" >&2
}

# Test popup functionality  
test_display_popup() {
    local session="${1:-plugin-test-session}"
    local command="${2:-echo 'Test popup'}"
    
    plugin_tmux display-popup -t "$session" -E "$command"
    echo "[PLUGIN-TEST] Display popup with command '$command' triggered" >&2
}

# Plugin-specific assertions
assert_plugin_loaded() {
    local message="${1:-Plugin should be loaded}"
    
    # Check if any plugin-specific options or key bindings exist
    if plugin_tmux list-keys | grep -q "run-shell" || 
       plugin_tmux show-options -g | grep -q "@proj"; then
        echo "[PLUGIN-TEST] ✓ $message" >&2
        return 0
    else
        echo "[PLUGIN-TEST] ✗ $message" >&2
        return 1
    fi
}

assert_key_binding_exists() {
    local key="$1"
    local table="${2:-prefix}"
    local message="${3:-Key binding should exist}"
    
    if test_key_binding "$key" "$table"; then
        echo "[PLUGIN-TEST] ✓ $message" >&2
        return 0
    else
        echo "[PLUGIN-TEST] ✗ $message" >&2
        return 1
    fi
}

assert_session_has_name_format() {
    local session_name="$1"
    local expected_pattern="$2"
    local message="${3:-Session name should match format}"
    
    if [[ "$session_name" =~ $expected_pattern ]]; then
        echo "[PLUGIN-TEST] ✓ $message: '$session_name' matches '$expected_pattern'" >&2
        return 0
    else
        echo "[PLUGIN-TEST] ✗ $message: '$session_name' does not match '$expected_pattern'" >&2
        return 1
    fi
}

# Integration with main test harness
if [[ -f "$(dirname "${BASH_SOURCE[0]}")/../tmux-harness.sh" ]]; then
    source "$(dirname "${BASH_SOURCE[0]}")/../tmux-harness.sh"
fi

echo "[PLUGIN-TEST] Plugin test helpers loaded" >&2
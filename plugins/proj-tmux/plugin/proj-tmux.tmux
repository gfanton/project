#!/usr/bin/env bash
# tmux-proj plugin - Main entry point
# Integrates proj CLI tool with tmux for seamless project and workspace management

CURRENT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SCRIPTS_DIR="$CURRENT_DIR/scripts"

# Default configuration
default_proj_key="P"
default_proj_popup_key="C-p"
default_proj_auto_session="on"
default_proj_show_status="on"
default_proj_session_format="proj-#{org}-#{name}"
default_proj_window_format="#{branch}"

# Get tmux option with default
tmux_option() {
    local option="$1"
    local default="$2"
    local value

    value=$(tmux show-option -gqv "$option" 2>/dev/null)
    if [[ -z "$value" ]]; then
        echo "$default"
    else
        echo "$value"
    fi
}

# Set up user configuration variables
setup_user_options() {
    # Main key binding (default: P)
    tmux set-option -gq "@proj_key" "$(tmux_option "@proj_key" "$default_proj_key")"

    # Popup key binding (default: C-p)
    tmux set-option -gq "@proj_popup_key" "$(tmux_option "@proj_popup_key" "$default_proj_popup_key")"

    # Auto create sessions (default: on)
    tmux set-option -gq "@proj_auto_session" "$(tmux_option "@proj_auto_session" "$default_proj_auto_session")"

    # Show in status bar (default: on)
    tmux set-option -gq "@proj_show_status" "$(tmux_option "@proj_show_status" "$default_proj_show_status")"

    # Session name format
    tmux set-option -gq "@proj_session_format" "$(tmux_option "@proj_session_format" "$default_proj_session_format")"

    # Window name format
    tmux set-option -gq "@proj_window_format" "$(tmux_option "@proj_window_format" "$default_proj_window_format")"
}

# Set up key bindings
setup_key_bindings() {
    local proj_key proj_popup_key

    proj_key=$(tmux_option "@proj_key" "$default_proj_key")
    proj_popup_key=$(tmux_option "@proj_popup_key" "$default_proj_popup_key")

    # Main project menu (Prefix + P)
    tmux bind-key "$proj_key" run-shell "$SCRIPTS_DIR/project_menu.sh"

    # Quick project popup (Prefix + Ctrl+P)
    tmux bind-key "$proj_popup_key" run-shell "$SCRIPTS_DIR/project_popup.sh"

    # Session switcher (Prefix + S) - override default
    tmux bind-key "S" run-shell "$SCRIPTS_DIR/session_switcher.sh"

    # Workspace menu (Prefix + W) - override default
    tmux bind-key "W" run-shell "$SCRIPTS_DIR/workspace_menu.sh"
}

# Set up status bar integration
setup_status_bar() {
    local show_status
    show_status=$(tmux_option "@proj_show_status" "$default_proj_show_status")

    if [[ "$show_status" == "on" ]]; then
        # Add project info to status bar
        # This can be customized by users in their tmux.conf
        tmux set-option -gq "status-right-length" "100"

        # Example status integration (users can customize)
        local current_right
        current_right=$(tmux show-option -gqv "status-right")

        # Only add if not already present
        if [[ "$current_right" != *"#($SCRIPTS_DIR/status_info.sh)"* ]]; then
            tmux set-option -gq "status-right" "#($SCRIPTS_DIR/status_info.sh) $current_right"
        fi
    fi
}

# Verify proj-tmux binary is available
check_dependencies() {
    if ! command -v proj-tmux >/dev/null 2>&1; then
        tmux display-message "Error: proj-tmux binary not found in PATH"
        return 1
    fi

    if ! command -v proj >/dev/null 2>&1; then
        tmux display-message "Error: proj binary not found in PATH"
        return 1
    fi

    return 0
}

# Main plugin initialization
main() {
    # Verify dependencies
    if ! check_dependencies; then
        return 1
    fi

    # Set up plugin
    setup_user_options
    setup_key_bindings
    setup_status_bar

    # Display success message (optional, can be disabled)
    # tmux display-message "tmux-proj plugin loaded"
}

# Run main initialization
main "$@"

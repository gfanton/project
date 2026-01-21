#!/usr/bin/env bash
#
# tmux-proj plugin - Main entry point
# Integrates proj CLI tool with tmux for seamless project and workspace management
#
# Requirements:
#   - proj binary in PATH
#   - proj-tmux binary in PATH
#
# Usage: run-shell /path/to/proj-tmux.tmux
#
# Configuration options (set in tmux.conf):
#   @proj_key          - Main key binding (default: P)
#   @proj_popup_key    - Popup key binding (default: C-p)
#   @proj_window_key   - Window popup key (default: C-w)
#   @proj_auto_session - Auto create sessions (default: on)
#   @proj_show_status  - Show in status bar (default: on)
#

set -o errexit
set -o nounset
set -o pipefail

readonly CURRENT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
readonly SCRIPTS_DIR="${CURRENT_DIR}/scripts"

# ---- Default Configuration
readonly DEFAULT_PROJ_KEY="P"
readonly DEFAULT_PROJ_POPUP_KEY="C-p"
readonly DEFAULT_PROJ_WINDOW_KEY="C-w"
readonly DEFAULT_PROJ_AUTO_SESSION="on"
readonly DEFAULT_PROJ_SHOW_STATUS="on"
readonly DEFAULT_PROJ_SESSION_FORMAT="proj-#{org}-#{name}"
readonly DEFAULT_PROJ_WINDOW_FORMAT="#{branch}"

# ---- Functions

# Log error to stderr
err() {
    printf '%s\n' "ERROR: $*" >&2
}

# Get tmux option with default
tmux_option() {
    local option="$1"
    local default="$2"
    local value

    value="$(tmux show-option -gqv "${option}" 2>/dev/null)" || true
    if [[ -z "${value}" ]]; then
        printf '%s\n' "${default}"
    else
        printf '%s\n' "${value}"
    fi
}

# Set up user configuration variables
setup_user_options() {
    # Main key binding (default: P)
    tmux set-option -gq "@proj_key" "$(tmux_option "@proj_key" "${DEFAULT_PROJ_KEY}")"

    # Popup key binding (default: C-p)
    tmux set-option -gq "@proj_popup_key" "$(tmux_option "@proj_popup_key" "${DEFAULT_PROJ_POPUP_KEY}")"

    # Window popup key binding (default: C-w)
    tmux set-option -gq "@proj_window_key" "$(tmux_option "@proj_window_key" "${DEFAULT_PROJ_WINDOW_KEY}")"

    # Auto create sessions (default: on)
    tmux set-option -gq "@proj_auto_session" "$(tmux_option "@proj_auto_session" "${DEFAULT_PROJ_AUTO_SESSION}")"

    # Show in status bar (default: on)
    tmux set-option -gq "@proj_show_status" "$(tmux_option "@proj_show_status" "${DEFAULT_PROJ_SHOW_STATUS}")"

    # Session name format
    tmux set-option -gq "@proj_session_format" "$(tmux_option "@proj_session_format" "${DEFAULT_PROJ_SESSION_FORMAT}")"

    # Window name format
    tmux set-option -gq "@proj_window_format" "$(tmux_option "@proj_window_format" "${DEFAULT_PROJ_WINDOW_FORMAT}")"
}

# Set up key bindings
setup_key_bindings() {
    local proj_key
    local proj_popup_key
    local proj_window_key

    proj_key="$(tmux_option "@proj_key" "${DEFAULT_PROJ_KEY}")"
    proj_popup_key="$(tmux_option "@proj_popup_key" "${DEFAULT_PROJ_POPUP_KEY}")"
    proj_window_key="$(tmux_option "@proj_window_key" "${DEFAULT_PROJ_WINDOW_KEY}")"

    # Main project menu (Prefix + P)
    tmux bind-key "${proj_key}" run-shell "${SCRIPTS_DIR}/project_menu.sh"

    # Quick project popup (Prefix + Ctrl+P) - for sessions
    tmux bind-key "${proj_popup_key}" run-shell "${SCRIPTS_DIR}/project_popup.sh"

    # Quick workspace popup (Prefix + Ctrl+W) - for windows
    tmux bind-key "${proj_window_key}" run-shell "${SCRIPTS_DIR}/window_popup.sh"

    # Session switcher (Prefix + S) - override default
    tmux bind-key "S" run-shell "${SCRIPTS_DIR}/session_switcher.sh"

    # Workspace menu (Prefix + W) - override default
    tmux bind-key "W" run-shell "${SCRIPTS_DIR}/workspace_menu.sh"
}

# Set up status bar integration
setup_status_bar() {
    local show_status
    show_status="$(tmux_option "@proj_show_status" "${DEFAULT_PROJ_SHOW_STATUS}")"

    if [[ "${show_status}" == "on" ]]; then
        # Add project info to status bar
        # This can be customized by users in their tmux.conf
        tmux set-option -gq "status-right-length" "100"

        # Example status integration (users can customize)
        local current_right
        current_right="$(tmux show-option -gqv "status-right")" || true

        # Only add if not already present
        if [[ "${current_right}" != *"#(${SCRIPTS_DIR}/status_info.sh)"* ]]; then
            if [[ -n "${current_right}" ]]; then
                tmux set-option -gq "status-right" "#(${SCRIPTS_DIR}/status_info.sh) ${current_right}"
            else
                tmux set-option -gq "status-right" "#(${SCRIPTS_DIR}/status_info.sh)"
            fi
        fi
    fi
}

# Verify proj-tmux binary is available and store paths for scripts
check_dependencies() {
    local proj_bin proj_tmux_bin

    # Check if paths were pre-set (e.g., by make dev-tmux for development)
    proj_tmux_bin="$(tmux show-environment -g PROJ_TMUX_BIN 2>/dev/null | cut -d= -f2-)"
    proj_bin="$(tmux show-environment -g PROJ_BIN 2>/dev/null | cut -d= -f2-)"

    # Validate pre-set proj-tmux path or find in PATH
    if [[ -n "$proj_tmux_bin" && -x "$proj_tmux_bin" ]]; then
        : # Already set and executable
    elif proj_tmux_bin="$(command -v proj-tmux 2>/dev/null)"; then
        : # Found in PATH
    else
        err "proj-tmux binary not found in PATH"
        tmux display-message "Error: proj-tmux binary not found in PATH"
        return 1
    fi

    # Validate pre-set proj path or find in PATH
    if [[ -n "$proj_bin" && -x "$proj_bin" ]]; then
        : # Already set and executable
    elif proj_bin="$(command -v proj 2>/dev/null)"; then
        : # Found in PATH
    else
        err "proj binary not found in PATH"
        tmux display-message "Error: proj binary not found in PATH"
        return 1
    fi

    # Store paths for use by scripts running in popup environments
    # where PATH may not include the binary locations
    tmux set-environment -g PROJ_BIN "$proj_bin"
    tmux set-environment -g PROJ_TMUX_BIN "$proj_tmux_bin"

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

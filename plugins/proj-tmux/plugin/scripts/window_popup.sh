#!/usr/bin/env bash
# Window picker popup using tmux display-popup with fzf
# Creates tmux windows for workspaces

set -euo pipefail
IFS=$'\n\t'

# Get binary paths from tmux environment (set by plugin at load time)
_proj_tmux_bin="${PROJ_TMUX_BIN:-}"
if [[ -z "${_proj_tmux_bin}" ]]; then
    _proj_tmux_bin="$(tmux show-environment -g PROJ_TMUX_BIN 2>/dev/null | cut -d= -f2-)" || true
fi
if [[ -z "${_proj_tmux_bin}" ]] || [[ ! -x "${_proj_tmux_bin}" ]]; then
    _proj_tmux_bin="proj-tmux"
fi
readonly PROJ_TMUX_BIN="${_proj_tmux_bin}"

# Logging configuration
LOG_FILE="${TMPDIR:-/tmp}/proj-tmux-window-popup.log"
DEBUG_MODE="${PROJ_DEBUG:-0}"

# Logging functions
log_debug() {
    if [[ "${DEBUG_MODE}" == "1" ]]; then
        echo "[$(date '+%Y-%m-%d %H:%M:%S')] DEBUG: $*" >> "${LOG_FILE}"
    fi
}

log_info() {
    echo "[$(date '+%Y-%m-%d %H:%M:%S')] INFO: $*" >> "${LOG_FILE}"
}

log_error() {
    echo "[$(date '+%Y-%m-%d %H:%M:%S')] ERROR: $*" >> "${LOG_FILE}"
    echo "[$(date '+%Y-%m-%d %H:%M:%S')] ERROR: $*" >&2
}

# Initialize logging
log_info "=== Starting window_popup.sh ==="
log_debug "Debug mode enabled"

# Check if fzf is available
check_fzf() {
    log_debug "Checking if fzf is available"
    if ! command -v fzf >/dev/null 2>&1; then
        log_error "fzf not found"
        tmux display-message "Error: fzf not found. Install fzf for popup functionality."
        return 1
    fi
    log_debug "fzf is available"
    return 0
}

# Get current project from session or directory
get_current_project() {
    # First try to get from tmux session
    local current_session
    current_session="$(tmux display-message -p '#{session_name}' 2>/dev/null)" || current_session=""

    if [[ "${current_session}" == proj-* ]]; then
        # Extract project from session name: proj-org-name -> org/name
        local remainder="${current_session#proj-}"
        local parts
        IFS='-' read -ra parts <<< "${remainder}"
        if [[ ${#parts[@]} -ge 2 ]]; then
            local name="${parts[-1]}"
            local org_parts=("${parts[@]:0:${#parts[@]}-1}")
            local org
            org="$(IFS='-'; echo "${org_parts[*]}")"
            echo "${org}/${name}"
            return 0
        fi
    fi

    # Fall back to current directory
    "${PROJ_TMUX_BIN}" session current 2>/dev/null | grep "Current directory project:" | cut -d':' -f2 | xargs || echo ""
}

# Check if we can show a popup (need active tmux client)
can_show_popup() {
    log_debug "Checking if popup can be displayed"
    # Check if we have an active tmux client that can display popups
    if tmux list-clients >/dev/null 2>&1 && [[ -n "${TMUX:-}" ]]; then
        log_debug "Popup can be displayed"
        return 0
    else
        log_debug "Cannot display popup: no active tmux client or not in tmux"
        return 1
    fi
}

# Show window/workspace popup
show_window_popup() {
    log_info "Showing window popup"
    if ! check_fzf; then
        return 1
    fi

    if ! can_show_popup; then
        log_error "Cannot show popup: no active tmux client"
        tmux display-message "Cannot show popup: no active tmux client. Use from within tmux session."
        return 1
    fi

    log_debug "Executing tmux display-popup"
    # Use simplified wrapper script
    local popup_cmd
    popup_cmd="$(dirname "${BASH_SOURCE[0]}")/window_picker.sh"

    if tmux display-popup -E -w 90% -h 80% -d "#{pane_current_path}" \
        -T 'Project/Workspace Selection (Window)' \
        "${popup_cmd}"; then
        log_info "Popup executed successfully"
    else
        local exit_code=$?
        log_error "Popup failed with exit code: ${exit_code}"
        tmux display-message "Popup failed. Try using Prefix+W for menu instead."
        return "${exit_code}"
    fi
}

# Main execution
main() {
    log_info "Starting main execution with args: $*"
    # Always use fzf popup
    if ! check_fzf; then
        log_error "fzf not available"
        tmux display-message "Error: fzf is required for workspace selection. Please install fzf."
        return 1
    fi

    if ! show_window_popup; then
        local exit_code=$?
        log_error "show_window_popup failed with exit code: ${exit_code}"
        return "${exit_code}"
    fi
    log_info "Main execution completed successfully"
}

# Set up error handling
trap 'log_error "Script failed on line $LINENO with exit code $?"' ERR

main "$@"

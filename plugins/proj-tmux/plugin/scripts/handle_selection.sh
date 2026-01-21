#!/usr/bin/env bash
# Helper script to handle project/workspace selection
# Usage: handle_selection.sh <session|window> <selection>

set -euo pipefail
IFS=$'\n\t'

# Get binary paths from tmux environment (set by plugin at load time)
_proj_tmux_bin="${PROJ_TMUX_BIN:-}"
if [[ -z "${_proj_tmux_bin}" ]]; then
    _proj_tmux_bin="$(tmux show-environment -g PROJ_TMUX_BIN 2>/dev/null | cut -d= -f2-)" || true
fi
# Fallback to PATH lookup if not set or not executable
if [[ -z "${_proj_tmux_bin}" ]] || [[ ! -x "${_proj_tmux_bin}" ]]; then
    _proj_tmux_bin="proj-tmux"
fi
readonly PROJ_TMUX_BIN="${_proj_tmux_bin}"

# Log error to stderr (visible in tmux logs with PROJ_DEBUG=1)
log_error() {
    echo "[ERROR] [handle_selection.sh] $*" >&2
}

# Get current tmux session name
get_current_session() {
    tmux display-message -p '#{session_name}' 2>/dev/null || echo ""
}

# Validate arguments
action="${1:-}"
selection="${2:-}"

if [[ -z "${action}" || -z "${selection}" ]]; then
    log_error "Usage: handle_selection.sh <session|window> <selection>"
    exit 1
fi

# Change to code directory
cd "${HOME}/code" || {
    log_error "Failed to change to code directory: ${HOME}/code"
    exit 1
}

# Handle workspace syntax (project:workspace)
if [[ "${selection}" == *:* ]]; then
    project="${selection%%:*}"
    workspace="${selection#*:}"

    if [[ "${action}" == "session" ]]; then
        # Create session (with --no-switch to get session name) and then create window for workspace
        session_name="$("${PROJ_TMUX_BIN}" session create --no-switch "${project}" 2>/dev/null)" || {
            log_error "Failed to create session for project: ${project}"
            exit 1
        }
        if ! "${PROJ_TMUX_BIN}" window create --no-switch "${workspace}" "${project}" 2>&1; then
            log_error "Failed to create window for workspace: ${workspace} in project: ${project}"
            exit 1
        fi
        # Output session name for parent script to capture and switch
        echo "${session_name}"
    else
        # Create window for workspace in current session
        current_session="$(get_current_session)"
        if [[ -n "${current_session}" ]]; then
            if ! "${PROJ_TMUX_BIN}" window create --session "${current_session}" "${workspace}" "${project}" 2>&1; then
                log_error "Failed to create window for workspace: ${workspace} in session: ${current_session}"
                exit 1
            fi
        else
            # Fallback: create in project-derived session
            if ! "${PROJ_TMUX_BIN}" window create "${workspace}" "${project}" 2>&1; then
                log_error "Failed to create window for workspace: ${workspace} in project: ${project}"
                exit 1
            fi
        fi
    fi
else
    # Handle plain project
    if [[ "${action}" == "session" ]]; then
        # Create session (with --no-switch to get session name)
        session_name="$("${PROJ_TMUX_BIN}" session create --no-switch "${selection}" 2>/dev/null)" || {
            log_error "Failed to create session for project: ${selection}"
            exit 1
        }
        # Output session name for parent script to capture and switch
        echo "${session_name}"
    else
        # For windows, create a main window in current session
        current_session="$(get_current_session)"
        if [[ -n "${current_session}" ]]; then
            if ! "${PROJ_TMUX_BIN}" window create --session "${current_session}" main "${selection}" 2>&1; then
                log_error "Failed to create main window for project: ${selection} in session: ${current_session}"
                exit 1
            fi
        else
            # Fallback: create in project-derived session
            if ! "${PROJ_TMUX_BIN}" window create main "${selection}" 2>&1; then
                log_error "Failed to create main window for project: ${selection}"
                exit 1
            fi
        fi
    fi
fi

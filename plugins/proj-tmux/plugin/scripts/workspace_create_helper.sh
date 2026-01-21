#!/usr/bin/env bash
# Helper script for creating a new workspace from tmux command-prompt
# Usage: workspace_create_helper.sh <workspace_name> <project>
#
# This script exists to work around tmux command-prompt's %% substitution
# only working once when && is used in the command.

set -euo pipefail
IFS=$'\n\t'

# Get binary paths from tmux environment (set by plugin at load time)
_proj_bin="${PROJ_BIN:-}"
if [[ -z "${_proj_bin}" ]]; then
    _proj_bin="$(tmux show-environment -g PROJ_BIN 2>/dev/null | cut -d= -f2-)" || true
fi
if [[ -z "${_proj_bin}" ]] || [[ ! -x "${_proj_bin}" ]]; then
    _proj_bin="proj"
fi
readonly PROJ_BIN="${_proj_bin}"

_proj_tmux_bin="${PROJ_TMUX_BIN:-}"
if [[ -z "${_proj_tmux_bin}" ]]; then
    _proj_tmux_bin="$(tmux show-environment -g PROJ_TMUX_BIN 2>/dev/null | cut -d= -f2-)" || true
fi
if [[ -z "${_proj_tmux_bin}" ]] || [[ ! -x "${_proj_tmux_bin}" ]]; then
    _proj_tmux_bin="proj-tmux"
fi
readonly PROJ_TMUX_BIN="${_proj_tmux_bin}"

# Get current tmux session name
get_current_session() {
    tmux display-message -p '#{session_name}' 2>/dev/null || echo ""
}

# Validate arguments
workspace="${1:-}"
project="${2:-}"

if [[ -z "${workspace}" ]]; then
    tmux display-message "Error: workspace name is required"
    exit 1
fi

if [[ -z "${project}" ]]; then
    tmux display-message "Error: project name is required"
    exit 1
fi

# Add workspace using proj
if ! "${PROJ_BIN}" workspace add "${workspace}" "${project}" 2>&1; then
    tmux display-message "Error: failed to add workspace '${workspace}'"
    exit 1
fi

# Create window using proj-tmux in current session
current_session="$(get_current_session)"
if [[ -n "${current_session}" ]]; then
    if ! "${PROJ_TMUX_BIN}" window create --session "${current_session}" "${workspace}" "${project}" 2>&1; then
        tmux display-message "Error: failed to create window for workspace '${workspace}'"
        exit 1
    fi
else
    # Fallback: create in project-derived session
    if ! "${PROJ_TMUX_BIN}" window create "${workspace}" "${project}" 2>&1; then
        tmux display-message "Error: failed to create window for workspace '${workspace}'"
        exit 1
    fi
fi

tmux display-message "Created workspace '${workspace}' in ${project}"

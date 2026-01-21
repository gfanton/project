#!/usr/bin/env bash
# Session picker wrapper - simplified for tmux popup

set -euo pipefail
IFS=$'\n\t'

SCRIPT_DIR="$(dirname "${BASH_SOURCE[0]}")"

# Get binary paths from tmux environment (set by plugin at load time)
_proj_bin="${PROJ_BIN:-}"
if [[ -z "${_proj_bin}" ]]; then
    _proj_bin="$(tmux show-environment -g PROJ_BIN 2>/dev/null | cut -d= -f2-)" || true
fi
if [[ -z "${_proj_bin}" ]] || [[ ! -x "${_proj_bin}" ]]; then
    _proj_bin="proj"
fi
readonly PROJ_BIN="${_proj_bin}"

# Get temp file path from argument (for passing session name back to parent)
readonly SESSION_OUTPUT_FILE="${1:-}"

# Configure fzf with --print-query to capture query even when no matches
fzf_output="$("${PROJ_BIN}" list | sed 's/ - \[.*\]$//' | fzf \
    --prompt='⚡ Project/Workspace (session): ' \
    --height=80% \
    --border=rounded \
    --reverse \
    --cycle \
    --print-query \
    --header='Navigate: ↑↓ | Tab: Complete | Enter: Create Session | Esc: Cancel | Use : for workspaces' \
    --bind='tab:replace-query' \
    --bind='enter:accept' \
    --bind='esc:cancel' \
    --bind="change:reload:${PROJ_BIN} query --limit 50 -- {q} 2>/dev/null || ${PROJ_BIN} list | sed 's/ - \[.*\]$//'" \
)" || fzf_output=""

# Parse fzf output: line 1 is query, line 2 is selection (if any)
query="$(echo "${fzf_output}" | head -n1)"
selection="$(echo "${fzf_output}" | tail -n +2 | head -n1)"

# Use selection if available, otherwise use query (for creating new workspaces)
final_selection="${selection:-${query}}"

# Execute selection
if [[ -n "${final_selection}" ]]; then
    session_name="$("${SCRIPT_DIR}/handle_selection.sh" session "${final_selection}")"
    # Write session name to temp file if provided (for parent to switch to)
    if [[ -n "${SESSION_OUTPUT_FILE}" && -n "${session_name}" ]]; then
        echo "${session_name}" > "${SESSION_OUTPUT_FILE}"
    fi
fi

#!/usr/bin/env bash
# Window picker wrapper - simplified for tmux popup

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

_proj_tmux_bin="${PROJ_TMUX_BIN:-}"
if [[ -z "${_proj_tmux_bin}" ]]; then
    _proj_tmux_bin="$(tmux show-environment -g PROJ_TMUX_BIN 2>/dev/null | cut -d= -f2-)" || true
fi
if [[ -z "${_proj_tmux_bin}" ]] || [[ ! -x "${_proj_tmux_bin}" ]]; then
    _proj_tmux_bin="proj-tmux"
fi
readonly PROJ_TMUX_BIN="${_proj_tmux_bin}"

# Get current project
get_current_project() {
    # First try to get from tmux session
    local current_session
    current_session="$(tmux display-message -p '#{session_name}' 2>/dev/null)" || current_session=""

    if [[ "${current_session}" == proj-* ]]; then
        # Extract project from session name
        local remainder="${current_session#proj-}"

        # New format: proj-org_name (underscore separator)
        if [[ "${remainder}" == *_* ]]; then
            local org="${remainder%%_*}"
            local name="${remainder#*_}"
            echo "${org}/${name}"
            return 0
        fi

        # Legacy format fallback: proj-org-name (hyphen separator, ambiguous)
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

    # Fall back to proj-tmux session current command
    local result
    result="$("${PROJ_TMUX_BIN}" session current 2>/dev/null | grep -E "Current (project session|directory project):" | cut -d':' -f2 | xargs)" || result=""
    if [[ -n "${result}" ]]; then
        echo "${result}"
        return 0
    fi

    echo ""
}

# Get initial input for fzf (replaces eval)
get_initial_input() {
    local query="${1:-}"
    if [[ -n "${query}" ]]; then
        "${PROJ_BIN}" query --limit 50 "${query}" 2>/dev/null || echo 'Type workspace name after :'
    else
        "${PROJ_BIN}" list | sed 's/ - \[.*\]$//'
    fi
}

# Get current project for pre-population
current_project="$(get_current_project)"
initial_query=""

if [[ -n "${current_project}" ]]; then
    initial_query="${current_project}:"
fi

# Configure fzf with --print-query to capture query even when no matches
fzf_output="$(get_initial_input "${initial_query}" | fzf \
    --prompt='⚡ Project/Workspace (window): ' \
    --height=80% \
    --border=rounded \
    --reverse \
    --cycle \
    --print-query \
    --header='Navigate: ↑↓ | Tab: Complete | Enter: Create Window | Esc: Cancel | Use : for workspaces' \
    --bind='tab:replace-query' \
    --bind='enter:accept' \
    --bind='esc:cancel' \
    --bind="change:reload:${PROJ_BIN} query --limit 50 -- {q} 2>/dev/null || ${PROJ_BIN} list | sed 's/ - \[.*\]$//'" \
    --query="${initial_query}" \
)" || fzf_output=""

# Parse fzf output: line 1 is query, line 2 is selection (if any)
query="$(echo "${fzf_output}" | head -n1)"
selection="$(echo "${fzf_output}" | tail -n +2 | head -n1)"

# Use selection if available, otherwise use query (for creating new workspaces)
final_selection="${selection:-${query}}"

# Execute selection
if [[ -n "${final_selection}" ]]; then
    exec "${SCRIPT_DIR}/handle_selection.sh" window "${final_selection}"
fi

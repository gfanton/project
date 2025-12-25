#!/usr/bin/env bash
# Window picker wrapper - simplified for tmux popup

set -euo pipefail

SCRIPT_DIR="$(dirname "$0")"

# Get current project
get_current_project() {
    # First try to get from tmux session
    local current_session
    current_session=$(tmux display-message -p '#{session_name}' 2>/dev/null || echo "")

    if [[ "$current_session" == proj-* ]]; then
        # Extract project from session name
        local remainder="${current_session#proj-}"

        # New format: proj-org_name (underscore separator)
        if [[ "$remainder" == *_* ]]; then
            local org="${remainder%%_*}"
            local name="${remainder#*_}"
            echo "${org}/${name}"
            return 0
        fi

        # Legacy format fallback: proj-org-name (hyphen separator, ambiguous)
        local parts=(${remainder//-/ })
        if [[ ${#parts[@]} -ge 2 ]]; then
            local name="${parts[-1]}"
            local org="${parts[*]:0:${#parts[@]}-1}"
            org="${org// /-}"
            echo "${org}/${name}"
            return 0
        fi
    fi

    # Fall back to proj-tmux session current command
    local result
    result=$(proj-tmux session current 2>/dev/null | grep -E "Current (project session|directory project):" | cut -d':' -f2 | xargs)
    if [[ -n "$result" ]]; then
        echo "$result"
        return 0
    fi

    echo ""
}

# Get current project for pre-population
current_project=$(get_current_project)
initial_query=""
initial_input="proj list | sed 's/ - \[.*\]$//'"

if [[ -n "$current_project" ]]; then
    initial_query="${current_project}:"
    # If we have a current project, start with workspace results
    initial_input="proj query --limit 50 '$initial_query' 2>/dev/null || echo 'Type workspace name after :'"
fi

# Configure fzf
selection=$(eval "$initial_input" | fzf \
    --prompt='⚡ Project/Workspace (window): ' \
    --height=80% \
    --border=rounded \
    --reverse \
    --cycle \
    --header='Navigate: ↑↓ | Tab: Complete | Enter: Create Window | Esc: Cancel | Use : for workspaces' \
    --bind='tab:replace-query' \
    --bind='enter:accept' \
    --bind='esc:cancel' \
    --bind='change:reload:proj query --limit 50 -- "{q}" 2>/dev/null || proj list | sed "s/ - \[.*\]$//"' \
    --query="$initial_query" \
)

# Execute selection
if [[ -n "$selection" ]]; then
    exec "$SCRIPT_DIR/handle_selection.sh" window "$selection"
fi
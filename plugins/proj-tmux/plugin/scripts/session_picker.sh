#!/usr/bin/env bash
# Session picker wrapper - simplified for tmux popup

set -euo pipefail

SCRIPT_DIR="$(dirname "$0")"

# Configure fzf
selection=$(proj list | sed 's/ - \[.*\]$//' | fzf \
    --prompt='⚡ Project/Workspace (session): ' \
    --height=80% \
    --border=rounded \
    --reverse \
    --cycle \
    --header='Navigate: ↑↓ | Tab: Complete | Enter: Create Session | Esc: Cancel | Use : for workspaces' \
    --bind='tab:replace-query' \
    --bind='enter:accept' \
    --bind='esc:cancel' \
    --bind='change:reload:proj query --limit 50 {q} 2>/dev/null || proj list | sed "s/ - \[.*\]$//"' \
)

# Execute selection
if [[ -n "$selection" ]]; then
    exec "$SCRIPT_DIR/handle_selection.sh" session "$selection"
fi
#!/usr/bin/env bash
# Helper script to handle project/workspace selection
# Usage: handle_selection.sh <session|window> <selection>

set -euo pipefail

action="$1"
selection="$2"

# Change to code directory
cd ~/code || exit 1

# Handle workspace syntax (project:workspace)
if [[ "$selection" == *:* ]]; then
    project="${selection%%:*}"
    workspace="${selection#*:}"
    
    if [[ "$action" == "session" ]]; then
        # Create session and then create window for workspace
        proj-tmux session create "$project" >/dev/null 2>&1 && \
        proj-tmux window create "$workspace" "$project" >/dev/null 2>&1
    else
        # Create window for workspace
        proj-tmux window create "$workspace" "$project" >/dev/null 2>&1
    fi
else
    # Handle plain project
    if [[ "$action" == "session" ]]; then
        proj-tmux session create "$selection" >/dev/null 2>&1
    else
        # For windows, create a main window (requires session to exist)
        proj-tmux window create main "$selection" >/dev/null 2>&1
    fi
fi
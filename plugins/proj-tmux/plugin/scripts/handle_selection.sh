#!/usr/bin/env bash
# Helper script to handle project/workspace selection
# Usage: handle_selection.sh <session|window> <selection>

set -euo pipefail

action="$1"
selection="$2"

# Log error to stderr (visible in tmux logs with PROJ_DEBUG=1)
log_error() {
    echo "[ERROR] $*" >&2
}

# Change to code directory
cd ~/code || exit 1

# Handle workspace syntax (project:workspace)
if [[ "$selection" == *:* ]]; then
    project="${selection%%:*}"
    workspace="${selection#*:}"

    if [[ "$action" == "session" ]]; then
        # Create session and then create window for workspace
        if ! proj-tmux session create "$project" 2>&1; then
            log_error "Failed to create session for project: $project"
            exit 1
        fi
        if ! proj-tmux window create "$workspace" "$project" 2>&1; then
            log_error "Failed to create window for workspace: $workspace in project: $project"
            exit 1
        fi
    else
        # Create window for workspace
        if ! proj-tmux window create "$workspace" "$project" 2>&1; then
            log_error "Failed to create window for workspace: $workspace in project: $project"
            exit 1
        fi
    fi
else
    # Handle plain project
    if [[ "$action" == "session" ]]; then
        if ! proj-tmux session create "$selection" 2>&1; then
            log_error "Failed to create session for project: $selection"
            exit 1
        fi
    else
        # For windows, create a main window (requires session to exist)
        if ! proj-tmux window create main "$selection" 2>&1; then
            log_error "Failed to create main window for project: $selection"
            exit 1
        fi
    fi
fi
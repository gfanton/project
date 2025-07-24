#!/usr/bin/env bash
# Project picker popup using tmux display-popup with fzf

set -euo pipefail

# Check if fzf is available
check_fzf() {
    if ! command -v fzf >/dev/null 2>&1; then
        tmux display-message "Error: fzf not found. Install fzf for popup functionality."
        return 1
    fi
    return 0
}

# Get projects and let user select with fzf
project_picker() {
    local selected_project
    
    # Get projects and pipe to fzf
    selected_project=$(proj list 2>/dev/null | fzf \
        --prompt="Select project: " \
        --height=40% \
        --border \
        --header="Use ↑↓ to navigate, Enter to select, Esc to cancel" \
        --preview="echo 'Project: {}'" \
        --preview-window=down:2 \
    ) || {
        # User cancelled or no selection
        return 1
    }
    
    if [[ -n "$selected_project" ]]; then
        # Create/switch to project session
        proj-tmux session create "$selected_project"
    fi
}

# Show project popup
show_project_popup() {
    if ! check_fzf; then
        return 1
    fi
    
    # Use tmux display-popup to show the project picker
    tmux display-popup -E -w 80% -h 60% -d "#{pane_current_path}" \
        "bash -c '$(declare -f project_picker); project_picker'"
}

# Alternative fallback menu if fzf not available
show_simple_menu() {
    # Fall back to the menu-based approach
    "$(dirname "$0")/project_menu.sh"
}

# Main execution
main() {
    if check_fzf; then
        show_project_popup
    else
        show_simple_menu
    fi
}

main "$@"
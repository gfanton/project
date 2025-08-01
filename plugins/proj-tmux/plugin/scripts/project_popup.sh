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
    
    # Get projects with status info for better display
    local projects_with_status projects_clean
    projects_with_status=$(proj list 2>/dev/null)
    projects_clean=$(echo "$projects_with_status" | sed 's/ - \[.*\]$//')
    
    # Use fzf with enhanced preview
    selected_project=$(echo "$projects_clean" | fzf \
        --prompt="⚡ Select project: " \
        --height=80% \
        --border=rounded \
        --header="Navigate: ↑↓ | Select: Enter | Cancel: Esc | Search: type to filter" \
        --preview="echo 'Project: {}' && echo '' && echo 'Path: ~/code/{}'" \
        --preview-window=down:4:wrap \
        --cycle \
        --reverse \
        --bind='ctrl-u:preview-page-up,ctrl-d:preview-page-down' \
    ) || {
        # User cancelled or no selection
        return 1
    }
    
    if [[ -n "$selected_project" ]]; then
        # Create/switch to project session
        cd ~/code && proj-tmux session create "$selected_project"
    fi
}

# Show project popup
show_project_popup() {
    if ! check_fzf; then
        return 1
    fi
    
    # Use tmux display-popup to show the project picker
    tmux display-popup -E -w 90% -h 80% -d "#{pane_current_path}" \
        -t 'Project Selection' \
        "bash -c '$(declare -f project_picker check_fzf); project_picker'"
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
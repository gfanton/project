#!/usr/bin/env bash
# Status bar information script

set -euo pipefail

# Get current project and workspace info
get_status_info() {
    local session_name current_project current_window status_text
    
    # Get current session name
    session_name=$(tmux display-message -p '#{session_name}' 2>/dev/null || echo "")
    
    # Get current window name
    current_window=$(tmux display-message -p '#{window_name}' 2>/dev/null || echo "")
    
    if [[ "$session_name" == proj-* ]]; then
        # Extract project name from session
        current_project=$(echo "$session_name" | sed 's/^proj-//' | sed 's/-/\//' | sed 's/-/\//')
        
        # Build status text
        if [[ "$current_window" != "zsh" && "$current_window" != "$session_name" ]]; then
            # Show project and workspace
            status_text="🚀 $current_project:$current_window"
        else
            # Show just project
            status_text="🚀 $current_project"
        fi
    else
        # Not a project session, show minimal info
        status_text=""
    fi
    
    echo "$status_text"
}

# Main execution
main() {
    get_status_info
}

main "$@"
#!/usr/bin/env bash
# Session switcher - Enhanced version of default tmux session switcher

set -euo pipefail

# Get all tmux sessions with project info
get_sessions_with_info() {
    # Get all sessions
    tmux list-sessions -F "#{session_name}" 2>/dev/null | while IFS= read -r session; do
        if [[ "$session" == proj-* ]]; then
            # Extract project name from proj session
            local project_name
            project_name=$(echo "$session" | sed 's/^proj-//' | sed 's/-/\//1' | sed 's/-/\//1')
            echo "$session ($project_name)"
        else
            echo "$session"
        fi
    done || {
        echo "No sessions found"
        return 1
    }
}

# Generate session menu items
generate_session_menu() {
    local sessions
    sessions=$(get_sessions_with_info)
    
    if [[ "$sessions" == "No sessions found" ]]; then
        echo "'' 'No sessions found' ''"
        return
    fi
    
    local current_session
    current_session=$(tmux display-message -p '#{session_name}' 2>/dev/null || echo "")
    
    # Generate menu items
    echo "$sessions" | while IFS= read -r session_info; do
        local session_name display_name key command
        
        # Extract session name (part before parentheses or whole string)
        session_name=$(echo "$session_info" | cut -d' ' -f1)
        display_name="$session_info"
        
        # Mark current session
        if [[ "$session_name" == "$current_session" ]]; then
            display_name="→ $display_name"
        fi
        
        key="${session_name:0:1}"  # First character as key
        command="switch-client -t '$session_name'"
        
        echo "'$display_name' '$key' '$command'"
    done
}

# Show session switcher menu
show_session_menu() {
    local menu_items
    menu_items=$(generate_session_menu)
    
    if [[ -z "$menu_items" ]]; then
        tmux display-message "No sessions available"
        return
    fi
    
    # Build the display-menu command
    local menu_cmd="display-menu -T 'Switch Session'"
    
    # Add separator and new session option
    menu_cmd="$menu_cmd '' '' ''"  # separator
    menu_cmd="$menu_cmd 'New Project Session...' 'n' 'run-shell \"$(dirname "${BASH_SOURCE[0]}")/project_popup.sh\"'"
    menu_cmd="$menu_cmd '' '' ''"  # separator
    
    # Add session items
    while IFS= read -r item; do
        if [[ -n "$item" ]]; then
            menu_cmd="$menu_cmd $item"
        fi
    done <<< "$menu_items"
    
    # Execute the menu
    eval "tmux $menu_cmd"
}

# Main execution
main() {
    show_session_menu
}

main "$@"
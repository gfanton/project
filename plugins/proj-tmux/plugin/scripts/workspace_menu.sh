#!/usr/bin/env bash
# Workspace management menu

set -euo pipefail

# Get current project context
get_current_project() {
    # Try to get from current session name
    local current_session
    current_session=$(tmux display-message -p '#{session_name}' 2>/dev/null || echo "")
    
    if [[ "$current_session" == proj-* ]]; then
        # Extract project from session name: proj-org-name -> org/name
        echo "$current_session" | sed 's/^proj-//' | sed 's/-/\//' | sed 's/-/\//'
        return 0
    fi
    
    # Try to get from current directory
    proj-tmux session current 2>/dev/null | grep "Current directory project:" | cut -d':' -f2 | xargs || {
        return 1
    }
}

# Get workspaces for current project
get_workspaces() {
    local project
    project=$(get_current_project)
    
    if [[ -z "$project" ]]; then
        echo "error: Not in a project session"
        return 1
    fi
    
    # Get workspaces using proj-tmux
    proj-tmux window list "$project" 2>/dev/null | grep -v "Windows in project" | sed 's/^  //' || {
        echo "error: No workspaces found"
        return 1
    }
}

# Generate workspace menu items
generate_workspace_menu() {
    local project workspaces
    project=$(get_current_project)
    
    if [[ -z "$project" ]]; then
        echo "'' 'Not in project session' ''"
        return
    fi
    
    workspaces=$(get_workspaces)
    
    if [[ "$workspaces" == "error:"* ]]; then
        echo "'' '${workspaces}' ''"
        return
    fi
    
    # Add header
    echo "'Current Project: $project' '' ''"
    echo "'' '' ''"  # separator
    
    # Add workspace items
    echo "$workspaces" | while IFS= read -r workspace; do
        if [[ -n "$workspace" && "$workspace" != "Windows in project"* ]]; then
            local display_name="Switch to: $workspace"
            local key="${workspace:0:1}"
            local command="run-shell 'proj-tmux window switch \"$workspace\" \"$project\"'"
            
            echo "'$display_name' '$key' '$command'"
        fi
    done
    
    # Add separator and management options
    echo "'' '' ''"  # separator
    echo "'Create New Workspace...' 'c' 'command-prompt -p \"Workspace name:\" \"run-shell \\\"proj workspace add %% $project && proj-tmux window create %% $project\\\"\"'"
    echo "'List All Workspaces' 'l' 'new-window \"proj workspace list $project; read\"'"
}

# Show workspace menu
show_workspace_menu() {
    local menu_items
    menu_items=$(generate_workspace_menu)
    
    if [[ -z "$menu_items" ]]; then
        tmux display-message "Unable to load workspace menu"
        return
    fi
    
    # Build the display-menu command
    local menu_cmd="display-menu -T 'Workspace Management'"
    
    # Add menu items
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
    show_workspace_menu
}

main "$@"
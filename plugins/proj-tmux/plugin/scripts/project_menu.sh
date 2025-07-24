#!/usr/bin/env bash
# Project selection menu using tmux display-menu

set -euo pipefail

# Get list of projects from proj CLI
get_projects() {
    proj list 2>/dev/null | head -20 || {
        echo "error No projects found"
        return 1
    }
}

# Generate menu items for projects
generate_menu_items() {
    local projects
    projects=$(get_projects)
    
    if [[ "$projects" == "error"* ]]; then
        echo "'' 'No projects found' ''"
        return
    fi
    
    # Generate menu items in format: "display_name" "key" "command"
    echo "$projects" | while IFS= read -r project; do
        local display_name="$project"
        local key="${project:0:1}"  # First character as key
        local command="run-shell 'proj-tmux session create \"$project\"'"
        
        echo "'$display_name' '$key' '$command'"
    done
}

# Show project selection menu
show_project_menu() {
    local menu_items
    menu_items=$(generate_menu_items)
    
    if [[ -z "$menu_items" ]]; then
        tmux display-message "No projects available"
        return
    fi
    
    # Build the display-menu command
    local menu_cmd="display-menu -T 'Select Project'"
    
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
    show_project_menu
}

main "$@"
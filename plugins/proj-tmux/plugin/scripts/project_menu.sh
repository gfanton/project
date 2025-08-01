#!/usr/bin/env bash
# Project selection menu using tmux display-menu

set -euo pipefail

# Get list of projects from proj CLI
get_projects() {
    # Get clean project names without status info
    proj list 2>/dev/null | sed 's/ - \[.*\]$//' || {
        echo "error No projects found"
        return 1
    }
}

# Count total projects
get_project_count() {
    get_projects | wc -l | xargs
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
    echo "$projects" | head -15 | nl | while IFS=$'\t' read -r line_num project; do
        local display_name="$project"
        # Use numbers for first 9, then letters
        local key
        if [[ $line_num -le 9 ]]; then
            key="$line_num"
        else
            # Use letters for 10+
            key=$(printf "%c" $((87 + line_num)))  # a, b, c...
        fi
        local command="run-shell 'cd ~/code && proj-tmux session create \"$project\"'"
        
        echo "'$line_num. $display_name' '$key' '$command'"
    done
}

# Show project selection menu
show_project_menu() {
    local project_count
    project_count=$(get_project_count)
    
    # If we have many projects (>15), use fzf popup instead of menu
    if [[ "$project_count" -gt 15 ]] && command -v fzf >/dev/null 2>&1; then
        "$(dirname "$0")/project_popup.sh"
        return
    fi
    
    local menu_items
    menu_items=$(generate_menu_items)
    
    if [[ -z "$menu_items" ]]; then
        tmux display-message "No projects available"
        return
    fi
    
    # Build the display-menu command with better formatting
    local menu_cmd="display-menu -T 'Select Project ($project_count projects)'"
    
    # Add help separator
    menu_cmd="$menu_cmd '' 'For all projects use Ctrl+P (popup)' ''"
    menu_cmd="$menu_cmd '' '' ''"  # separator
    
    # Add menu items (limited to first 15 for readability)
    while IFS= read -r item; do
        if [[ -n "$item" ]]; then
            menu_cmd="$menu_cmd $item"
        fi
    done <<< "$menu_items"
    
    # Add more options if truncated
    if [[ "$project_count" -gt 15 ]]; then
        menu_cmd="$menu_cmd '' '' ''"  # separator
        menu_cmd="$menu_cmd 'Show All Projects (popup)' 'a' 'run-shell \"$(dirname "$0")/project_popup.sh\"'"
    fi
    
    # Execute the menu
    eval "tmux $menu_cmd"
}

# Main execution
main() {
    show_project_menu
}

main "$@"
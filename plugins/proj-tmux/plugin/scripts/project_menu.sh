#!/usr/bin/env bash
# Project selection menu using tmux display-menu

set -euo pipefail

# Logging configuration
LOG_FILE="${TMPDIR:-/tmp}/proj-tmux-menu.log"
DEBUG_MODE="${PROJ_DEBUG:-0}"

# Logging functions
log_debug() {
    if [[ "$DEBUG_MODE" == "1" ]]; then
        echo "[$(date '+%Y-%m-%d %H:%M:%S')] DEBUG: $*" >> "$LOG_FILE"
    fi
}

log_info() {
    echo "[$(date '+%Y-%m-%d %H:%M:%S')] INFO: $*" >> "$LOG_FILE"
}

log_error() {
    echo "[$(date '+%Y-%m-%d %H:%M:%S')] ERROR: $*" >> "$LOG_FILE"
    echo "[$(date '+%Y-%m-%d %H:%M:%S')] ERROR: $*" >&2
}

# Initialize logging
log_info "=== Starting project_menu.sh ==="
log_debug "Debug mode enabled"

# Get list of projects from proj CLI
get_projects() {
    log_debug "Getting projects list"
    # Get clean project names without status info
    local projects
    projects=$(proj list 2>/dev/null | sed 's/ - \[.*\]$//' || {
        log_error "Failed to get projects list"
        echo "error No projects found"
        return 1
    })
    log_debug "Found $(echo "$projects" | wc -l | xargs) projects"
    echo "$projects"
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
        # Fix quote handling: avoid nested single quotes completely
        local escaped_project="${project//\'/\\\'}"

        # Use printf for consistent formatting
        printf "%s|%s|%s\n" "$line_num. $display_name" "$key" "run-shell \"cd ~/code && proj-tmux session create '$escaped_project'\""
    done
}

# Check if we're in an interactive tmux context
is_interactive_context() {
    # Check if we have a controlling terminal
    [[ -t 0 ]] && [[ -t 1 ]] && [[ -t 2 ]]
}

# Show project selection menu
show_project_menu() {
    log_info "Showing project selection menu"
    # Always use fzf popup if available
    if command -v fzf >/dev/null 2>&1; then
        log_info "Using fzf popup for project selection"
        "$(dirname "$0")/project_popup.sh"
        return
    fi
    
    # Fall back to menu only if fzf is not available
    log_info "fzf not available, using menu"
    local project_count
    project_count=$(get_project_count)
    log_debug "Project count: $project_count"

    local menu_items
    menu_items=$(generate_menu_items)
    log_debug "Generated menu items ($(echo "$menu_items" | wc -l | xargs) items)"

    if [[ -z "$menu_items" ]]; then
        log_error "No menu items generated"
        tmux display-message "No projects available"
        return
    fi

    # Build the display-menu command using array for proper quote handling
    local menu_args=()
    menu_args+=("display-menu")
    menu_args+=("-T" "Select Project ($project_count projects)")

    # Add help separator
    menu_args+=("" "For all projects use Ctrl+P (popup)" "")
    menu_args+=("" "" "")  # separator

    # Add menu items (limited to first 15 for readability)
    local item_count=0
    while IFS='|' read -r display_name key command && [[ $item_count -lt 15 ]]; do
        if [[ -n "$display_name" ]]; then
            menu_args+=("$display_name" "$key" "$command")
            ((item_count++))
            log_debug "Added menu item: $display_name"
        fi
    done <<< "$menu_items"

    # Add more options if truncated
    if [[ "$project_count" -gt 15 ]]; then
        menu_args+=("" "" "")  # separator
        menu_args+=("Show All Projects (popup)" "a" "run-shell $(dirname "$0")/project_popup.sh")
    fi

    log_debug "Final menu command has ${#menu_args[@]} arguments"
    log_debug "Menu command: tmux ${menu_args[*]}"

    # Execute the menu using array expansion for proper quoting
    if ! tmux "${menu_args[@]}"; then
        local exit_code=$?
        log_error "Menu execution failed with exit code: $exit_code"
        log_error "Command was: tmux ${menu_args[*]}"
        return $exit_code
    fi

    log_info "Menu executed successfully"
}

# Main execution
main() {
    log_info "Starting main execution with args: $*"
    if ! show_project_menu; then
        local exit_code=$?
        log_error "show_project_menu failed with exit code: $exit_code"
        return $exit_code
    fi
    log_info "Main execution completed successfully"
}

# Set up error handling
trap 'log_error "Script failed on line $LINENO with exit code $?"' ERR

main "$@"

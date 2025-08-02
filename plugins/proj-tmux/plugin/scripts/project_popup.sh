#!/usr/bin/env bash
# Project/Workspace picker popup using tmux display-popup with fzf
# Creates tmux sessions based on selection

set -euo pipefail

# Logging configuration
LOG_FILE="${TMPDIR:-/tmp}/proj-tmux-popup.log"
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
log_info "=== Starting project_popup.sh ==="
log_debug "Debug mode enabled"

# Check if fzf is available
check_fzf() {
    log_debug "Checking if fzf is available"
    if ! command -v fzf >/dev/null 2>&1; then
        log_error "fzf not found"
        tmux display-message "Error: fzf not found. Install fzf for popup functionality."
        return 1
    fi
    log_debug "fzf is available"
    return 0
}

# Get projects and let user select with fzf
project_picker() {
    log_info "Starting project picker"
    local selected_project
    
    # Get projects with status info for better display
    local projects_with_status projects_clean
    log_debug "Getting projects list"
    projects_with_status=$(proj list 2>/dev/null || {
        log_error "Failed to get projects list"
        echo "Error getting projects" >&2
        return 1
    })
    projects_clean=$(echo "$projects_with_status" | sed 's/ - \[.*\]$//') 
    log_debug "Found $(echo "$projects_clean" | wc -l | xargs) projects"
    
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
        log_info "Selected project: $selected_project"
        # Create/switch to project session
        if cd ~/code && proj-tmux session create "$selected_project"; then
            log_info "Successfully created/switched to session for: $selected_project"
        else
            log_error "Failed to create/switch to session for: $selected_project"
            return 1
        fi
    else
        log_info "No project selected (user cancelled)"
    fi
}

# Check if we can show a popup (need active tmux client)
can_show_popup() {
    log_debug "Checking if popup can be displayed"
    # Check if we have an active tmux client that can display popups
    if tmux list-clients >/dev/null 2>&1 && [[ -n "${TMUX:-}" ]]; then
        log_debug "Popup can be displayed"
        return 0
    else
        log_debug "Cannot display popup: no active tmux client or not in tmux"
        return 1
    fi
}

# Show project popup
show_project_popup() {
    log_info "Showing project popup"
    if ! check_fzf; then
        return 1
    fi
    
    if ! can_show_popup; then
        log_error "Cannot show popup: no active tmux client"
        tmux display-message "Cannot show popup: no active tmux client. Use from within tmux session."
        return 1
    fi
    
    log_debug "Executing tmux display-popup"
    # Use simplified wrapper script
    local popup_cmd="$(dirname "$0")/session_picker.sh"
    
    if tmux display-popup -E -w 90% -h 80% -d "#{pane_current_path}" \
        -T 'Project/Workspace Selection (Session)' \
        "$popup_cmd"; then
        log_info "Popup executed successfully"
    else
        local exit_code=$?
        log_error "Popup failed with exit code: $exit_code"
        # If popup fails, show helpful message
        tmux display-message "Popup failed. Try using Prefix+P for menu instead."
        return $exit_code
    fi
}

# Alternative fallback menu if fzf not available
show_simple_menu() {
    # Fall back to the menu-based approach
    "$(dirname "$0")/project_menu.sh"
}

# Main execution
main() {
    log_info "Starting main execution with args: $*"
    # Always use fzf popup unless fzf is not available
    if ! check_fzf; then
        log_error "fzf not available"
        tmux display-message "Error: fzf is required for project selection. Please install fzf."
        return 1
    fi
    
    if ! show_project_popup; then
        local exit_code=$?
        log_error "show_project_popup failed with exit code: $exit_code"
        return $exit_code
    fi
    log_info "Main execution completed successfully"
}

# Set up error handling
trap 'log_error "Script failed on line $LINENO with exit code $?"' ERR

main "$@"
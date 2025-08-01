#!/usr/bin/env bash
# Enhanced status bar information script

set -euo pipefail

# Get tmux option with default
tmux_option() {
    local option="$1"
    local default="$2"
    local value
    
    value=$(tmux show-option -gqv "$option" 2>/dev/null)
    if [[ -z "$value" ]]; then
        echo "$default"
    else
        echo "$value"
    fi
}

# Get current project and workspace info
get_status_info() {
    local session_name current_project current_window status_text
    local session_count project_count workspace_indicator
    
    # Get current session name
    session_name=$(tmux display-message -p '#{session_name}' 2>/dev/null || echo "")
    
    # Get current window name
    current_window=$(tmux display-message -p '#{window_name}' 2>/dev/null || echo "")
    
    if [[ "$session_name" == proj-* ]]; then
        # Extract project name from session using proj-tmux
        current_project=$(proj-tmux session current 2>/dev/null | grep -oE '[^:]+/[^:]+' || echo "")
        
        if [[ -z "$current_project" ]]; then
            # Fallback to parsing session name
            current_project=$(echo "$session_name" | sed 's/^proj-//' | sed 's/-/\//' | sed 's/-/\//')
        fi
        
        # Get additional context
        session_count=$(tmux list-sessions 2>/dev/null | grep -c "^proj-" || echo "0")
        project_count=$(proj list 2>/dev/null | wc -l || echo "0")
        
        # Check if current window is a workspace
        workspace_indicator=""
        if [[ "$current_window" != "zsh" && "$current_window" != "$session_name" && "$current_window" != "0" ]]; then
            workspace_indicator=":$current_window"
        fi
        
        # Build status text with configurable format
        local show_counts="$(tmux_option "@proj_status_show_counts" "off")"
        local show_icons="$(tmux_option "@proj_status_show_icons" "on")"
        
        if [[ "$show_icons" == "on" ]]; then
            status_text="🚀 $current_project$workspace_indicator"
        else
            status_text="$current_project$workspace_indicator"
        fi
        
        if [[ "$show_counts" == "on" ]]; then
            status_text="$status_text [$session_count/$project_count]"
        fi
    else
        # Not a project session, show minimal info
        status_text=""
    fi
    
    echo "$status_text"
}

# Get project count for status
get_project_count() {
    proj list 2>/dev/null | wc -l || echo "0"
}

# Get active session count for status
get_session_count() {
    tmux list-sessions 2>/dev/null | grep -c "^proj-" || echo "0"
}

# Format status for different modes
format_status() {
    local mode="${1:-default}"
    
    case "$mode" in
        "compact")
            # Compact format: just project name
            local session_name current_project
            session_name=$(tmux display-message -p '#{session_name}' 2>/dev/null || echo "")
            if [[ "$session_name" == proj-* ]]; then
                current_project=$(echo "$session_name" | sed 's/^proj-//' | sed 's/-/\//' | sed 's/-/\//')
                echo "$current_project"
            fi
            ;;
        "full")
            # Full format with all details
            get_status_info
            ;;
        *)
            # Default format
            get_status_info
            ;;
    esac
}

# Main execution
main() {
    local mode="${1:-default}"
    format_status "$mode"
}

main "$@"
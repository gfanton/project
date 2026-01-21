#!/usr/bin/env bash
# Workspace management menu

set -euo pipefail
IFS=$'\n\t'

SCRIPT_DIR="$(dirname "${BASH_SOURCE[0]}")"

# Get binary paths from tmux environment (set by plugin at load time)
_proj_bin="${PROJ_BIN:-}"
if [[ -z "${_proj_bin}" ]]; then
    _proj_bin="$(tmux show-environment -g PROJ_BIN 2>/dev/null | cut -d= -f2-)" || true
fi
if [[ -z "${_proj_bin}" ]] || [[ ! -x "${_proj_bin}" ]]; then
    _proj_bin="proj"
fi
readonly PROJ_BIN="${_proj_bin}"

_proj_tmux_bin="${PROJ_TMUX_BIN:-}"
if [[ -z "${_proj_tmux_bin}" ]]; then
    _proj_tmux_bin="$(tmux show-environment -g PROJ_TMUX_BIN 2>/dev/null | cut -d= -f2-)" || true
fi
if [[ -z "${_proj_tmux_bin}" ]] || [[ ! -x "${_proj_tmux_bin}" ]]; then
    _proj_tmux_bin="proj-tmux"
fi
readonly PROJ_TMUX_BIN="${_proj_tmux_bin}"

# Get current project context
get_current_project() {
    # Try to get from current session name
    local current_session
    current_session="$(tmux display-message -p '#{session_name}' 2>/dev/null)" || current_session=""

    if [[ "${current_session}" == proj-* ]]; then
        # Extract project from session name: proj-org-name -> org/name
        echo "${current_session}" | sed 's/^proj-//' | sed 's/-/\//' | sed 's/-/\//'
        return 0
    fi

    # Try to get from current directory
    "${PROJ_TMUX_BIN}" session current 2>/dev/null | grep "Current directory project:" | cut -d':' -f2 | xargs || {
        return 1
    }
}

# Get workspaces for current project
get_workspaces() {
    local project
    project="$(get_current_project)" || project=""

    if [[ -z "${project}" ]]; then
        echo "error: Not in a project session"
        return 1
    fi

    # Get workspaces using proj-tmux
    "${PROJ_TMUX_BIN}" window list "${project}" 2>/dev/null | grep -v "Windows in project" | sed 's/^  //' || {
        echo "error: No workspaces found"
        return 1
    }
}

# Generate workspace menu items using pipe-separated format for safe parsing
generate_workspace_menu() {
    local project workspaces
    project="$(get_current_project)" || project=""

    if [[ -z "${project}" ]]; then
        echo "|Not in project session|"
        return
    fi

    workspaces="$(get_workspaces)" || workspaces=""

    if [[ "${workspaces}" == "error:"* ]]; then
        echo "|${workspaces}|"
        return
    fi

    # Add header
    echo "Current Project: ${project}||"
    echo "||"  # separator

    # Add workspace items
    echo "${workspaces}" | while IFS= read -r workspace; do
        if [[ -n "${workspace}" && "${workspace}" != "Windows in project"* ]]; then
            local display_name="Switch to: ${workspace}"
            local key="${workspace:0:1}"
            local command="run-shell '${PROJ_TMUX_BIN} window switch \"${workspace}\" \"${project}\"'"

            echo "${display_name}|${key}|${command}"
        fi
    done

    # Add separator and management options
    echo "||"  # separator
    # Use helper script to avoid tmux %% substitution issues with &&
    local helper_script="${SCRIPT_DIR}/workspace_create_helper.sh"
    echo "Create New Workspace...|c|command-prompt -p \"Workspace name:\" \"run-shell \\\"${helper_script} %% ${project}\\\"\""
    echo "List All Workspaces|l|new-window \"${PROJ_BIN} workspace list ${project}; read\""
}

# Show workspace menu
show_workspace_menu() {
    local menu_items
    menu_items="$(generate_workspace_menu)"

    if [[ -z "${menu_items}" ]]; then
        tmux display-message "Unable to load workspace menu"
        return
    fi

    # Build the display-menu command using array for proper quote handling
    local menu_args=()
    menu_args+=("display-menu")
    menu_args+=("-T" "Workspace Management")

    # Add menu items
    while IFS='|' read -r display_name key command; do
        menu_args+=("${display_name}" "${key}" "${command}")
    done <<< "${menu_items}"

    # Execute the menu using array expansion for proper quoting
    tmux "${menu_args[@]}"
}

# Main execution
main() {
    show_workspace_menu
}

main "$@"

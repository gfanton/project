#!/usr/bin/env bash
# View tmux plugin logs for debugging

set -euo pipefail

MENU_LOG="${TMPDIR:-/tmp}/proj-tmux-menu.log"
POPUP_LOG="${TMPDIR:-/tmp}/proj-tmux-popup.log"

show_usage() {
    cat << 'EOF'
Usage: view_logs.sh [OPTION]

View tmux plugin logs for debugging.

Options:
    -m, --menu      Show menu script logs only
    -p, --popup     Show popup script logs only  
    -f, --follow    Follow logs in real-time (like tail -f)
    -c, --clear     Clear all log files
    -h, --help      Show this help message

Without options, shows both menu and popup logs.

To enable debug logging, run:
    export PROJ_DEBUG=1

Then use your tmux key bindings (Prefix+P or Prefix+Ctrl+P) to generate logs.
EOF
}

show_menu_logs() {
    if [[ -f "$MENU_LOG" ]]; then
        echo "=== Menu Script Logs ($MENU_LOG) ==="
        cat "$MENU_LOG"
        echo ""
    else
        echo "No menu logs found at $MENU_LOG"
        echo "Enable debug mode with: export PROJ_DEBUG=1"
        echo ""
    fi
}

show_popup_logs() {
    if [[ -f "$POPUP_LOG" ]]; then
        echo "=== Popup Script Logs ($POPUP_LOG) ==="
        cat "$POPUP_LOG"
        echo ""
    else
        echo "No popup logs found at $POPUP_LOG"
        echo "Enable debug mode with: export PROJ_DEBUG=1"
        echo ""
    fi
}

follow_logs() {
    echo "Following tmux plugin logs (Ctrl+C to stop)..."
    echo "Use your tmux key bindings to generate log entries."
    echo ""
    
    # Create log files if they don't exist
    touch "$MENU_LOG" "$POPUP_LOG"
    
    # Follow both log files
    tail -f "$MENU_LOG" "$POPUP_LOG" 2>/dev/null || {
        echo "Error: Cannot follow log files. Make sure PROJ_DEBUG=1 is set and use tmux key bindings."
        return 1
    }
}

clear_logs() {
    echo "Clearing tmux plugin logs..."
    rm -f "$MENU_LOG" "$POPUP_LOG"
    echo "✓ Logs cleared"
}

main() {
    case "${1:-}" in
        -m|--menu)
            show_menu_logs
            ;;
        -p|--popup)
            show_popup_logs
            ;;
        -f|--follow)
            follow_logs
            ;;
        -c|--clear)
            clear_logs
            ;;
        -h|--help)
            show_usage
            ;;
        "")
            show_menu_logs
            show_popup_logs
            ;;
        *)
            echo "Unknown option: $1"
            echo ""
            show_usage
            exit 1
            ;;
    esac
}

main "$@"
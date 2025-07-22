# Simple mock tmux-proj plugin for testing
# This file contains raw tmux configuration commands

# Set default options
set-option -gq "@proj_key" "P"
set-option -gq "@proj_popup_key" "C-p"  
set-option -gq "@proj_auto_session" "on"

# Key bindings - use simple echo commands for testing
bind-key "P" run-shell "echo 'Project menu'"
bind-key "C-p" run-shell "echo 'Project popup'"
bind-key "S" run-shell "echo 'Session switcher'"
bind-key "W" run-shell "echo 'Workspace menu'"

# Test bindings that run scripts (if they exist)
bind-key "1" run-shell "echo 'Project menu script'"
bind-key "2" run-shell "echo 'Project picker script'"
bind-key "3" run-shell "echo 'Session manager script'"

# Status bar configuration
set-option -g status-right-length 100
set-option -g status-right "Project: testorg/webapp | #H"

# Plugin loaded indicator
set-option -g "@proj_loaded" "true"
# tmux-proj Plugin

A tmux plugin for seamless project and workspace management, integrating with the `proj` CLI tool.

## Features

- **Project Sessions**: Quick creation and switching between project-based tmux sessions
- **Workspace Windows**: Automatic window management for git worktree workspaces  
- **Interactive Menus**: Menu-based project and workspace selection
- **Popup Interface**: fzf-powered project picker (when fzf is available)
- **Status Bar Integration**: Show current project/workspace in tmux status bar
- **Customizable Key Bindings**: Configure your own key bindings and preferences

## Installation

### Using TPM (Tmux Plugin Manager)

Add this line to your `~/.tmux.conf`:

```bash
set -g @plugin 'gfanton/project'
```

Then press `prefix + I` to install.

### Manual Installation

1. Clone this repository:
```bash
git clone https://github.com/gfanton/project ~/.tmux/plugins/project
```

2. Add to your `~/.tmux.conf`:
```bash
run-shell ~/.tmux/plugins/project/plugins/proj-tmux/plugin/proj-tmux.tmux
```

3. Reload tmux configuration:
```bash
tmux source-file ~/.tmux.conf
```

## Prerequisites

- `proj` CLI tool must be installed and in PATH
- `proj-tmux` binary must be installed and in PATH
- `fzf` (optional, for popup interface)

## Default Key Bindings

| Key Binding | Action |
|-------------|--------|
| `Prefix + P` | Open project selection menu |
| `Prefix + Ctrl+P` | Open project picker popup (requires fzf) |
| `Prefix + S` | Enhanced session switcher |
| `Prefix + W` | Workspace management menu |

## Configuration Options

Add these options to your `~/.tmux.conf` to customize the plugin:

```bash
# Main key binding (default: P)
set -g @proj_key 'P'

# Popup key binding (default: C-p)
set -g @proj_popup_key 'C-p'

# Auto create sessions when switching (default: on)
set -g @proj_auto_session 'on'

# Show project info in status bar (default: on)
set -g @proj_show_status 'on'

# Session name format (default: proj-#{org}-#{name})
set -g @proj_session_format 'proj-#{org}-#{name}'

# Window name format (default: #{branch})
set -g @proj_window_format '#{branch}'
```

## Usage

### Project Management

1. **Open Project Menu**: Press `Prefix + P` to see a menu of available projects
2. **Quick Project Popup**: Press `Prefix + Ctrl+P` for an fzf-powered project picker
3. **Session Switching**: Press `Prefix + S` for enhanced session management

### Workspace Management

1. **Open Workspace Menu**: Press `Prefix + W` to manage workspaces in current project
2. **Create New Workspace**: Use the workspace menu to create new workspaces
3. **Switch Workspaces**: Select workspace windows directly from the menu

### Status Bar Integration

The plugin automatically adds project/workspace information to your tmux status bar:

- `🚀 org/project` - Shows current project
- `🚀 org/project:branch` - Shows current project and workspace

To customize the status bar, modify your tmux status-right configuration:

```bash
set -g status-right "#{@proj_status} [%Y-%m-%d %H:%M]"
```

## Workflow Examples

### Basic Project Workflow

1. Press `Prefix + P` to select a project
2. Plugin creates a tmux session named `proj-org-project`
3. Use `Prefix + W` to create and switch between workspace windows
4. Each workspace window is set to the correct workspace directory

### Combined Workspace Creation

```bash
# Traditional approach (2 commands)
proj workspace add feature org/project
proj-tmux window create feature org/project

# Or create a shell function for convenience
workspace-tmux() {
    proj workspace add "$1" "$2" && proj-tmux window create "$1" "$2"
}
```

## Troubleshooting

### "proj-tmux binary not found"

Ensure both `proj` and `proj-tmux` are installed and in your PATH:

```bash
which proj
which proj-tmux
```

### "fzf not found" 

The popup interface requires fzf. Install it or use the menu interface instead:

```bash
# macOS
brew install fzf

# Ubuntu/Debian
sudo apt install fzf
```

### Plugin not loading

Check your tmux configuration:

```bash
tmux show-options -g | grep @proj
```

## Integration with proj CLI

This plugin works seamlessly with the `proj` CLI tool:

- **Sessions**: Created using `proj-tmux session create <project>`
- **Windows**: Created using `proj-tmux window create <workspace> <project>`
- **Workspaces**: Managed using `proj workspace add/list/remove`

## License

MIT License - see the main project repository for details.
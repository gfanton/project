# Nix Flake Integration Guide

This guide shows you how to integrate the `project` tool into your Nix setup using flakes.

## 🚀 Quick Start

### Option 1: Direct Installation with Nix Flake

```bash
# Install directly from GitHub
nix profile install github:gfanton/project

# Or run without installing
nix run github:gfanton/project -- --help

# Enable shell integration
eval "$(proj init zsh)"
```

### Option 2: Integration with Home Manager

Add the project as an input to your `flake.nix`:

```nix
{
  inputs = {
    # ... your existing inputs
    project.url = "github:gfanton/project";
    project.inputs.nixpkgs.follows = "nixpkgs-unstable";
  };

  outputs = { self, nixpkgs, home-manager, project, ... }@inputs: {
    # In your home-manager configuration
    homeConfigurations.yourusername = home-manager.lib.homeManagerConfiguration {
      pkgs = nixpkgs.legacyPackages.x86_64-linux; # or your system
      modules = [
        {
          home.packages = [ project.packages.x86_64-linux.default ];
          
          # Enable shell integration automatically
          programs.zsh.initExtra = ''
            eval "$(${project.packages.x86_64-linux.default}/bin/proj init zsh)"
          '';
        }
      ];
    };
  };
}
```

### Option 3: Using Overlays (Advanced)

Add to your overlays:

```nix
{
  nixpkgs.overlays = [
    (final: prev: {
      project = inputs.project.packages.${prev.system}.default;
    })
  ];
}
```

Then use `pkgs.project` in your packages list.

## 🏠 Home Manager Integration

### Complete Example

Here's a complete example of integrating project into your home-manager setup:

```nix
# flake.nix
{
  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixpkgs-unstable";
    home-manager.url = "github:nix-community/home-manager";
    home-manager.inputs.nixpkgs.follows = "nixpkgs";
    
    # Add project
    project.url = "github:gfanton/project";
    project.inputs.nixpkgs.follows = "nixpkgs";
  };

  outputs = { self, nixpkgs, home-manager, project, ... }@inputs: {
    homeConfigurations.yourusername = home-manager.lib.homeManagerConfiguration {
      pkgs = nixpkgs.legacyPackages.aarch64-darwin; # or your system
      modules = [
        {
          # Install the package
          home.packages = [ inputs.project.packages.aarch64-darwin.default ];
          
          # Configure shell integration
          programs.zsh = {
            enable = true;
            initExtra = ''
              # Project shell integration - provides 'p' command
              eval "$(${inputs.project.packages.aarch64-darwin.default}/bin/proj init zsh)"
            '';
          };
          
          # Optional: Set up project configuration
          home.file.".projectrc".text = ''
            root = "${config.home.homeDirectory}/code"
            user = "yourusername"
            debug = false
          '';
        }
      ];
    };
  };
}
```

### Configuration Options

Create `~/.projectrc` (or use home.file as shown above):

```toml
# Root directory for all projects
root = "~/code"

# Default username for projects
user = "yourusername"

# Enable debug logging
debug = false
```

## 🧪 Development Environment

The project includes a development shell with all testing dependencies:

```bash
# Enter development environment
nix develop github:gfanton/project

# Or with direnv (create .envrc with: use flake github:gfanton/project)
echo "use flake github:gfanton/project" > .envrc
direnv allow
```

### Available Commands in Dev Shell

```bash
make build              # Build the project
make test               # Run Go unit tests
make test-shell         # Run Go-based shell tests
make test-integration   # Run BATS/Expect tests (requires Nix)
make test-nix           # Run all tests in Nix environment
./test-completion.sh    # Test enhanced completion with sample projects
```

## 🔍 Features Included

- **Enhanced zsh completion**: Shows up to 20 project options with menu selection
- **Fuzzy search**: Find projects with partial matches
- **Shell integration**: The `p` command for quick navigation
- **GitHub-style organization**: Projects stored as `username/project-name`
- **Git integration**: Automatic repository detection and status

## 📦 Package Contents

The Nix package includes:

- **Binary**: `/bin/proj` - The main executable
- **Shell completion**: `/share/zsh/site-functions/_project` - Zsh completion
- **Examples**: `/share/doc/project/examples/` - Test scripts and configurations

## 🚨 Troubleshooting

### Common Issues

1. **"proj: command not found"**
   - Make sure the package is in your `home.packages` or system packages
   - Restart your shell or run `hash -r`

2. **Shell completion not working**
   - Ensure zsh completion is enabled: `autoload -U compinit && compinit`
   - Check that the completion file exists in your fpath

3. **No projects found**
   - Create your first project: `proj new myproject`
   - Or clone one: `proj get username/repo`
   - Check your config: `proj list --all`

### Debug Mode

Enable debug logging:

```bash
export PROJECT_DEBUG=true
proj list
```

Or set in `~/.projectrc`:

```toml
debug = true
```

## 🔄 Updates

To update to the latest version:

```bash
# If using nix profile
nix profile upgrade

# If using flakes, update the lock file
nix flake update
```

## 📚 Usage Examples

```bash
# Create a new project
proj new awesome-tool

# Clone from GitHub
proj get microsoft/vscode

# List all projects
proj list

# Navigate with fuzzy search
p awesome        # Navigates to best match
p micro/vs<TAB>  # Tab completion
p -              # Go back to previous directory

# Search projects
proj query tool    # Find projects matching "tool"
```

## 🎯 Advanced Configuration

### Custom Shell Integration

If you want to customize the shell integration:

```bash
# Generate the shell script
project init zsh > ~/.config/project.zsh

# Edit the file as needed, then source it
source ~/.config/project.zsh
```

### Integration with Other Tools

Works great with:
- **direnv**: Automatic environment loading per project
- **tmux**: Session management per project
- **fzf**: Enhanced fuzzy finding (already integrated)

## 🤝 Contributing

See the main repository for development setup and contribution guidelines.
# Project

A Git-based project management tool that organizes your repositories in a GitHub-style directory structure (`username/project-name`) and provides fast project navigation similar to zoxide.

## Overview

This tool helps you manage local Git projects by:
- Organizing projects in a structured directory layout (`~/code/username/project-name`)
- Providing fast fuzzy search and navigation between projects
- Cloning repositories from GitHub with automatic directory structure
- Offering zsh shell integration with intelligent completion
- Listing and querying projects with various filters

## Installation

### Using Nix Flakes (Recommended)

Add to your `flake.nix`:

```nix
{
  inputs = {
    # Latest stable release (recommended)
    project.url = "github:gfanton/project?ref=latest";
    
    # Or specific version
    # project.url = "github:gfanton/project?ref=v1.2.3";
    
    # Or always latest from release branch
    # project.url = "github:gfanton/project/release";
  };

  outputs = { self, nixpkgs, project, ... }: {
    # Add to your packages
    environment.systemPackages = [ project.packages.${system}.default ];
  };
}
```

### Direct Installation with Nix

```bash
# From latest tag (recommended for stability)
nix profile install github:gfanton/project?ref=latest

# From release branch (always latest, but may be unstable)
nix profile install github:gfanton/project/release

# Into development shell
nix shell github:gfanton/project?ref=latest
```

### Build from source
```bash
git clone https://github.com/gfanton/project
cd project

# With Nix (recommended)
nix develop --impure  # --impure needed for development
make build

# Or traditional Go
make build
# Or install to $GOBIN
make install
```

## Usage

### Initialize shell integration
Add this to your `~/.zshrc`:
```bash
eval "$(proj init zsh)"
```

This enables the `p` command for quick project navigation.

### Commands

#### `proj new <name>`
Create a new project directory structure.
```bash
proj new myproject          # Creates ~/code/$USER/myproject
proj new username/myproject # Creates ~/code/username/myproject
```

#### `proj get <repo>`
Clone a repository from GitHub into the appropriate directory structure.
```bash
proj get username/repo      # Clones to ~/code/username/repo
proj get myrepo            # Clones to ~/code/$USER/myrepo (if default user set)
```

#### `proj list [--all]`
List all projects in your root directory.
```bash
proj list       # Shows only valid Git repositories
proj list --all # Shows all directories (including non-Git)
```

#### `proj query <search> [options]`
Search for projects using fuzzy matching.
```bash
proj query myproj                    # Find best match for "myproj"
proj query --limit 5 myproj          # Show up to 5 matches
proj query --exclude $(pwd) myproj   # Exclude current directory
proj query --abspath myproj          # Return absolute paths
```

#### `p <search>` (shell integration)
Navigate quickly to projects using fuzzy search.
```bash
p myproj          # Navigate to best matching project
p username/proj   # Navigate to specific user's project
p -               # Navigate to previous directory
```

## Configuration

### Config file
Create `~/.projectrc` (TOML format):
```toml
root = "~/code"           # Root directory for projects
user = "your-username"    # Default username for single-name projects
debug = false            # Enable debug logging
```

### Environment variables
- `PROJECT_ROOT`: Root directory (default: `~/code`)
- `PROJECT_USER`: Default username
- `PROJECT_CONFIG`: Config file path (default: `~/.projectrc`)
- `PROJECT_DEBUG`: Enable debug mode

### Command line flags
```bash
proj --root ~/my-projects --user myname --debug command
```

## Directory Structure

Projects are organized as:
```
~/code/
├── username1/
│   ├── project1/
│   ├── project2/
│   └── ...
├── username2/
│   ├── project1/
│   └── ...
└── ...
```

This mirrors GitHub's organization structure and makes it easy to find and manage projects.

## Shell Integration Features

- **Fast navigation**: Type `p projectname` to jump to any project
- **Enhanced completion**: Tab completion shows up to 20 matching projects with menu selection
- **Cycling support**: Use TAB to cycle through multiple completion options
- **Fuzzy search**: Finds projects even with partial/misspelled names
- **Visual menu**: Arrow keys to navigate completion menu when multiple matches exist
- **Exclude current**: Automatically excludes current directory from search results
- **Previous directory**: Use `p -` to return to previous location

## Dependencies

- Go 1.23+
- Git
- zsh (for shell integration)

## Development

### Build and Development
```bash
make build              # Build to ./build/proj
make install            # Install to $GOBIN
make test               # Run all tests
make test-coverage      # Run tests with coverage report
make test-coverage-html # Generate HTML coverage report
make bench              # Run performance benchmarks
make lint               # Run go vet and go fmt
make clean              # Remove build artifacts
make tidy               # Clean up dependencies
make dev                # Build and run
```

### Testing
The project has comprehensive test coverage (80-95%) across all packages:
- **Unit tests**: All core functionality tested
- **Integration tests**: Shell integration and interactive completion testing
- **Performance benchmarks**: Query operations tested with large datasets
- **Coverage reporting**: HTML and text coverage reports available

#### Integration Testing with Nix
For comprehensive shell integration testing, we use Nix to provide a reproducible test environment:

```bash
# Run integration tests locally (requires BATS and Expect)
make test-integration

# Run all tests in Nix environment (recommended)
make test-nix

# Enter Nix shell for interactive testing
make shell-nix
```

The integration tests verify:
- Enhanced zsh completion with menu selection (up to 20 options)
- Interactive shell navigation with the `p` command
- Completion cycling and arrow key navigation
- Fuzzy search behavior
- Directory exclusion in completions
- Shell initialization and function definitions

#### Test Dependencies
- **Nix**: For reproducible test environments
- **BATS**: Bash Automated Testing System for shell function tests
- **Expect**: For interactive shell testing
- **Go 1.23+**: For unit tests and benchmarks

## License

This project is open source. See LICENSE file for details.
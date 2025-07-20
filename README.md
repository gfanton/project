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

### Build from source
```bash
git clone https://github.com/gfanton/project
cd project
make build
# Or install to $GOBIN
make install
```

### Add to PATH
After building, add the binary to your PATH or use `make install` to install to `$GOBIN`.

## Usage

### Initialize shell integration
Add this to your `~/.zshrc`:
```bash
eval "$(project init zsh)"
```

This enables the `p` command for quick project navigation.

### Commands

#### `project new <name>`
Create a new project directory structure.
```bash
project new myproject          # Creates ~/code/$USER/myproject
project new username/myproject # Creates ~/code/username/myproject
```

#### `project get <repo>`
Clone a repository from GitHub into the appropriate directory structure.
```bash
project get username/repo      # Clones to ~/code/username/repo
project get myrepo            # Clones to ~/code/$USER/myrepo (if default user set)
```

#### `project list [--all]`
List all projects in your root directory.
```bash
project list       # Shows only valid Git repositories
project list --all # Shows all directories (including non-Git)
```

#### `project query <search> [options]`
Search for projects using fuzzy matching.
```bash
project query myproj                    # Find best match for "myproj"
project query --all myproj              # Show all matches ranked by relevance
project query --exclude $(pwd) myproj   # Exclude current directory
project query --abspath myproj          # Return absolute paths
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
project --root ~/my-projects --user myname --debug command
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
- **Intelligent completion**: Tab completion shows matching projects
- **Fuzzy search**: Finds projects even with partial/misspelled names
- **Exclude current**: Automatically excludes current directory from search results
- **Previous directory**: Use `p -` to return to previous location

## Dependencies

- Go 1.23+
- Git
- zsh (for shell integration)

## Development

### Build and Development
```bash
make build              # Build to ./build/project
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
- **Integration tests**: Real Git repository operations
- **Performance benchmarks**: Query operations tested with large datasets
- **Coverage reporting**: HTML and text coverage reports available

## License

This project is open source. See LICENSE file for details.
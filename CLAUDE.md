# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Documentation Maintenance

**IMPORTANT**: At the start of each session, read all files in the `docs/` folder to understand current documentation state.

When making changes to this codebase:
1. **Keep CLAUDE.md up to date** - Update project structure, commands, or architecture sections when they change
2. **Keep docs/ up to date** - Update relevant documentation when features change
3. **Add new docs** - Create new documentation files in `docs/` for significant new features

Current documentation:
- `docs/tmux-usage.md` - Tmux plugin installation and usage guide

## Project Overview

A Git-based project management tool that organizes repositories in a GitHub-style directory structure (`~/code/username/project-name`) and provides fast project navigation similar to zoxide. The project includes both a main CLI tool (`proj`) and a tmux plugin (`proj-tmux`) for seamless integration with tmux sessions.

## Build and Development Commands

### Essential Commands
```bash
# Build main application
make build                # Builds to ./build/proj

# Build tmux integration binary  
make build-tmux           # Builds to ./build/proj-tmux

# Build everything
make build-all            # Builds both proj and proj-tmux

# Run tests
make test                 # Run all Go unit tests (80-95% coverage)
make test-tmux            # Run BATS unit tests for tmux integration
make test-nix             # Run all tests in Nix environment

# Development workflow
make dev                  # Build and run proj
make lint                 # Run go vet and go fmt
make tidy                 # Clean up Go dependencies
make clean                # Remove build artifacts

# Release
./scripts/release/release.sh          # Interactive release (prompts for version)
./scripts/release/release.sh v1.2.3   # Direct release with specified version
make update-vendor-hash               # Update vendorHash in flake.nix when deps change
```

### Testing a Single Test
```bash
go test -v -run TestFunctionName ./internal/package/
go test -v -run TestFunctionName/SubTest ./internal/package/
```

## Project Structure

```
.
├── cmd/proj/                   # Main CLI application
│   ├── main.go                 # Entry point, command registration
│   ├── clone.go                # proj clone command
│   ├── query.go                # proj query command
│   ├── workspace.go            # proj workspace command
│   └── *_test.go               # Command tests
│
├── internal/                   # Private packages (not importable)
│   ├── config/                 # Configuration & logging
│   ├── project/                # Project parsing & walking
│   ├── query/                  # Fuzzy search service
│   ├── git/                    # Git operations (clone, auth)
│   ├── workspace/              # Git worktree management
│   └── shell/                  # Shell detection & integration
│
├── pkg/                        # Public packages
│   └── template/               # Shell init templates (zsh, bash, fish)
│
├── plugins/proj-tmux/          # Tmux integration binary
│   ├── main.go                 # Entry point
│   ├── session.go              # Session management
│   ├── window.go               # Window management
│   ├── tmux.go                 # Tmux service wrapper
│   └── plugin/                 # Tmux plugin files
│       ├── proj-tmux.tmux      # Plugin entry script
│       └── scripts/            # fzf popup scripts
│
├── projects_*.go               # Public API (delegates to internal/)
│
├── docs/                       # User documentation
│   └── tmux-usage.md           # Tmux plugin guide
│
├── flake.nix                   # Nix build & dev environment
├── .goreleaser.yml             # Release configuration
└── .github/workflows/          # CI/CD pipelines
    ├── test.yml                # Test on push/PR
    └── release.yml             # Build releases on tags
```

## Architecture and Code Structure

### Core Architecture
The project follows Clean Architecture with clear separation of concerns:

- **Commands** (`cmd/proj/*.go`): Each command is self-contained with embedded config structs using ffcli framework
- **Internal packages** (`internal/`): Private business logic, not exposed to consumers
  - `config`: Global configuration management with slog integration
  - `project`: Core project parsing and directory walking logic  
  - `git`: Git operations with SSH/HTTPS authentication support
  - `query`: Fuzzy search service with ranking
  - `workspace`: Workspace management for multi-branch workflows
- **Public packages** (`pkg/`): Template system for shell integration
- **Plugins** (`plugins/proj-tmux/`): Tmux integration with session/window management

### Key Design Patterns
1. **Dependency Injection**: Services (logger, config) passed through context
2. **Embedded Templates**: Shell templates embedded in binary using `//go:embed`
3. **Structured Logging**: Using standard library slog (replaced zap in Go 1.23 upgrade)
4. **Error Wrapping**: All errors wrapped with context using `fmt.Errorf`
5. **Configuration Cascade**: CLI flags > Environment vars > Config file > Defaults

### Package Dependencies

```
┌─────────────────────────────────────────────────────────────┐
│                      cmd/proj                               │
│                   (CLI commands)                            │
└──────────────┬──────────────────────────────────────────────┘
               │ imports
               ▼
┌─────────────────────────────────────────────────────────────┐
│                   projects (root)                           │
│              Public API: ProjectService,                    │
│              QueryService, WorkspaceService                 │
└──────────────┬──────────────────────────────────────────────┘
               │ delegates to
               ▼
┌─────────────────────────────────────────────────────────────┐
│                      internal/                              │
├─────────────┬─────────────┬─────────────┬──────────────────┤
│   config    │   project   │    query    │    workspace     │
│  (logging)  │  (parsing)  │  (search)   │   (worktrees)    │
└─────────────┴─────────────┴─────────────┴──────────────────┘
               │
               ▼
┌─────────────────────────────────────────────────────────────┐
│                    internal/git                             │
│                 (clone, auth)                               │
└─────────────────────────────────────────────────────────────┘
```

The public `projects` package provides a stable API that delegates to internal packages, allowing internal refactoring without breaking consumers.

### Project/Workspace Syntax
- Projects: `username/projectname` or `projectname` (uses default user)
- Workspaces: `project:workspace` for branch-specific windows in tmux
- Directory structure: `~/code/username/projectname/`

### Critical Files and Functions

**Main Entry Points:**
- `cmd/proj/main.go`: CLI entry with ffcli framework
- `plugins/proj-tmux/main.go`: Tmux integration binary

**Core Logic:**
- `internal/project/project.go:31-59`: ParseProject() - parses project names
- `internal/project/project.go:139-158`: Walk() - walks project directories
- `internal/query/query.go:53-104`: Search() - fuzzy search implementation
- `internal/workspace/workspace.go`: Workspace parsing and management

**Shell Integration:**
- `pkg/template/zsh.init`: Zsh completion template (embedded)
- `pkg/template/template.go`: Template rendering logic

**Tmux Plugin:**
- `plugins/proj-tmux/plugin/proj-tmux.tmux`: Main plugin entry point
- `plugins/proj-tmux/plugin/scripts/`: Shell scripts for tmux UI

## Dependencies and Versions

- **Go**: 1.23.0+ (using Go 1.23 toolchain features)
- **Main dependencies**:
  - `github.com/peterbourgon/ff/v3`: CLI framework
  - `github.com/go-git/go-git/v5`: Git operations  
  - `github.com/lithammer/fuzzysearch`: Fuzzy search
- **Nix**: Used for reproducible builds and testing environment
- **vendorHash**: Must be updated when Go dependencies change

## CI/CD and Release Process

### GitHub Actions
- **Test workflow** (`.github/workflows/test.yml`): Runs on push/PR, tests on Ubuntu/macOS
- **Release workflow** (`.github/workflows/release.yml`): Triggered on version tags, builds with GoReleaser

### Release Flow

```
┌─────────────────────────────────────────────────────────────────┐
│  LOCAL: ./scripts/release/release.sh v0.17.0                    │
├─────────────────────────────────────────────────────────────────┤
│  1. Calculate & update vendorHash in flake.nix                  │
│  2. Update projectVersion = "0.17.0" in flake.nix               │
│  3. Test nix build                                              │
│  4. Create commit: "chore: release v0.17.0"                     │
│  5. Create annotated tag: v0.17.0                               │
└─────────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────────┐
│  LOCAL: git push <remote> master && git push <remote> v0.17.0   │
└─────────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────────┐
│  GITHUB ACTIONS: .github/workflows/release.yml                  │
├─────────────────────────────────────────────────────────────────┤
│  1. Verify tag matches flake.nix projectVersion                 │
│  2. Verify nix build succeeds                                   │
│  3. Run test suite                                              │
│  4. GoReleaser builds:                                          │
│     • proj (linux, darwin, windows) × (amd64, arm64)            │
│     • proj-tmux (linux, darwin) × (amd64, arm64)                │
│  5. Create GitHub release with:                                 │
│     • tar.gz/zip archives                                       │
│     • deb/rpm packages                                          │
│     • checksums.txt                                             │
└─────────────────────────────────────────────────────────────────┘
```

### Release Commands
```bash
./scripts/release/release.sh          # Interactive (prompts for version)
./scripts/release/release.sh v1.2.3   # Direct release with specified version
```

### Version Management
- **Source of truth**: `flake.nix` → `projectVersion` variable
- **Packages sharing version**: proj, proj-tmux, tmux-proj (Nix)
- **Injection**: ldflags `-X main.version=${version}` at build time
- **Verification**: Release workflow validates tag matches flake.nix

## Tmux Plugin Development

The project includes a comprehensive tmux plugin at `plugins/proj-tmux/`:

### Plugin Structure
- **Binary**: `proj-tmux` provides session/window management commands
- **Scripts**: Shell scripts in `plugin/scripts/` handle tmux UI interactions
- **Key bindings**: Ctrl+P (session picker), Ctrl+W (window picker)

### Testing Tmux Integration
```bash
make build-tmux           # Build proj-tmux binary
make test-tmux           # Run BATS tests
make test-plugin         # Test plugin structure
nix develop .#testing    # Enter testing environment with tmux
```

### Key Features
- Smart context detection (auto-fills current project)
- Unified fzf interface for session and window creation
- Workspace support (`project:workspace` syntax)
- Tab completion in popups without exiting

## Recent Important Changes

### 2025-08-02: Tmux Plugin Improvements
- Fixed Tab completion in fzf popups (completes without exiting)
- Unified Ctrl+P and Ctrl+W interfaces with identical configurations
- Smart context detection for Ctrl+W (pre-fills current project with ':')
- Always uses fzf popup when available

### 2025-07-20: Major Refactor
- Upgraded from Go 1.19 to Go 1.23
- Replaced zap with standard library slog
- Restructured to cmd/internal/pkg architecture
- Added 80-95% test coverage across packages
- Enhanced shell completion (up to 20 options with cycling)

## Git Commit Guidelines

**IMPORTANT**: Do NOT add "Co-Authored-By: Claude" or any Claude/AI attribution to commit messages. Commits should appear as normal developer commits without any AI tooling references.

## Working with This Codebase

1. **Read docs first**: At session start, read `docs/` folder to understand current state
2. **Before making changes**: Check existing patterns in similar files
3. **Use existing libraries**: Check `go.mod` before adding new dependencies
4. **Follow conventions**: Maintain consistent error handling, logging patterns
5. **Test coverage**: Aim for 80%+ coverage, run `make test-coverage`
6. **Lint before commit**: Always run `make lint`
7. **Update vendorHash**: Run `make update-vendor-hash` after changing dependencies
8. **Update documentation**: Keep CLAUDE.md and `docs/` in sync with code changes
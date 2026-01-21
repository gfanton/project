package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/gfanton/projects/internal/config"
	"github.com/gfanton/projects/internal/project"
	"github.com/peterbourgon/ff/v4"
)

func newAddCommand(logger *slog.Logger, cfg *config.Config) *ff.Command {
	return &ff.Command{
		Name:      "add",
		Usage:     "proj add [name]",
		ShortHelp: "Add current directory as a project symlink",
		LongHelp: `Add the current directory as a project symlink in the configured root.

If no name is provided, uses the current directory name with the default user.
If name is provided, it can be:
  - "project" (uses default user from config)
  - "user/project" (explicit user specification)

This creates a symlink from the project directory structure to the current directory.

Example:
  cd /path/to/my-existing-project
  proj add                    # Creates ~/code/defaultuser/my-existing-project -> /path/to/my-existing-project
  proj add myapp              # Creates ~/code/defaultuser/myapp -> /path/to/my-existing-project
  proj add johndoe/webapp     # Creates ~/code/johndoe/webapp -> /path/to/my-existing-project`,
		Exec: func(ctx context.Context, args []string) error {
			return runAdd(ctx, logger, cfg, args)
		},
	}
}

func runAdd(ctx context.Context, logger *slog.Logger, cfg *config.Config, args []string) error {
	currentDir, err := getCurrentDir()
	if err != nil {
		return err
	}

	var projectName string
	if len(args) == 0 {
		// Use current directory name as project name
		projectName = filepath.Base(currentDir)
	} else if len(args) == 1 {
		projectName = args[0]
	} else {
		return fmt.Errorf("too many arguments, expected 0 or 1 project name")
	}

	p, err := project.ParseProject(cfg.RootDir, cfg.RootUser, projectName)
	if err != nil {
		return fmt.Errorf("failed to parse project name: %w", err)
	}

	// Check if current directory is already inside the project root
	relPath, err := filepath.Rel(cfg.RootDir, currentDir)
	if err == nil && !filepath.IsAbs(relPath) && !strings.HasPrefix(relPath, "..") {
		return fmt.Errorf("current directory is already inside project root: %s", cfg.RootDir)
	}

	// Check if symlink target already exists
	if _, err := os.Lstat(p.Path); err == nil {
		return fmt.Errorf("project already exists: %s", p.Path)
	}

	// Create parent directory if it doesn't exist
	parentDir := filepath.Dir(p.Path)
	if err := os.MkdirAll(parentDir, 0755); err != nil {
		return fmt.Errorf("failed to create parent directory: %w", err)
	}

	// Create symlink
	if err := os.Symlink(currentDir, p.Path); err != nil {
		return fmt.Errorf("failed to create symlink: %w", err)
	}

	logger.Info("added project symlink",
		"name", p.String(),
		"link", p.Path,
		"target", currentDir)

	fmt.Printf("Added project: %s\n", p.String())
	fmt.Printf("Symlink: %s -> %s\n", p.Path, currentDir)

	return nil
}

// getCurrentDir returns the current working directory, preserving symlinks if possible.
// It prefers the PWD environment variable over os.Getwd() to preserve symlink paths.
func getCurrentDir() (string, error) {
	// Try PWD first to preserve symlinks
	if currentDir := os.Getenv("PWD"); currentDir != "" {
		currentDir = filepath.Clean(currentDir)

		// Verify PWD points to a valid directory
		if _, err := os.Stat(currentDir); err == nil {
			return currentDir, nil
		}
		// PWD is stale, fall through to os.Getwd()
	}

	// Fallback to os.Getwd()
	currentDir, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("failed to get current directory: %w", err)
	}

	currentDir, err = filepath.Abs(currentDir)
	if err != nil {
		return "", fmt.Errorf("failed to get absolute path: %w", err)
	}

	return currentDir, nil
}

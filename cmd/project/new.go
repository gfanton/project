package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"

	"github.com/gfanton/project/internal/config"
	"github.com/gfanton/project/internal/project"
	"github.com/peterbourgon/ff/v3/ffcli"
)

func newNewCommand(logger *slog.Logger, cfg *config.Config) *ffcli.Command {
	return &ffcli.Command{
		Name:       "new",
		ShortUsage: "project new <name>",
		ShortHelp:  "Create a new project directory",
		LongHelp: `Create a new project directory in the configured root.

The project name can be:
  - "project" (uses default user from config)
  - "user/project" (explicit user specification)

Example:
  project new myapp
  project new johndoe/webapp`,
		Exec: func(ctx context.Context, args []string) error {
			return runNew(ctx, logger, cfg, args)
		},
	}
}

func runNew(ctx context.Context, logger *slog.Logger, cfg *config.Config, args []string) error {
	if len(args) != 1 {
		return fmt.Errorf("exactly one project name required")
	}

	p, err := project.ParseProject(cfg.RootDir, cfg.RootUser, args[0])
	if err != nil {
		return fmt.Errorf("failed to parse project name: %w", err)
	}

	// Check if directory already exists
	if _, err := os.Stat(p.Path); err == nil {
		return fmt.Errorf("project directory already exists: %s", p.Path)
	}

	// Create the directory
	if err := os.MkdirAll(p.Path, 0755); err != nil {
		return fmt.Errorf("failed to create project directory: %w", err)
	}

	logger.Info("created new project", "name", p.String(), "path", p.Path)
	fmt.Printf("Created project: %s\n", p.String())
	fmt.Printf("Location: %s\n", p.Path)

	return nil
}

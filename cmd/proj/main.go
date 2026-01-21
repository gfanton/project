package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"

	"github.com/gfanton/projects"
	"github.com/gfanton/projects/internal/config"
	"github.com/peterbourgon/ff/v4"
	"github.com/peterbourgon/ff/v4/ffhelp"
)

type rootConfig struct {
	config *config.Config
	logger *slog.Logger
}

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	cfg, err := config.NewConfig()
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: failed to create config: %v\n", err)
		os.Exit(1)
	}

	if err := cfg.Load(os.Args[1:]); err != nil {
		fmt.Fprintf(os.Stderr, "error: failed to load config: %v\n", err)
		os.Exit(1)
	}

	logger := cfg.Logger()

	// Create projects config and services
	projectsCfg := &projects.Config{
		ConfigFile: cfg.ConfigFile,
		Debug:      cfg.Debug,
		RootDir:    cfg.RootDir,
		RootUser:   cfg.RootUser,
	}
	projectsLogger := projects.NewSlogAdapter(logger)

	rootCfg := &rootConfig{
		config: cfg,
		logger: logger,
	}

	// Create root flag set with global flags
	// These flags are parsed by config.Load() first, but need to be defined
	// here so the command parser accepts them without error
	rootFlags := ff.NewFlagSet("proj")
	rootFlags.BoolVar(&cfg.Debug, 0, "debug", "enable debug logging")
	rootFlags.StringVar(&cfg.RootDir, 0, "root", cfg.RootDir, "root directory for projects")
	rootFlags.StringVar(&cfg.RootUser, 0, "user", cfg.RootUser, "default user for projects")
	rootFlags.StringVar(&cfg.ConfigFile, 0, "config", cfg.ConfigFile, "configuration file path")

	root := &ff.Command{
		Name:      "proj",
		Usage:     "proj [flags] <subcommand>",
		ShortHelp: "A tool for managing Git projects in GitHub-style directory structure",
		LongHelp: `proj is a command-line tool for managing Git projects organized in a GitHub-style
directory structure. It provides fast project navigation, creation, and management.

Use 'proj <subcommand> -h' for more information about a specific command.`,
		Flags: rootFlags,
		Exec: func(ctx context.Context, args []string) error {
			return ff.ErrHelp
		},
		Subcommands: []*ff.Command{
			newInitCommand(logger, cfg),
			newListCommand(logger, cfg, projectsCfg, projectsLogger),
			newNewCommand(logger, cfg),
			newAddCommand(logger, cfg),
			newGetCommand(logger, cfg),
			newQueryCommand(logger, cfg, projectsCfg, projectsLogger),
			newWorkspaceCommand(logger, cfg, projectsCfg, projectsLogger),
			NewVersionCommand(rootCfg),
		},
	}

	if err := root.ParseAndRun(ctx, os.Args[1:]); err != nil {
		if errors.Is(err, ff.ErrHelp) {
			fmt.Fprint(os.Stdout, ffhelp.Command(root))
			os.Exit(0)
		}
		logger.Error("command failed", "error", err)
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

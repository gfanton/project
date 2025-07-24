package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"

	"github.com/peterbourgon/ff/v3/ffcli"
	"projects"
	"projects/internal/config"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Load configuration using existing config system
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

	root := &ffcli.Command{
		Name:       "proj-tmux",
		ShortUsage: "proj-tmux [flags] <subcommand>",
		ShortHelp:  "Tmux integration for proj - session and workspace management",
		LongHelp: `proj-tmux provides tmux session and workspace management for proj.

This binary is designed to be called from tmux key bindings and plugins.
It integrates with the project management system to provide seamless
tmux session and workspace operations.

Use 'proj-tmux <subcommand> -h' for more information about a specific command.`,
		FlagSet: flag.NewFlagSet("proj-tmux", flag.ContinueOnError),
		Exec: func(ctx context.Context, args []string) error {
			return flag.ErrHelp
		},
		Subcommands: []*ffcli.Command{
			newSessionCommand(logger, projectsCfg, projectsLogger),
			newWindowCommand(logger, projectsCfg, projectsLogger),
			newSwitchCommand(logger, projectsCfg, projectsLogger),
			newStatusCommand(logger, projectsCfg, projectsLogger),
		},
	}

	if err := root.ParseAndRun(ctx, os.Args[1:]); err != nil {
		if !errors.Is(err, flag.ErrHelp) {
			logger.Error("command failed", "error", err)
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
		}
		os.Exit(1)
	}
}

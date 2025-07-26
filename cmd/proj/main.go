package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"

	"github.com/peterbourgon/ff/v3/ffcli"
	"github.com/gfanton/projects"
	"github.com/gfanton/projects/internal/config"
)

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

	root := &ffcli.Command{
		Name:       "proj",
		ShortUsage: "proj [flags] <subcommand>",
		ShortHelp:  "A tool for managing Git projects in GitHub-style directory structure",
		LongHelp: `proj is a command-line tool for managing Git projects organized in a GitHub-style
directory structure. It provides fast project navigation, creation, and management.

Use 'proj <subcommand> -h' for more information about a specific command.`,
		FlagSet: flag.NewFlagSet("proj", flag.ContinueOnError),
		Exec: func(ctx context.Context, args []string) error {
			return flag.ErrHelp
		},
		Subcommands: []*ffcli.Command{
			newInitCommand(logger, cfg),
			newListCommand(logger, cfg, projectsCfg, projectsLogger),
			newNewCommand(logger, cfg),
			newGetCommand(logger, cfg),
			newQueryCommand(logger, cfg, projectsCfg, projectsLogger),
			newWorkspaceCommand(logger, cfg, projectsCfg, projectsLogger),
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

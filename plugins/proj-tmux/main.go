package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"runtime"

	"github.com/gfanton/projects"
	"github.com/gfanton/projects/internal/config"
	"github.com/peterbourgon/ff/v4"
	"github.com/peterbourgon/ff/v4/ffhelp"
)

// ---- Version Variables (injected at build time)
var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
	builtBy = "unknown"
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

	// Create root flag set with global flags
	rootFlags := ff.NewFlagSet("proj-tmux")
	rootFlags.BoolVar(&cfg.Debug, 0, "debug", "enable debug logging")
	rootFlags.StringVar(&cfg.RootDir, 0, "root", cfg.RootDir, "root directory for projects")
	rootFlags.StringVar(&cfg.RootUser, 0, "user", cfg.RootUser, "default user for projects")
	rootFlags.StringVar(&cfg.ConfigFile, 0, "config", cfg.ConfigFile, "configuration file path")

	root := &ff.Command{
		Name:      "proj-tmux",
		Usage:     "proj-tmux [flags] <subcommand>",
		ShortHelp: "Tmux integration for proj - session and workspace management",
		LongHelp: `proj-tmux provides tmux session and workspace management for proj.

This binary is designed to be called from tmux key bindings and plugins.
It integrates with the project management system to provide seamless
tmux session and workspace operations.

Use 'proj-tmux <subcommand> -h' for more information about a specific command.`,
		Flags: rootFlags,
		Exec: func(ctx context.Context, args []string) error {
			return ff.ErrHelp
		},
		Subcommands: []*ff.Command{
			newSessionCommand(logger, projectsCfg, projectsLogger),
			newWindowCommand(logger, projectsCfg, projectsLogger),
			newSwitchCommand(logger, projectsCfg, projectsLogger),
			newStatusCommand(logger, projectsCfg, projectsLogger),
			newVersionCommand(),
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

func newVersionCommand() *ff.Command {
	var verbose bool
	fs := ff.NewFlagSet("proj-tmux version")
	fs.BoolVar(&verbose, 'v', "verbose", "show verbose version information")

	return &ff.Command{
		Name:      "version",
		Usage:     "proj-tmux version [-v]",
		ShortHelp: "Show version information",
		Flags:     fs,
		Exec: func(ctx context.Context, args []string) error {
			if verbose {
				fmt.Printf("proj-tmux version %s\n", version)
				fmt.Printf("  commit: %s\n", commit)
				fmt.Printf("  built at: %s\n", date)
				fmt.Printf("  built by: %s\n", builtBy)
				fmt.Printf("  go version: %s\n", runtime.Version())
				fmt.Printf("  platform: %s/%s\n", runtime.GOOS, runtime.GOARCH)
			} else {
				fmt.Println(version)
			}
			return nil
		},
	}
}

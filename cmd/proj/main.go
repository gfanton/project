package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"

	"github.com/gfanton/project/internal/config"
	"github.com/peterbourgon/ff/v3/ffcli"
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

	root := &ffcli.Command{
		Name:    "project [flags] <subcommand>",
		FlagSet: flag.NewFlagSet("project", flag.ExitOnError),
		Exec: func(ctx context.Context, args []string) error {
			return flag.ErrHelp
		},
		Subcommands: []*ffcli.Command{
			newInitCommand(logger, cfg),
			newListCommand(logger, cfg),
			newNewCommand(logger, cfg),
			newGetCommand(logger, cfg),
			newQueryCommand(logger, cfg),
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

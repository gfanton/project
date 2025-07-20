package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"strings"

	"github.com/peterbourgon/ff/v3/ffcli"
)

type NewConfig struct {
	*RootConfig

	Capital bool
}

func ProjectNew(ctx context.Context, logger *log.Logger, rcfg *NewConfig, args ...string) error {
	fmt.Println("new folder", strings.Join(args, " "))
	return nil
}

func newCommand(logger *log.Logger, rcfg *RootConfig) *ffcli.Command {
	var cfg NewConfig
	cfg.RootConfig = rcfg

	flagSet := flag.NewFlagSet("new", flag.ExitOnError)
	flagSet.BoolVar(&cfg.Capital, "c", false, "display in capital")

	return &ffcli.Command{
		Name:        "new",
		ShortUsage:  "project new <name>",
		ShortHelp:   "new project",
		FlagSet:     flagSet,
		Subcommands: []*ffcli.Command{},
		Exec: func(ctx context.Context, args []string) error {
			return ProjectNew(ctx, logger, &cfg, args...)
		},
	}
}

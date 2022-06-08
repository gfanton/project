package main

import (
	"context"
	"flag"
	"fmt"
	"strings"

	"github.com/peterbourgon/ff/v3/ffcli"
)

type NewConfig struct {
	*RootConfig

	Capital bool
}

func ProjectsNew(ctx context.Context, rcfg *NewConfig, args ...string) error {
	fmt.Println("new folder", strings.Join(args, " "))
	return nil
}

func newCommand(rcfg *RootConfig) *ffcli.Command {
	var cfg NewConfig
	cfg.RootConfig = rcfg

	flagSet := flag.NewFlagSet("new", flag.ExitOnError)
	flagSet.BoolVar(&cfg.Capital, "c", false, "display in capital")

	return &ffcli.Command{
		Name:        "new",
		ShortUsage:  "projects new <name>",
		ShortHelp:   "new projects",
		FlagSet:     flagSet,
		Subcommands: []*ffcli.Command{},
		Exec: func(ctx context.Context, args []string) error {
			return ProjectsNew(ctx, &cfg, args...)
		},
	}
}

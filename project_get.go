package main

import (
	"context"
	"flag"
	"fmt"
	"log"

	"github.com/peterbourgon/ff/v3/ffcli"
)

type GetConfig struct {
	*RootConfig

	Capital bool
}

func ProjectGet(ctx context.Context, logger *log.Logger, rcfg *GetConfig, args ...string) error {
	ps := make([]*Project, len(args))
	for i, name := range args {
		var err error
		if ps[i], err = ParseProject(rcfg.RootConfig, name); err != nil {
			return fmt.Errorf("unable to parse: `%s`", err.Error())
		}
	}

	for _, p := range ps {
		repo_path := fmt.Sprintf("%s/%s", p.Organisation, p.Name)
		err := Git().CloneContext(ctx, DefaultProvider, repo_path, p.Path)
		if err != nil {
			fmt.Printf("unable to clone: %s/%s: %s\n",
				p.Organisation, p.Name, err.Error())
		}
	}

	return nil
}

func getCommand(logger *log.Logger, rcfg *RootConfig) *ffcli.Command {
	var cfg GetConfig
	cfg.RootConfig = rcfg

	flagSet := flag.NewFlagSet("get", flag.ExitOnError)

	return &ffcli.Command{
		Name:        "get",
		ShortUsage:  "project get <name>",
		ShortHelp:   "get project",
		FlagSet:     flagSet,
		Subcommands: []*ffcli.Command{},
		Exec: func(ctx context.Context, args []string) error {
			return ProjectGet(ctx, logger, &cfg, args...)
		},
	}
}

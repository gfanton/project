package main

import (
	"context"
	"flag"
	"fmt"
	"io/fs"

	"github.com/go-git/go-git/v5"
	"github.com/peterbourgon/ff/v3/ffcli"
)

type ListConfig struct {
	*RootConfig

	ListAll bool
}

type ListStatus string

const (
	ListStatus_NotAGit    ListStatus = "not a git"
	ListStatus_Git        ListStatus = "valid"
	ListStatus_InvalidGit ListStatus = "invalid"
)

func ProjectsList(ctx context.Context, cfg *ListConfig) error {
	return WalkProject(cfg.RootDir, func(d fs.DirEntry, p *Project) error {
		var status ListStatus
		_, err := p.OpenRepo()
		switch err {
		case git.ErrRepositoryNotExists:
			status = ListStatus_NotAGit
			return nil
		case nil:
			status = ListStatus_Git
		default:
			status = ListStatus_InvalidGit
		}

		fmt.Printf("%s/%s - [%s]\n", p.Organisation, p.Name, status)
		return nil
	})
}

func listCommand(rcfg *RootConfig) *ffcli.Command {
	var cfg ListConfig
	cfg.RootConfig = rcfg

	flagSet := flag.NewFlagSet("list", flag.ExitOnError)
	flagSet.BoolVar(&cfg.ListAll, "all", false, "display all project (valid/invalid)")

	return &ffcli.Command{
		Name:        "list",
		ShortUsage:  "projects list",
		ShortHelp:   "list projects",
		FlagSet:     flagSet,
		Subcommands: []*ffcli.Command{},
		Exec: func(ctx context.Context, args []string) error {
			return ProjectsList(ctx, &cfg)
		},
	}
}

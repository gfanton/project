package main

import (
	"context"
	"flag"
	"fmt"
	"io/fs"
	"log/slog"

	"github.com/gfanton/project/internal/config"
	"github.com/gfanton/project/internal/project"
	"github.com/peterbourgon/ff/v3/ffcli"
)

type listConfig struct {
	all bool
}

func newListCommand(logger *slog.Logger, cfg *config.Config) *ffcli.Command {
	var listCfg listConfig

	fs := flag.NewFlagSet("list", flag.ExitOnError)
	fs.BoolVar(&listCfg.all, "all", false, "display all projects (including non-Git directories)")

	return &ffcli.Command{
		Name:       "list",
		ShortUsage: "project list [flags]",
		ShortHelp:  "List all projects",
		LongHelp: `List all projects in the configured root directory.

By default, only Git repositories are shown. Use --all to show all directories.`,
		FlagSet: fs,
		Exec: func(ctx context.Context, args []string) error {
			return runList(ctx, logger, cfg, listCfg)
		},
	}
}

func runList(ctx context.Context, logger *slog.Logger, cfg *config.Config, listCfg listConfig) error {
	return project.Walk(cfg.RootDir, func(d fs.DirEntry, p *project.Project) error {
		status := p.GetGitStatus()

		// Skip non-Git directories unless --all is specified
		if status == project.GitStatusNotGit && !listCfg.all {
			return nil
		}

		fmt.Printf("%s - [%s]\n", p.String(), status)
		return nil
	})
}

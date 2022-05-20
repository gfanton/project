package main

import (
	"context"
	"flag"
	"fmt"
	"os"

	"github.com/go-git/go-git/v5"
	"github.com/peterbourgon/ff/v3/ffcli"
)

type GetConfig struct {
	*RootConfig

	Capital bool
}

func ProjectsGet(ctx context.Context, rcfg *GetConfig, args ...string) error {
	ps := make([]*Project, len(args))
	for i, name := range args {
		var err error
		if ps[i], err = ParseProject(rcfg.RootConfig, name); err != nil {
			return fmt.Errorf("unable to parse: `%s`", err.Error())
		}
	}

	// agent, err := ssh.NewSSHAgentAuth("")
	// if err != nil {
	// 	return fmt.Errorf("unable to get ssh agent: %w", err)
	// }

	for _, p := range ps {
		fmt.Printf("git clone %s\n", p.GitHTTPUrl())
		_, err := git.PlainCloneContext(ctx, p.Path, false, &git.CloneOptions{
			URL:      p.GitHTTPUrl(),
			Progress: os.Stdout,
			// Auth:     agent,
		})
		if err != nil {
			fmt.Printf("unable to clone: %s/%s: %s\n",
				p.Organisation, p.Name, err.Error())
		}
	}

	return nil
}

func getCommand(rcfg *RootConfig) *ffcli.Command {
	var cfg GetConfig
	cfg.RootConfig = rcfg

	flagSet := flag.NewFlagSet("get", flag.ExitOnError)

	return &ffcli.Command{
		Name:        "get",
		ShortUsage:  "projects get <name>",
		ShortHelp:   "get projects",
		FlagSet:     flagSet,
		Subcommands: []*ffcli.Command{},
		Exec: func(ctx context.Context, args []string) error {
			return ProjectsGet(ctx, &cfg, args...)
		},
	}
}

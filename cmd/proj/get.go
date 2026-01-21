package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"

	"github.com/gfanton/projects/internal/config"
	"github.com/gfanton/projects/internal/git"
	"github.com/gfanton/projects/internal/project"
	"github.com/peterbourgon/ff/v4"
)

type getConfig struct {
	UseSSH bool
	Token  string
}

func newGetCommand(logger *slog.Logger, cfg *config.Config) *ff.Command {
	getCfg := &getConfig{}
	fs := ff.NewFlagSet("get")
	fs.BoolVar(&getCfg.UseSSH, 0, "ssh", "use SSH for cloning instead of HTTPS")
	fs.StringVar(&getCfg.Token, 0, "token", os.Getenv("GITHUB_TOKEN"), "GitHub token for authentication")

	return &ff.Command{
		Name:      "get",
		Usage:     "proj get [flags] <name>...",
		ShortHelp: "Clone projects from GitHub",
		LongHelp: `Clone one or more projects from GitHub into the configured directory structure.

The project name can be:
  - "project" (uses default user from config)
  - "user/project" (explicit user specification)

Examples:
  proj get myrepo
  proj get johndoe/webapp
  proj get --ssh johndoe/webapp
  proj get repo1 user2/repo2`,
		Flags: fs,
		Exec: func(ctx context.Context, args []string) error {
			return runGet(ctx, logger, cfg, *getCfg, args)
		},
	}
}

func runGet(ctx context.Context, logger *slog.Logger, cfg *config.Config, getCfg getConfig, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("at least one project name required")
	}

	gitClient := git.NewClient(logger)

	for _, arg := range args {
		p, err := project.ParseProject(cfg.RootDir, cfg.RootUser, arg)
		if err != nil {
			logger.Error("failed to parse project name", "name", arg, "error", err)
			fmt.Printf("Error: failed to parse project name '%s': %v\n", arg, err)
			continue
		}

		// Check if directory already exists
		if _, err := os.Stat(p.Path); err == nil {
			logger.Warn("project directory already exists", "name", p.String(), "path", p.Path)
			fmt.Printf("Warning: project directory already exists: %s\n", p.Path)
			continue
		}

		// Determine URL to use
		url := p.GitHTTPURL()
		if getCfg.UseSSH {
			url = p.GitSSHURL()
		}

		cloneOpts := git.CloneOptions{
			URL:         url,
			Destination: p.Path,
			UseSSH:      getCfg.UseSSH,
			Token:       getCfg.Token,
		}

		if err := gitClient.Clone(ctx, cloneOpts); err != nil {
			logger.Error("failed to clone project", "name", p.String(), "url", url, "error", err)
			fmt.Printf("Error: failed to clone %s: %v\n", p.String(), err)
			continue
		}

		fmt.Printf("Cloned: %s\n", p.String())
	}

	return nil
}

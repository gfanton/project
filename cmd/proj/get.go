package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"os"

	"github.com/gfanton/project/internal/config"
	"github.com/gfanton/project/internal/git"
	"github.com/gfanton/project/internal/project"
	"github.com/peterbourgon/ff/v3/ffcli"
)

type getConfig struct {
	useSSH bool
	token  string
}

func newGetCommand(logger *slog.Logger, cfg *config.Config) *ffcli.Command {
	var getCfg getConfig

	fs := flag.NewFlagSet("get", flag.ExitOnError)
	fs.BoolVar(&getCfg.useSSH, "ssh", false, "use SSH for cloning instead of HTTPS")
	fs.StringVar(&getCfg.token, "token", os.Getenv("GITHUB_TOKEN"), "GitHub token for authentication")

	return &ffcli.Command{
		Name:       "get",
		ShortUsage: "proj get [flags] <name>...",
		ShortHelp:  "Clone projects from GitHub",
		LongHelp: `Clone one or more projects from GitHub into the configured directory structure.

The project name can be:
  - "project" (uses default user from config)
  - "user/project" (explicit user specification)

Examples:
  proj get myrepo
  proj get johndoe/webapp
  proj get --ssh johndoe/webapp
  proj get repo1 user2/repo2`,
		FlagSet: fs,
		Exec: func(ctx context.Context, args []string) error {
			return runGet(ctx, logger, cfg, getCfg, args)
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
		if getCfg.useSSH {
			url = p.GitSSHURL()
		}

		cloneOpts := git.CloneOptions{
			URL:         url,
			Destination: p.Path,
			UseSSH:      getCfg.useSSH,
			Token:       getCfg.token,
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

package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"os"

	"github.com/gfanton/project/internal/config"
	"github.com/gfanton/project/internal/project"
	"github.com/gfanton/project/internal/workspace"
	"github.com/peterbourgon/ff/v3/ffcli"
)

func newWorkspaceCommand(logger *slog.Logger, cfg *config.Config) *ffcli.Command {
	return &ffcli.Command{
		Name:       "workspace",
		ShortUsage: "workspace <subcommand>",
		ShortHelp:  "Manage git worktrees for projects",
		LongHelp: `Manage git worktrees for projects.

Workspaces are created in <projects_root>/.workspace/<org>/<name>.<branch>/

Commands:
  add <branch> [project]     Add new workspace
  remove <branch> [project]  Remove workspace
  list [project]             List workspaces

When inside a project directory, the project parameter is optional.
When outside a project directory, the project parameter is required.`,
		Subcommands: []*ffcli.Command{
			newWorkspaceAddCommand(logger, cfg),
			newWorkspaceRemoveCommand(logger, cfg),
			newWorkspaceListCommand(logger, cfg),
		},
		Exec: func(ctx context.Context, args []string) error {
			return flag.ErrHelp
		},
	}
}

func newWorkspaceAddCommand(logger *slog.Logger, cfg *config.Config) *ffcli.Command {
	return &ffcli.Command{
		Name:       "add",
		ShortUsage: "workspace add <branch> [project]",
		ShortHelp:  "Add new workspace",
		LongHelp: `Add a new git worktree workspace.

The branch parameter specifies which branch to checkout in the workspace.
If the project parameter is not provided, the current directory must be inside a project.`,
		Exec: func(ctx context.Context, args []string) error {
			if len(args) < 1 {
				return errors.New("branch name is required")
			}

			branch := args[0]
			var projectStr string
			if len(args) > 1 {
				projectStr = args[1]
			}

			proj, err := resolveProject(cfg, projectStr)
			if err != nil {
				return err
			}

			svc := workspace.NewService(logger, cfg.RootDir)
			return svc.Add(ctx, proj, branch)
		},
	}
}

func newWorkspaceRemoveCommand(logger *slog.Logger, cfg *config.Config) *ffcli.Command {
	var removeCfg struct {
		deleteBranch bool
	}

	fs := flag.NewFlagSet("workspace remove", flag.ContinueOnError)
	fs.BoolVar(&removeCfg.deleteBranch, "delete-branch", false, "also delete the git branch")

	return &ffcli.Command{
		Name:       "remove",
		ShortUsage: "workspace remove [flags] <branch> [project]",
		ShortHelp:  "Remove workspace",
		LongHelp: `Remove a git worktree workspace.

The branch parameter specifies which workspace branch to remove.
If the project parameter is not provided, the current directory must be inside a project.

FLAGS
  --delete-branch    Also delete the git branch (use with caution)`,
		FlagSet: fs,
		Exec: func(ctx context.Context, args []string) error {
			if len(args) < 1 {
				return errors.New("branch name is required")
			}

			branch := args[0]
			var projectStr string
			if len(args) > 1 {
				projectStr = args[1]
			}

			proj, err := resolveProject(cfg, projectStr)
			if err != nil {
				return err
			}

			svc := workspace.NewService(logger, cfg.RootDir)
			return svc.Remove(ctx, proj, branch, removeCfg.deleteBranch)
		},
	}
}

func newWorkspaceListCommand(logger *slog.Logger, cfg *config.Config) *ffcli.Command {
	return &ffcli.Command{
		Name:       "list",
		ShortUsage: "workspace list [project]",
		ShortHelp:  "List workspaces",
		LongHelp: `List git worktree workspaces for a project.

If the project parameter is not provided, the current directory must be inside a project.`,
		Exec: func(ctx context.Context, args []string) error {
			var projectStr string
			if len(args) > 0 {
				projectStr = args[0]
			}

			proj, err := resolveProject(cfg, projectStr)
			if err != nil {
				return err
			}

			svc := workspace.NewService(logger, cfg.RootDir)
			workspaces, err := svc.List(ctx, proj)
			if err != nil {
				return err
			}

			if len(workspaces) == 0 {
				fmt.Printf("No workspaces found for %s/%s\n", proj.Organisation, proj.Name)
				return nil
			}

			fmt.Printf("Workspaces for %s/%s:\n", proj.Organisation, proj.Name)
			for _, ws := range workspaces {
				fmt.Printf("  %-20s %s\n", ws.Branch, ws.Path)
			}

			return nil
		},
	}
}

func resolveProject(cfg *config.Config, projectStr string) (project.Project, error) {
	if projectStr != "" {
		proj, err := project.ParseProject(cfg.RootDir, cfg.RootUser, projectStr)
		if err != nil {
			return project.Project{}, err
		}
		return *proj, nil
	}

	wd, err := os.Getwd()
	if err != nil {
		return project.Project{}, fmt.Errorf("failed to get working directory: %w", err)
	}

	proj, err := project.FindFromPath(cfg.RootDir, wd)
	if err != nil {
		return project.Project{}, fmt.Errorf("not inside a project directory and no project specified: %w", err)
	}

	return *proj, nil
}

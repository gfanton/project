package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"

	"github.com/gfanton/projects"
	"github.com/gfanton/projects/internal/config"
	"github.com/peterbourgon/ff/v4"
)

func newWorkspaceCommand(logger *slog.Logger, cfg *config.Config, projectsCfg *projects.Config, projectsLogger projects.Logger) *ff.Command {
	return &ff.Command{
		Name:      "workspace",
		Usage:     "workspace <subcommand>",
		ShortHelp: "Manage git worktrees for projects",
		LongHelp: `Manage git worktrees for projects.

Workspaces are created in <projects_root>/.workspace/<org>/<name>/<branch>/

Commands:
  add <branch|#pr> [project]     Add new workspace (supports PR checkout with #123)
  remove <branch> [project]      Remove workspace
  list [project]                 List workspaces

When inside a project directory, the project parameter is optional.
When outside a project directory, the project parameter is required.`,
		Subcommands: []*ff.Command{
			newWorkspaceAddCommand(projectsCfg, projectsLogger),
			newWorkspaceRemoveCommand(projectsCfg, projectsLogger),
			newWorkspaceListCommand(projectsCfg, projectsLogger),
		},
		Exec: func(ctx context.Context, args []string) error {
			return ff.ErrHelp
		},
	}
}

func newWorkspaceAddCommand(projectsCfg *projects.Config, projectsLogger projects.Logger) *ff.Command {
	return &ff.Command{
		Name:      "add",
		Usage:     "workspace add <branch|#pr> [project]",
		ShortHelp: "Add new workspace",
		LongHelp: `Add a new git worktree workspace.

The branch parameter specifies which branch to checkout in the workspace.
You can also checkout a pull request by using #<number> format (e.g., #123).

If the project parameter is not provided, the current directory must be inside a project.

Examples:
  proj workspace add feature-branch     # Create workspace for branch
  proj workspace add #123               # Create workspace for PR #123`,
		Exec: func(ctx context.Context, args []string) error {
			if len(args) < 1 {
				return errors.New("branch name is required")
			}

			branch := args[0]
			var projectStr string
			if len(args) > 1 {
				projectStr = args[1]
			}

			proj, err := resolveProject(projectsCfg, projectsLogger, projectStr)
			if err != nil {
				return err
			}

			svc := projects.NewWorkspaceService(projectsCfg, projectsLogger)
			return svc.Add(ctx, *proj, branch)
		},
	}
}

type workspaceRemoveConfig struct {
	DeleteBranch bool
}

func newWorkspaceRemoveCommand(projectsCfg *projects.Config, projectsLogger projects.Logger) *ff.Command {
	removeCfg := &workspaceRemoveConfig{}
	fs := ff.NewFlagSet("workspace remove")
	fs.BoolVar(&removeCfg.DeleteBranch, 0, "delete-branch", "also delete the git branch (use with caution)")

	return &ff.Command{
		Name:      "remove",
		Usage:     "workspace remove [flags] <branch> [project]",
		ShortHelp: "Remove workspace",
		LongHelp: `Remove a git worktree workspace.

The branch parameter specifies which workspace branch to remove.
If the project parameter is not provided, the current directory must be inside a project.

FLAGS
  --delete-branch    Also delete the git branch (use with caution)`,
		Flags: fs,
		Exec: func(ctx context.Context, args []string) error {
			if len(args) < 1 {
				return errors.New("branch name is required")
			}

			branch := args[0]
			var projectStr string
			if len(args) > 1 {
				projectStr = args[1]
			}

			proj, err := resolveProject(projectsCfg, projectsLogger, projectStr)
			if err != nil {
				return err
			}

			svc := projects.NewWorkspaceService(projectsCfg, projectsLogger)
			return svc.Remove(ctx, *proj, branch, removeCfg.DeleteBranch)
		},
	}
}

func newWorkspaceListCommand(projectsCfg *projects.Config, projectsLogger projects.Logger) *ff.Command {
	return &ff.Command{
		Name:      "list",
		Usage:     "workspace list [project]",
		ShortHelp: "List workspaces",
		LongHelp: `List git worktree workspaces for a project.

If the project parameter is not provided, the current directory must be inside a project.`,
		Exec: func(ctx context.Context, args []string) error {
			var projectStr string
			if len(args) > 0 {
				projectStr = args[0]
			}

			proj, err := resolveProject(projectsCfg, projectsLogger, projectStr)
			if err != nil {
				return err
			}

			svc := projects.NewWorkspaceService(projectsCfg, projectsLogger)
			workspaces, err := svc.List(ctx, *proj)
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

func resolveProject(projectsCfg *projects.Config, projectsLogger projects.Logger, projectStr string) (*projects.Project, error) {
	projectSvc := projects.NewProjectService(projectsCfg, projectsLogger)

	if projectStr != "" {
		return projectSvc.ParseProject(projectStr)
	}

	wd, err := os.Getwd()
	if err != nil {
		return nil, fmt.Errorf("failed to get working directory: %w", err)
	}

	proj, err := projectSvc.FindFromPath(wd)
	if err != nil {
		return nil, fmt.Errorf("not inside a project directory and no project specified: %w", err)
	}

	return proj, nil
}

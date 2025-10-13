package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"strings"

	"github.com/gfanton/projects"
	"github.com/peterbourgon/ff/v3/ffcli"
)

func newWindowCommand(logger *slog.Logger, projectsCfg *projects.Config, projectsLogger projects.Logger) *ffcli.Command {
	return &ffcli.Command{
		Name:       "window",
		ShortUsage: "proj-tmux window <subcommand>",
		ShortHelp:  "Manage tmux windows for project workspaces",
		LongHelp: `Manage tmux windows for project workspaces.

Commands:
  create <workspace> [project]    Create window for workspace
  list [project]                  List workspace windows
  switch <workspace> [project]    Switch to workspace window`,
		Subcommands: []*ffcli.Command{
			newWindowCreateCommand(logger, projectsCfg, projectsLogger),
			newWindowListCommand(logger, projectsCfg, projectsLogger),
			newWindowSwitchCommand(logger, projectsCfg, projectsLogger),
		},
		Exec: func(ctx context.Context, args []string) error {
			return flag.ErrHelp
		},
	}
}

func newWindowCreateCommand(logger *slog.Logger, projectsCfg *projects.Config, projectsLogger projects.Logger) *ffcli.Command {
	var createCfg struct {
		autoSwitch bool
	}

	fs := flag.NewFlagSet("window create", flag.ExitOnError)
	fs.BoolVar(&createCfg.autoSwitch, "switch", true, "automatically switch to created window")

	return &ffcli.Command{
		Name:       "create",
		ShortUsage: "proj-tmux window create [flags] <workspace> [project]",
		ShortHelp:  "Create tmux window for workspace",
		LongHelp: `Create a tmux window for the specified workspace.

The window will be created in the appropriate project session and will
be named after the workspace branch. The working directory will be set
to the workspace path.

FLAGS:
  --switch    Automatically switch to the created window (default: true)`,
		FlagSet: fs,
		Exec: func(ctx context.Context, args []string) error {
			if len(args) < 1 {
				return fmt.Errorf("workspace name is required")
			}

			workspace := args[0]
			var projectName string
			if len(args) > 1 {
				projectName = args[1]
			}

			return runWindowCreate(ctx, logger, projectsCfg, projectsLogger, workspace, projectName, createCfg.autoSwitch)
		},
	}
}

func newWindowListCommand(logger *slog.Logger, projectsCfg *projects.Config, projectsLogger projects.Logger) *ffcli.Command {
	return &ffcli.Command{
		Name:       "list",
		ShortUsage: "proj-tmux window list [project]",
		ShortHelp:  "List workspace windows for project",
		LongHelp:   `List all tmux windows in the current or specified project session.`,
		Exec: func(ctx context.Context, args []string) error {
			var projectName string
			if len(args) > 0 {
				projectName = args[0]
			}

			return runWindowList(ctx, logger, projectsCfg, projectsLogger, projectName)
		},
	}
}

func newWindowSwitchCommand(logger *slog.Logger, projectsCfg *projects.Config, projectsLogger projects.Logger) *ffcli.Command {
	return &ffcli.Command{
		Name:       "switch",
		ShortUsage: "proj-tmux window switch <workspace> [project]",
		ShortHelp:  "Switch to workspace window",
		LongHelp:   `Switch to the tmux window for the specified workspace. Creates the window if it doesn't exist.`,
		Exec: func(ctx context.Context, args []string) error {
			if len(args) < 1 {
				return fmt.Errorf("workspace name is required")
			}

			workspace := args[0]
			var projectName string
			if len(args) > 1 {
				projectName = args[1]
			}

			return runWindowSwitch(ctx, logger, projectsCfg, projectsLogger, workspace, projectName)
		},
	}
}

func runWindowCreate(ctx context.Context, logger *slog.Logger, projectsCfg *projects.Config, projectsLogger projects.Logger, workspace, projectName string, autoSwitch bool) error {
	project, err := resolveProjectForWindow(projectsCfg, projectsLogger, projectName)
	if err != nil {
		return err
	}

	workspaceSvc := projects.NewWorkspaceService(projectsCfg, projectsLogger)
	tmuxSvc := NewTmuxService(logger)

	// Get workspace details
	workspaces, err := workspaceSvc.List(ctx, *project)
	if err != nil {
		return fmt.Errorf("failed to list workspaces: %w", err)
	}

	var targetWorkspace *projects.Workspace
	for _, ws := range workspaces {
		if ws.Branch == workspace {
			targetWorkspace = &ws
			break
		}
	}

	if targetWorkspace == nil {
		return fmt.Errorf("workspace '%s' not found in project %s", workspace, project.String())
	}

	sessionName := generateSessionName(project)
	windowName := workspace

	logger.Debug("creating window", "project", project.String(), "workspace", workspace, "session", sessionName, "window", windowName)

	// Ensure project session exists
	sessionExists, err := tmuxSvc.SessionExists(ctx, sessionName)
	if err != nil {
		return fmt.Errorf("failed to check session existence: %w", err)
	}

	if !sessionExists {
		logger.Info("creating project session first", "session", sessionName)
		if err := tmuxSvc.NewSession(ctx, sessionName, project.Path); err != nil {
			return fmt.Errorf("failed to create project session: %w", err)
		}
	}

	// Check if window already exists
	windowExists, err := tmuxSvc.WindowExists(ctx, sessionName, windowName)
	if err != nil {
		return fmt.Errorf("failed to check window existence: %w", err)
	}

	if windowExists {
		logger.Info("window already exists", "window", windowName, "session", sessionName)
		if autoSwitch {
			return tmuxSvc.SwitchWindow(ctx, sessionName, windowName)
		}
		return nil
	}

	// Create new window
	if err := tmuxSvc.NewWindow(ctx, sessionName, windowName, targetWorkspace.Path); err != nil {
		return fmt.Errorf("failed to create window: %w", err)
	}

	logger.Info("window created", "window", windowName, "session", sessionName, "workspace", targetWorkspace.Path)

	if autoSwitch {
		return tmuxSvc.SwitchWindow(ctx, sessionName, windowName)
	}

	return nil
}

func runWindowList(ctx context.Context, logger *slog.Logger, projectsCfg *projects.Config, projectsLogger projects.Logger, projectName string) error {
	project, err := resolveProjectForWindow(projectsCfg, projectsLogger, projectName)
	if err != nil {
		return err
	}

	sessionName := generateSessionName(project)
	tmuxSvc := NewTmuxService(logger)

	// Check if session exists
	sessionExists, err := tmuxSvc.SessionExists(ctx, sessionName)
	if err != nil {
		return fmt.Errorf("failed to check session existence: %w", err)
	}

	if !sessionExists {
		fmt.Printf("No session found for project %s\n", project.String())
		return nil
	}

	// List windows in the session
	windows, err := tmuxSvc.ListWindows(ctx, sessionName)
	if err != nil {
		return fmt.Errorf("failed to list windows: %w", err)
	}

	if len(windows) == 0 {
		fmt.Printf("No windows found in project session %s\n", project.String())
		return nil
	}

	fmt.Printf("Windows in project %s:\n", project.String())
	for _, window := range windows {
		fmt.Printf("  %s\n", window)
	}

	return nil
}

func runWindowSwitch(ctx context.Context, logger *slog.Logger, projectsCfg *projects.Config, projectsLogger projects.Logger, workspace, projectName string) error {
	// Create window if it doesn't exist, then switch
	if err := runWindowCreate(ctx, logger, projectsCfg, projectsLogger, workspace, projectName, false); err != nil {
		return err
	}

	project, err := resolveProjectForWindow(projectsCfg, projectsLogger, projectName)
	if err != nil {
		return err
	}

	sessionName := generateSessionName(project)
	windowName := workspace

	tmuxSvc := NewTmuxService(logger)
	return tmuxSvc.SwitchWindow(ctx, sessionName, windowName)
}

// resolveProjectForWindow resolves project for window operations
func resolveProjectForWindow(projectsCfg *projects.Config, projectsLogger projects.Logger, projectName string) (*projects.Project, error) {
	projectSvc := projects.NewProjectService(projectsCfg, projectsLogger)

	if projectName != "" {
		return projectSvc.ParseProject(projectName)
	}

	// Try to detect from current tmux session
	if currentSession := os.Getenv("TMUX_SESSION"); currentSession != "" && strings.HasPrefix(currentSession, "proj-") {
		if projectStr := extractProjectFromSession(currentSession); projectStr != "" {
			return projectSvc.ParseProject(projectStr)
		}
	}

	// Fall back to working directory
	wd, err := os.Getwd()
	if err != nil {
		return nil, fmt.Errorf("failed to get working directory: %w", err)
	}

	project, err := projectSvc.FindFromPath(wd)
	if err != nil {
		return nil, fmt.Errorf("not inside a project directory and no project specified: %w", err)
	}

	return project, nil
}

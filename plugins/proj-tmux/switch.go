package main

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"github.com/gfanton/projects"
	"github.com/peterbourgon/ff/v4"
)

type switchConfig struct {
	CreateSession bool
	CreateWindow  bool
}

func newSwitchCommand(logger *slog.Logger, projectsCfg *projects.Config, projectsLogger projects.Logger) *ff.Command {
	switchCfg := &switchConfig{CreateSession: true, CreateWindow: true}
	fs := ff.NewFlagSet("switch")
	fs.BoolVar(&switchCfg.CreateSession, 0, "create-session", "create session if it doesn't exist")
	fs.BoolVar(&switchCfg.CreateWindow, 0, "create-window", "create window if it doesn't exist (for workspace targets)")

	return &ff.Command{
		Name:      "switch",
		Usage:     "proj-tmux switch [flags] <target>",
		ShortHelp: "Quick switch to project or workspace",
		LongHelp: `Quick switch to a project session or workspace window.

Targets can be:
  project               Switch to project session (e.g., 'gfanton/projects')
  project:workspace     Switch to workspace window (e.g., 'gfanton/projects:feature')

FLAGS:
  --create-session    Create session if it doesn't exist (default: true)
  --create-window     Create window if it doesn't exist for workspace targets (default: true)`,
		Flags: fs,
		Exec: func(ctx context.Context, args []string) error {
			if len(args) < 1 {
				return fmt.Errorf("target is required")
			}

			target := args[0]
			return runSwitch(ctx, logger, projectsCfg, projectsLogger, target, switchCfg.CreateSession, switchCfg.CreateWindow)
		},
	}
}

func runSwitch(ctx context.Context, logger *slog.Logger, projectsCfg *projects.Config, projectsLogger projects.Logger, target string, createSession, createWindow bool) error {
	// Parse target: project or project:workspace
	if strings.Contains(target, ":") {
		// Workspace target
		parts := strings.SplitN(target, ":", 2)
		projectName := parts[0]
		workspace := parts[1]

		logger.Debug("switching to workspace", "project", projectName, "workspace", workspace)

		if createWindow {
			return runWindowSwitch(ctx, logger, projectsCfg, projectsLogger, workspace, projectName)
		} else {
			// Just switch to existing window
			projectSvc := projects.NewProjectService(projectsCfg, projectsLogger)
			project, err := projectSvc.ParseProject(projectName)
			if err != nil {
				return fmt.Errorf("invalid project name: %w", err)
			}

			sessionName := generateSessionName(project)
			windowName := workspace

			tmuxSvc := NewTmuxService(logger)
			return tmuxSvc.SwitchWindow(ctx, sessionName, windowName)
		}
	} else {
		// Project session target
		projectName := target
		logger.Debug("switching to project session", "project", projectName)

		if createSession {
			return runSessionSwitch(ctx, logger, projectsCfg, projectsLogger, projectName)
		} else {
			// Just switch to existing session
			projectSvc := projects.NewProjectService(projectsCfg, projectsLogger)
			project, err := projectSvc.ParseProject(projectName)
			if err != nil {
				return fmt.Errorf("invalid project name: %w", err)
			}

			sessionName := generateSessionName(project)
			tmuxSvc := NewTmuxService(logger)
			return tmuxSvc.SwitchSession(ctx, sessionName)
		}
	}
}

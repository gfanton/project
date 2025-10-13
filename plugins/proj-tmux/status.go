package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"strings"

	"github.com/gfanton/projects"
	"github.com/peterbourgon/ff/v3/ffcli"
)

func newStatusCommand(logger *slog.Logger, projectsCfg *projects.Config, projectsLogger projects.Logger) *ffcli.Command {
	var statusCfg struct {
		format string
		short  bool
	}

	fs := flag.NewFlagSet("status", flag.ExitOnError)
	fs.StringVar(&statusCfg.format, "format", "#{project}", "status format string")
	fs.BoolVar(&statusCfg.short, "short", false, "show short status")

	return &ffcli.Command{
		Name:       "status",
		ShortUsage: "proj-tmux status [flags]",
		ShortHelp:  "Show project status for tmux status bar",
		LongHelp: `Show project and workspace status information for tmux status bar.

Format variables:
  #{project}      Project name (org/name)
  #{org}          Organization name
  #{name}         Project name only
  #{workspace}    Current workspace (if any)
  #{session}      Tmux session name
  #{window}       Tmux window name

FLAGS:
  --format        Custom format string (default: "#{project}")
  --short         Show abbreviated status`,
		FlagSet: fs,
		Exec: func(ctx context.Context, args []string) error {
			return runStatus(ctx, logger, projectsCfg, projectsLogger, statusCfg.format, statusCfg.short)
		},
	}
}

func runStatus(ctx context.Context, logger *slog.Logger, projectsCfg *projects.Config, projectsLogger projects.Logger, format string, short bool) error {
	tmuxSvc := NewTmuxService(logger)
	projectSvc := projects.NewProjectService(projectsCfg, projectsLogger)
	workspaceSvc := projects.NewWorkspaceService(projectsCfg, projectsLogger)

	// Get current tmux session and window
	currentSession, err := tmuxSvc.CurrentSession(ctx)
	if err != nil {
		logger.Debug("failed to get current session", "error", err)
		currentSession = ""
	}

	var currentWindow string
	if currentSession != "" {
		cmd := []string{"tmux", "display-message", "-p", "#{window_name}"}
		if output, err := runCommand(ctx, cmd); err == nil {
			currentWindow = strings.TrimSpace(output)
		}
	}

	// Determine current project
	var currentProject *projects.Project
	var currentWorkspace string

	// Try to extract from tmux session name first
	if currentSession != "" && strings.HasPrefix(currentSession, "proj-") {
		if projectStr := extractProjectFromSession(currentSession); projectStr != "" {
			if proj, err := projectSvc.ParseProject(projectStr); err == nil {
				currentProject = proj

				// Check if current window corresponds to a workspace
				if currentWindow != "" && currentWindow != "0" {
					workspaces, err := workspaceSvc.List(ctx, *currentProject)
					if err == nil {
						for _, ws := range workspaces {
							if ws.Branch == currentWindow {
								currentWorkspace = ws.Branch
								break
							}
						}
					}
				}
			}
		}
	}

	// Fall back to working directory if no project found from session
	if currentProject == nil {
		wd, err := os.Getwd()
		if err == nil {
			if proj, err := projectSvc.FindFromPath(wd); err == nil {
				currentProject = proj
			}
		}
	}

	// If no project found, output empty or minimal status
	if currentProject == nil {
		if short {
			fmt.Print("")
		} else {
			fmt.Print("no project")
		}
		return nil
	}

	// Build status output
	status := buildStatus(currentProject, currentWorkspace, currentSession, currentWindow, format, short)
	fmt.Print(status)

	return nil
}

func buildStatus(project *projects.Project, workspace, session, window, format string, short bool) string {
	if short {
		if workspace != "" {
			return fmt.Sprintf("%s:%s", project.Name, workspace)
		}
		return project.Name
	}

	// Default format
	if format == "" || format == "#{project}" {
		if workspace != "" {
			return fmt.Sprintf("%s:%s", project.String(), workspace)
		}
		return project.String()
	}

	// Custom format substitution
	result := format
	result = strings.ReplaceAll(result, "#{project}", project.String())
	result = strings.ReplaceAll(result, "#{org}", project.Organisation)
	result = strings.ReplaceAll(result, "#{name}", project.Name)
	result = strings.ReplaceAll(result, "#{workspace}", workspace)
	result = strings.ReplaceAll(result, "#{session}", session)
	result = strings.ReplaceAll(result, "#{window}", window)

	return result
}

// runCommand executes a command and returns its output
func runCommand(ctx context.Context, args []string) (string, error) {
	if len(args) == 0 {
		return "", fmt.Errorf("no command specified")
	}

	cmd := args[0]
	cmdArgs := args[1:]

	execCmd := exec.CommandContext(ctx, cmd, cmdArgs...)
	output, err := execCmd.Output()
	if err != nil {
		return "", err
	}

	return string(output), nil
}

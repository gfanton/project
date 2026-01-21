package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"strings"

	"github.com/gfanton/projects"
	"github.com/peterbourgon/ff/v4"
)

func newSessionCommand(logger *slog.Logger, projectsCfg *projects.Config, projectsLogger projects.Logger) *ff.Command {
	return &ff.Command{
		Name:      "session",
		Usage:     "proj-tmux session <subcommand>",
		ShortHelp: "Manage tmux sessions for projects",
		LongHelp: `Manage tmux sessions for projects.

Commands:
  create <project>    Create or switch to project session
  list                List project sessions
  current             Show current project context
  switch <project>    Switch to project session`,
		Subcommands: []*ff.Command{
			newSessionCreateCommand(logger, projectsCfg, projectsLogger),
			newSessionListCommand(logger, projectsCfg, projectsLogger),
			newSessionCurrentCommand(logger, projectsCfg, projectsLogger),
			newSessionSwitchCommand(logger, projectsCfg, projectsLogger),
		},
		Exec: func(ctx context.Context, args []string) error {
			return ff.ErrHelp
		},
	}
}

type sessionCreateConfig struct {
	AutoSwitch bool
}

func newSessionCreateCommand(logger *slog.Logger, projectsCfg *projects.Config, projectsLogger projects.Logger) *ff.Command {
	createCfg := &sessionCreateConfig{AutoSwitch: true}
	fs := ff.NewFlagSet("session create")
	fs.BoolVar(&createCfg.AutoSwitch, 0, "switch", "automatically switch to created session")

	return &ff.Command{
		Name:      "create",
		Usage:     "proj-tmux session create [flags] <project>",
		ShortHelp: "Create tmux session for project",
		LongHelp: `Create a tmux session for the specified project.

The session will be named using the format: proj-<org>-<name>
If the session already exists, this command will switch to it.

FLAGS:
  --switch    Automatically switch to the created session (default: true)`,
		Flags: fs,
		Exec: func(ctx context.Context, args []string) error {
			if len(args) < 1 {
				return fmt.Errorf("project name is required")
			}

			projectName := args[0]
			return runSessionCreate(ctx, logger, projectsCfg, projectsLogger, projectName, createCfg.AutoSwitch)
		},
	}
}

func newSessionListCommand(logger *slog.Logger, projectsCfg *projects.Config, projectsLogger projects.Logger) *ff.Command {
	return &ff.Command{
		Name:      "list",
		Usage:     "proj-tmux session list",
		ShortHelp: "List project tmux sessions",
		LongHelp:  `List all tmux sessions that are managed by proj-tmux.`,
		Exec: func(ctx context.Context, args []string) error {
			return runSessionList(ctx, logger, projectsCfg, projectsLogger)
		},
	}
}

func newSessionCurrentCommand(logger *slog.Logger, projectsCfg *projects.Config, projectsLogger projects.Logger) *ff.Command {
	return &ff.Command{
		Name:      "current",
		Usage:     "proj-tmux session current",
		ShortHelp: "Show current project context",
		LongHelp:  `Show the current project context based on tmux session or working directory.`,
		Exec: func(ctx context.Context, args []string) error {
			return runSessionCurrent(ctx, logger, projectsCfg, projectsLogger)
		},
	}
}

func newSessionSwitchCommand(logger *slog.Logger, projectsCfg *projects.Config, projectsLogger projects.Logger) *ff.Command {
	return &ff.Command{
		Name:      "switch",
		Usage:     "proj-tmux session switch <project>",
		ShortHelp: "Switch to project session",
		LongHelp:  `Switch to the tmux session for the specified project. Creates the session if it doesn't exist.`,
		Exec: func(ctx context.Context, args []string) error {
			if len(args) < 1 {
				return fmt.Errorf("project name is required")
			}

			projectName := args[0]
			return runSessionSwitch(ctx, logger, projectsCfg, projectsLogger, projectName)
		},
	}
}

func runSessionCreate(ctx context.Context, logger *slog.Logger, projectsCfg *projects.Config, projectsLogger projects.Logger, projectName string, autoSwitch bool) error {
	projectSvc := projects.NewProjectService(projectsCfg, projectsLogger)

	tmuxSvc := newTmuxServiceFromEnv(logger)

	// Parse and validate project
	project, err := projectSvc.ParseProject(projectName)
	if err != nil {
		return fmt.Errorf("invalid project name: %w", err)
	}

	sessionName := generateSessionName(project)
	logger.Debug("creating session", "project", project.String(), "session", sessionName)

	// Check if session already exists
	exists, err := tmuxSvc.SessionExists(ctx, sessionName)
	if err != nil {
		return fmt.Errorf("failed to check if session exists: %w", err)
	}

	if exists {
		logger.Info("session already exists", "session", sessionName)
		if autoSwitch {
			return tmuxSvc.SwitchSession(ctx, sessionName)
		}
		return nil
	}

	// Create new session
	if err := tmuxSvc.NewSession(ctx, sessionName, project.Path); err != nil {
		return fmt.Errorf("failed to create session: %w", err)
	}

	logger.Info("session created", "session", sessionName, "project", project.String())

	if autoSwitch {
		return tmuxSvc.SwitchSession(ctx, sessionName)
	}

	return nil
}

func runSessionList(ctx context.Context, logger *slog.Logger, projectsCfg *projects.Config, projectsLogger projects.Logger) error {
	tmuxSvc := newTmuxServiceFromEnv(logger)

	sessions, err := tmuxSvc.ListSessions(ctx)
	if err != nil {
		return fmt.Errorf("failed to list sessions: %w", err)
	}

	// Filter to show only proj-tmux managed sessions
	var projSessions []string
	for _, session := range sessions {
		if strings.HasPrefix(session, "proj-") {
			projSessions = append(projSessions, session)
		}
	}

	if len(projSessions) == 0 {
		fmt.Println("No project sessions found")
		return nil
	}

	fmt.Println("Project sessions:")
	for _, session := range projSessions {
		// Extract project name from session name (proj-org-name -> org/name)
		if projectName := extractProjectFromSession(session); projectName != "" {
			fmt.Printf("  %s -> %s\n", session, projectName)
		} else {
			fmt.Printf("  %s\n", session)
		}
	}

	return nil
}

func runSessionCurrent(ctx context.Context, logger *slog.Logger, projectsCfg *projects.Config, projectsLogger projects.Logger) error {
	tmuxSvc := newTmuxServiceFromEnv(logger)
	projectSvc := projects.NewProjectService(projectsCfg, projectsLogger)

	// Try to get current tmux session
	currentSession, err := tmuxSvc.CurrentSession(ctx)
	if err == nil && strings.HasPrefix(currentSession, "proj-") {
		if projectName := extractProjectFromSession(currentSession); projectName != "" {
			fmt.Printf("Current project session: %s (%s)\n", projectName, currentSession)
			return nil
		}
	}

	// Fall back to working directory
	wd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get working directory: %w", err)
	}

	project, err := projectSvc.FindFromPath(wd)
	if err != nil {
		fmt.Println("Not in a project session or directory")
		return nil
	}

	fmt.Printf("Current directory project: %s\n", project.String())
	return nil
}

func runSessionSwitch(ctx context.Context, logger *slog.Logger, projectsCfg *projects.Config, projectsLogger projects.Logger, projectName string) error {
	// Create session if it doesn't exist, then switch
	if err := runSessionCreate(ctx, logger, projectsCfg, projectsLogger, projectName, false); err != nil {
		return err
	}

	projectSvc := projects.NewProjectService(projectsCfg, projectsLogger)
	project, err := projectSvc.ParseProject(projectName)
	if err != nil {
		return fmt.Errorf("invalid project name: %w", err)
	}

	sessionName := generateSessionName(project)
	tmuxSvc := newTmuxServiceFromEnv(logger)
	return tmuxSvc.SwitchSession(ctx, sessionName)
}

// newTmuxServiceFromEnv creates a TmuxService, using a custom socket if
// the TMUX_SOCKET environment variable is set (for testing).
func newTmuxServiceFromEnv(logger *slog.Logger) *TmuxService {
	if socketPath := os.Getenv("TMUX_SOCKET"); socketPath != "" {
		return NewTmuxServiceWithSocket(logger, socketPath)
	}
	return NewTmuxService(logger)
}

// generateSessionName creates a tmux session name from a project.
// Format: proj-<org>_<name> (underscore separates org from name).
func generateSessionName(project *projects.Project) string {
	org := strings.ReplaceAll(project.Organisation, ".", "-")
	name := strings.ReplaceAll(project.Name, ".", "-")
	return fmt.Sprintf("proj-%s_%s", org, name)
}

// extractProjectFromSession extracts project name from session name.
// Handles both new format (proj-org_name) and legacy format (proj-org-name).
func extractProjectFromSession(sessionName string) string {
	if !strings.HasPrefix(sessionName, "proj-") {
		return ""
	}

	// Remove "proj-" prefix
	remainder := strings.TrimPrefix(sessionName, "proj-")

	// New format: split by underscore (unambiguous separator)
	if strings.Contains(remainder, "_") {
		parts := strings.SplitN(remainder, "_", 2)
		if len(parts) == 2 {
			return fmt.Sprintf("%s/%s", parts[0], parts[1])
		}
	}

	// Legacy format fallback: split by hyphen (ambiguous for hyphenated orgs)
	parts := strings.Split(remainder, "-")
	if len(parts) < 2 {
		return ""
	}

	// Simple heuristic: assume last part is name, everything before is org
	name := parts[len(parts)-1]
	org := strings.Join(parts[:len(parts)-1], "-")

	return fmt.Sprintf("%s/%s", org, name)
}

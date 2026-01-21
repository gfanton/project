package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os/exec"
	"strings"
)

// TmuxService provides tmux command execution
type TmuxService struct {
	logger     *slog.Logger
	socketPath string
}

// NewTmuxService creates a new tmux service
func NewTmuxService(logger *slog.Logger) *TmuxService {
	return &TmuxService{
		logger: logger,
	}
}

// NewTmuxServiceWithSocket creates a new tmux service with custom socket
func NewTmuxServiceWithSocket(logger *slog.Logger, socketPath string) *TmuxService {
	return &TmuxService{
		logger:     logger,
		socketPath: socketPath,
	}
}

// buildTmuxCommand builds a tmux command with optional socket
func (s *TmuxService) buildTmuxCommand(ctx context.Context, args ...string) *exec.Cmd {
	if s.socketPath != "" {
		tmuxArgs := []string{"-S", s.socketPath}
		tmuxArgs = append(tmuxArgs, args...)
		return exec.CommandContext(ctx, "tmux", tmuxArgs...)
	}
	return exec.CommandContext(ctx, "tmux", args...)
}

// SessionExists checks if a tmux session exists
func (s *TmuxService) SessionExists(ctx context.Context, sessionName string) (bool, error) {
	cmd := s.buildTmuxCommand(ctx, "has-session", "-t", sessionName)
	err := cmd.Run()
	if err != nil {
		// tmux returns non-zero exit code if session doesn't exist
		var exitError *exec.ExitError
		if errors.As(err, &exitError) {
			if exitError.ExitCode() == 1 {
				return false, nil
			}
		}
		return false, fmt.Errorf("check session existence: %w", err)
	}
	return true, nil
}

// NewSession creates a new tmux session
func (s *TmuxService) NewSession(ctx context.Context, sessionName, workingDir string) error {
	s.logger.Debug("creating tmux session", "session", sessionName, "dir", workingDir)

	cmd := s.buildTmuxCommand(ctx, "new-session", "-d", "-s", sessionName, "-c", workingDir)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to create session %s: %w", sessionName, err)
	}

	s.logger.Info("created tmux session", "session", sessionName)
	return nil
}

// SwitchSession switches to a tmux session
func (s *TmuxService) SwitchSession(ctx context.Context, sessionName string) error {
	s.logger.Debug("switching to tmux session", "session", sessionName)

	cmd := s.buildTmuxCommand(ctx, "switch-client", "-t", sessionName)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to switch to session %s: %w", sessionName, err)
	}

	s.logger.Info("switched to tmux session", "session", sessionName)
	return nil
}

// ListSessions lists all tmux sessions
func (s *TmuxService) ListSessions(ctx context.Context) ([]string, error) {
	cmd := s.buildTmuxCommand(ctx, "list-sessions", "-F", "#{session_name}")
	output, err := cmd.Output()
	if err != nil {
		// Handle case where no sessions exist
		var exitError *exec.ExitError
		if errors.As(err, &exitError) {
			if exitError.ExitCode() == 1 {
				return []string{}, nil
			}
		}
		return nil, fmt.Errorf("list sessions: %w", err)
	}

	sessions := strings.Split(strings.TrimSpace(string(output)), "\n")
	if len(sessions) == 1 && sessions[0] == "" {
		return []string{}, nil
	}

	return sessions, nil
}

// CurrentSession returns the current tmux session name
func (s *TmuxService) CurrentSession(ctx context.Context) (string, error) {
	cmd := s.buildTmuxCommand(ctx, "display-message", "-p", "#{session_name}")
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to get current session: %w", err)
	}

	return strings.TrimSpace(string(output)), nil
}

// WindowExists checks if a window exists in a session
func (s *TmuxService) WindowExists(ctx context.Context, sessionName, windowName string) (bool, error) {
	cmd := s.buildTmuxCommand(ctx, "list-windows", "-t", sessionName, "-F", "#{window_name}")
	output, err := cmd.Output()
	if err != nil {
		return false, fmt.Errorf("failed to list windows: %w", err)
	}

	windows := strings.Split(strings.TrimSpace(string(output)), "\n")
	for _, window := range windows {
		if window == windowName {
			return true, nil
		}
	}

	return false, nil
}

// NewWindow creates a new window in a session
func (s *TmuxService) NewWindow(ctx context.Context, sessionName, windowName, workingDir string) error {
	s.logger.Debug("creating tmux window", "session", sessionName, "window", windowName, "dir", workingDir)

	cmd := s.buildTmuxCommand(ctx, "new-window", "-t", sessionName, "-n", windowName, "-c", workingDir)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to create window %s in session %s: %w", windowName, sessionName, err)
	}

	s.logger.Info("created tmux window", "session", sessionName, "window", windowName)
	return nil
}

// SwitchWindow switches to a window in a session
func (s *TmuxService) SwitchWindow(ctx context.Context, sessionName, windowName string) error {
	s.logger.Debug("switching to tmux window", "session", sessionName, "window", windowName)

	target := fmt.Sprintf("%s:%s", sessionName, windowName)
	cmd := s.buildTmuxCommand(ctx, "select-window", "-t", target)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to switch to window %s in session %s: %w", windowName, sessionName, err)
	}

	s.logger.Info("switched to tmux window", "session", sessionName, "window", windowName)
	return nil
}

// ListWindows lists all windows in a session
func (s *TmuxService) ListWindows(ctx context.Context, sessionName string) ([]string, error) {
	cmd := s.buildTmuxCommand(ctx, "list-windows", "-t", sessionName, "-F", "#{window_name}")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to list windows: %w", err)
	}

	windows := strings.Split(strings.TrimSpace(string(output)), "\n")
	if len(windows) == 1 && windows[0] == "" {
		return []string{}, nil
	}

	return windows, nil
}

// KillSession kills a tmux session
func (s *TmuxService) KillSession(ctx context.Context, sessionName string) error {
	s.logger.Debug("killing tmux session", "session", sessionName)

	cmd := s.buildTmuxCommand(ctx, "kill-session", "-t", sessionName)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to kill session %s: %w", sessionName, err)
	}

	s.logger.Info("killed tmux session", "session", sessionName)
	return nil
}

// KillWindow kills a window in a session
func (s *TmuxService) KillWindow(ctx context.Context, sessionName, windowName string) error {
	s.logger.Debug("killing tmux window", "session", sessionName, "window", windowName)

	target := fmt.Sprintf("%s:%s", sessionName, windowName)
	cmd := s.buildTmuxCommand(ctx, "kill-window", "-t", target)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to kill window %s in session %s: %w", windowName, sessionName, err)
	}

	s.logger.Info("killed tmux window", "session", sessionName, "window", windowName)
	return nil
}

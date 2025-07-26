package workspace

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/gfanton/projects/internal/project"
)

type Service struct {
	logger      *slog.Logger
	projectRoot string
}

type Workspace struct {
	Project project.Project
	Branch  string
	Path    string
}

func NewService(logger *slog.Logger, projectRoot string) *Service {
	return &Service{
		logger:      logger,
		projectRoot: projectRoot,
	}
}

func (s *Service) WorkspaceDir() string {
	return filepath.Join(s.projectRoot, ".workspace")
}

func (s *Service) WorkspacePath(proj project.Project, branch string) string {
	return filepath.Join(s.WorkspaceDir(), proj.Organisation, fmt.Sprintf("%s.%s", proj.Name, branch))
}

func (s *Service) Add(ctx context.Context, proj project.Project, branch string) error {
	s.logger.Debug("adding workspace", "project", proj.Name, "org", proj.Organisation, "branch", branch)

	workspacePath := s.WorkspacePath(proj, branch)

	if _, err := os.Stat(workspacePath); err == nil {
		return fmt.Errorf("workspace already exists: %s", workspacePath)
	}

	if err := os.MkdirAll(filepath.Dir(workspacePath), 0755); err != nil {
		return fmt.Errorf("failed to create workspace directory: %w", err)
	}

	// Try to create worktree with existing branch first
	cmd := exec.CommandContext(ctx, "git", "worktree", "add", workspacePath, branch)
	cmd.Dir = proj.Path

	if output, err := cmd.CombinedOutput(); err != nil {
		// If branch doesn't exist, try creating it
		s.logger.Debug("branch doesn't exist, creating new branch", "branch", branch, "error", err, "output", string(output))

		cmd = exec.CommandContext(ctx, "git", "worktree", "add", "-b", branch, workspacePath)
		cmd.Dir = proj.Path

		if output, err := cmd.CombinedOutput(); err != nil {
			return fmt.Errorf("failed to create worktree with new branch: %w\nOutput: %s", err, string(output))
		}
		s.logger.Info("workspace created with new branch", "path", workspacePath, "branch", branch)
	} else {
		s.logger.Info("workspace created with existing branch", "path", workspacePath, "branch", branch)
	}

	return nil
}

func (s *Service) Remove(ctx context.Context, proj project.Project, branch string, deleteBranch bool) error {
	s.logger.Debug("removing workspace", "project", proj.Name, "org", proj.Organisation, "branch", branch, "deleteBranch", deleteBranch)

	workspacePath := s.WorkspacePath(proj, branch)

	if _, err := os.Stat(workspacePath); os.IsNotExist(err) {
		return fmt.Errorf("workspace does not exist: %s", workspacePath)
	}

	cmd := exec.CommandContext(ctx, "git", "worktree", "remove", workspacePath)
	cmd.Dir = proj.Path

	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to remove worktree: %w\nOutput: %s", err, string(output))
	}

	if deleteBranch {
		s.logger.Debug("deleting branch", "branch", branch)
		cmd = exec.CommandContext(ctx, "git", "branch", "-D", branch)
		cmd.Dir = proj.Path

		if output, err := cmd.CombinedOutput(); err != nil {
			s.logger.Warn("failed to delete branch", "branch", branch, "error", err, "output", string(output))
			// Don't fail the operation if branch deletion fails - workspace is already removed
		} else {
			s.logger.Info("branch deleted", "branch", branch)
		}
	}

	s.logger.Info("workspace removed", "path", workspacePath, "branch", branch)
	return nil
}

func (s *Service) List(ctx context.Context, proj project.Project) ([]Workspace, error) {
	s.logger.Debug("listing workspaces", "project", proj.Name, "org", proj.Organisation)

	cmd := exec.CommandContext(ctx, "git", "worktree", "list", "--porcelain")
	cmd.Dir = proj.Path

	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("failed to list worktrees: %w\nOutput: %s", err, string(output))
	}

	return s.parseWorktreeList(proj, string(output))
}

func (s *Service) parseWorktreeList(proj project.Project, output string) ([]Workspace, error) {
	lines := strings.Split(strings.TrimSpace(output), "\n")
	var workspaces []Workspace
	var currentWorkspace *Workspace

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			if currentWorkspace != nil {
				workspaces = append(workspaces, *currentWorkspace)
				currentWorkspace = nil
			}
			continue
		}

		if strings.HasPrefix(line, "worktree ") {
			path := strings.TrimPrefix(line, "worktree ")
			currentWorkspace = &Workspace{
				Project: proj,
				Path:    path,
			}
		} else if strings.HasPrefix(line, "branch ") && currentWorkspace != nil {
			branch := strings.TrimPrefix(line, "branch ")
			currentWorkspace.Branch = strings.TrimPrefix(branch, "refs/heads/")
		}
	}

	if currentWorkspace != nil {
		workspaces = append(workspaces, *currentWorkspace)
	}

	var filteredWorkspaces []Workspace
	workspaceDir, err := filepath.EvalSymlinks(s.WorkspaceDir())
	if err != nil {
		workspaceDir = s.WorkspaceDir()
	}

	for _, ws := range workspaces {
		wsPath := ws.Path
		if evalPath, err := filepath.EvalSymlinks(ws.Path); err == nil {
			wsPath = evalPath
		}

		if strings.HasPrefix(wsPath, workspaceDir) {
			filteredWorkspaces = append(filteredWorkspaces, ws)
		}
	}

	return filteredWorkspaces, nil
}

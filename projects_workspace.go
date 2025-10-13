package projects

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
)

// WorkspaceService provides workspace operations.
type WorkspaceService struct {
	logger Logger
	config *Config
}

// NewWorkspaceService creates a new workspace service.
func NewWorkspaceService(config *Config, logger Logger) *WorkspaceService {
	return &WorkspaceService{
		logger: logger,
		config: config,
	}
}

// WorkspaceDir returns the directory where workspaces are stored.
func (s *WorkspaceService) WorkspaceDir() string {
	return filepath.Join(s.config.RootDir, ".workspace")
}

// WorkspacePath returns the path for a specific workspace.
func (s *WorkspaceService) WorkspacePath(proj Project, branch string) string {
	return filepath.Join(s.WorkspaceDir(), proj.Organisation, fmt.Sprintf("%s.%s", proj.Name, branch))
}

// isPullRequest checks if the branch string is a PR number (#123 format)
func (s *WorkspaceService) isPullRequest(branch string) (int, bool) {
	if !strings.HasPrefix(branch, "#") {
		return 0, false
	}

	prNumStr := strings.TrimPrefix(branch, "#")
	prNum, err := strconv.Atoi(prNumStr)
	if err != nil {
		return 0, false
	}

	return prNum, true
}

// getDefaultRemote returns the first available remote, preferring 'origin'
func (s *WorkspaceService) getDefaultRemote(ctx context.Context, proj Project) (string, error) {
	cmd := exec.CommandContext(ctx, "git", "remote")
	cmd.Dir = proj.Path

	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("failed to list remotes: %w", err)
	}

	remotes := strings.Fields(strings.TrimSpace(string(output)))
	if len(remotes) == 0 {
		return "", fmt.Errorf("no git remotes found")
	}

	// Prefer 'origin' if it exists
	for _, remote := range remotes {
		if remote == "origin" {
			return remote, nil
		}
	}

	// Otherwise return the first remote
	return remotes[0], nil
}

// validatePullRequest checks if a PR exists by trying to fetch its ref
func (s *WorkspaceService) validatePullRequest(ctx context.Context, proj Project, prNum int) error {
	s.logger.Debug("validating pull request", "project", proj.Name, "pr", prNum)

	remote, err := s.getDefaultRemote(ctx, proj)
	if err != nil {
		return fmt.Errorf("failed to get remote: %w", err)
	}

	// Try to fetch the PR ref to validate it exists
	cmd := exec.CommandContext(ctx, "git", "ls-remote", remote, fmt.Sprintf("refs/pull/%d/head", prNum))
	cmd.Dir = proj.Path

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to validate PR #%d: %w\nOutput: %s", prNum, err, string(output))
	}

	if strings.TrimSpace(string(output)) == "" {
		return fmt.Errorf("pull request #%d does not exist", prNum)
	}

	s.logger.Debug("pull request validated", "pr", prNum)
	return nil
}

// addPullRequestWorkspace creates a workspace for a pull request
func (s *WorkspaceService) addPullRequestWorkspace(ctx context.Context, proj Project, prNum int, branch string) error {
	s.logger.Debug("adding pull request workspace", "project", proj.Name, "pr", prNum)

	// First validate that the PR exists
	if err := s.validatePullRequest(ctx, proj, prNum); err != nil {
		return err
	}

	remote, err := s.getDefaultRemote(ctx, proj)
	if err != nil {
		return fmt.Errorf("failed to get remote: %w", err)
	}

	workspacePath := s.WorkspacePath(proj, branch)

	if _, err := os.Stat(workspacePath); err == nil {
		return fmt.Errorf("workspace already exists: %s", workspacePath)
	}

	if err := os.MkdirAll(filepath.Dir(workspacePath), 0755); err != nil {
		return fmt.Errorf("failed to create workspace directory: %w", err)
	}

	// Fetch the PR ref first
	prRef := fmt.Sprintf("refs/pull/%d/head", prNum)
	localBranch := fmt.Sprintf("pr-%d", prNum)

	s.logger.Debug("fetching pull request", "ref", prRef, "local_branch", localBranch)

	// Fetch the PR ref
	cmd := exec.CommandContext(ctx, "git", "fetch", remote, fmt.Sprintf("%s:%s", prRef, localBranch))
	cmd.Dir = proj.Path

	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to fetch PR #%d: %w\nOutput: %s", prNum, err, string(output))
	}

	// Create worktree with the fetched PR branch
	cmd = exec.CommandContext(ctx, "git", "worktree", "add", workspacePath, localBranch)
	cmd.Dir = proj.Path

	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to create worktree for PR #%d: %w\nOutput: %s", prNum, err, string(output))
	}

	s.logger.Info("workspace created for pull request", "path", workspacePath, "pr", prNum, "branch", localBranch)
	return nil
}

// Add creates a new workspace for the given project and branch.
func (s *WorkspaceService) Add(ctx context.Context, proj Project, branch string) error {
	s.logger.Debug("adding workspace", "project", proj.Name, "org", proj.Organisation, "branch", branch)

	// Check if this is a pull request
	if prNum, isPR := s.isPullRequest(branch); isPR {
		return s.addPullRequestWorkspace(ctx, proj, prNum, branch)
	}

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

// Remove removes a workspace for the given project and branch.
func (s *WorkspaceService) Remove(ctx context.Context, proj Project, branch string, deleteBranch bool) error {
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

// List returns all workspaces for the given project.
func (s *WorkspaceService) List(ctx context.Context, proj Project) ([]Workspace, error) {
	s.logger.Debug("listing workspaces", "project", proj.Name, "org", proj.Organisation)

	cmd := exec.CommandContext(ctx, "git", "worktree", "list", "--porcelain")
	cmd.Dir = proj.Path

	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("failed to list worktrees: %w\nOutput: %s", err, string(output))
	}

	return s.parseWorktreeList(proj, string(output))
}

func (s *WorkspaceService) parseWorktreeList(proj Project, output string) ([]Workspace, error) {
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

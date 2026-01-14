package workspace

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
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

// isPullRequest checks if the branch string is a PR number (#123 format)
func (s *Service) isPullRequest(branch string) (int, bool) {
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

// validatePullRequest checks if a PR exists by trying to fetch its ref
func (s *Service) validatePullRequest(ctx context.Context, proj project.Project, prNum int) error {
	s.logger.Debug("validating pull request", "project", proj.Name, "pr", prNum)

	// Try to fetch the PR ref to validate it exists
	cmd := exec.CommandContext(ctx, "git", "ls-remote", "origin", fmt.Sprintf("refs/pull/%d/head", prNum))
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
func (s *Service) addPullRequestWorkspace(ctx context.Context, proj project.Project, prNum int, branch string) error {
	s.logger.Debug("adding pull request workspace", "project", proj.Name, "pr", prNum)

	// First validate that the PR exists
	if err := s.validatePullRequest(ctx, proj, prNum); err != nil {
		return err
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
	cmd := exec.CommandContext(ctx, "git", "fetch", "origin", fmt.Sprintf("%s:%s", prRef, localBranch))
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

func (s *Service) Add(ctx context.Context, proj project.Project, branch string) error {
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

	// Check if this is a git repository
	gitDir := filepath.Join(proj.Path, ".git")
	if _, err := os.Stat(gitDir); os.IsNotExist(err) {
		// Not a git repository, return empty list
		s.logger.Debug("not a git repository, skipping workspace list", "project", proj.Name)
		return []Workspace{}, nil
	}

	cmd := exec.CommandContext(ctx, "git", "worktree", "list", "--porcelain")
	cmd.Dir = proj.Path

	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("failed to list worktrees: %w\nOutput: %s", err, string(output))
	}

	return s.parseWorktreeList(proj, string(output))
}

// extractBranchFromPath extracts branch from workspace directory path.
// Handles branch names with slashes (e.g., feat/auth) which create subdirectories.
// e.g., path ".workspace/org/project.feature" with projectName "project" -> "feature"
// e.g., path ".workspace/org/project.feat/auth" with projectName "project" -> "feat/auth"
func extractBranchFromPath(workspacePath, projectName, workspaceDir string) string {
	// Resolve symlinks in both paths to handle macOS /var -> /private/var
	if resolved, err := filepath.EvalSymlinks(workspacePath); err == nil {
		workspacePath = resolved
	}
	if resolved, err := filepath.EvalSymlinks(workspaceDir); err == nil {
		workspaceDir = resolved
	}

	// Get path relative to workspace dir
	// workspacePath: /root/.workspace/org/project.feat/auth
	// workspaceDir: /root/.workspace
	// relPath: org/project.feat/auth
	relPath, err := filepath.Rel(workspaceDir, workspacePath)
	if err != nil {
		return ""
	}

	// Split into org and the rest: ["org", "project.feat/auth"]
	prefix := projectName + "."
	parts := strings.SplitN(relPath, string(filepath.Separator), 2)
	if len(parts) < 2 {
		return ""
	}

	// branchPath is "project.feat/auth" or "project.feature"
	branchPath := parts[1]
	if strings.HasPrefix(branchPath, prefix) {
		return strings.TrimPrefix(branchPath, prefix)
	}
	return ""
}

func (s *Service) parseWorktreeList(proj project.Project, output string) ([]Workspace, error) {
	lines := strings.Split(strings.TrimSpace(output), "\n")
	var workspaces []Workspace
	var currentWorkspace *Workspace
	workspaceDir := s.WorkspaceDir()

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
			branch := extractBranchFromPath(path, proj.Name, workspaceDir)
			currentWorkspace = &Workspace{
				Project: proj,
				Path:    path,
				Branch:  branch,
			}
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

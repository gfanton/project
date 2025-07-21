package workspace

import (
	"context"
	"io"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gfanton/project/internal/project"
)

func TestService_WorkspaceDir(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	svc := NewService(logger, "/test/root")

	expected := "/test/root/.workspace"
	if got := svc.WorkspaceDir(); got != expected {
		t.Errorf("WorkspaceDir() = %q, want %q", got, expected)
	}
}

func TestService_WorkspacePath(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	svc := NewService(logger, "/test/root")

	proj := project.Project{
		Name:         "testproject",
		Organisation: "testorg",
		Path:         "/test/root/testorg/testproject",
	}

	expected := "/test/root/.workspace/testorg/testproject.feature"
	if got := svc.WorkspacePath(proj, "feature"); got != expected {
		t.Errorf("WorkspacePath() = %q, want %q", got, expected)
	}
}

func TestService_parseWorktreeList(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	tempDir := t.TempDir()
	svc := NewService(logger, tempDir)

	proj := project.Project{
		Name:         "testproject",
		Organisation: "testorg",
		Path:         "/test/repo",
	}

	tests := []struct {
		name     string
		output   string
		expected int
	}{
		{
			name: "single worktree",
			output: `worktree /test/.workspace/testorg/testproject.feature
HEAD abc123
branch refs/heads/feature

`,
			expected: 1,
		},
		{
			name: "multiple worktrees",
			output: `worktree /test/repo
HEAD def456
branch refs/heads/main

worktree /test/.workspace/testorg/testproject.feature
HEAD abc123
branch refs/heads/feature

worktree /test/.workspace/testorg/testproject.bugfix
HEAD ghi789
branch refs/heads/bugfix

`,
			expected: 2, // Only workspace worktrees, not main repo
		},
		{
			name:     "empty output",
			output:   "",
			expected: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc.projectRoot = "/test" // Set to match the paths in output
			workspaces, err := svc.parseWorktreeList(proj, tt.output)
			if err != nil {
				t.Fatalf("parseWorktreeList() error = %v", err)
			}

			if len(workspaces) != tt.expected {
				t.Errorf("parseWorktreeList() got %d workspaces, want %d", len(workspaces), tt.expected)
			}

			for _, ws := range workspaces {
				if ws.Project.Name != proj.Name {
					t.Errorf("workspace project name = %q, want %q", ws.Project.Name, proj.Name)
				}
				if ws.Project.Organisation != proj.Organisation {
					t.Errorf("workspace project org = %q, want %q", ws.Project.Organisation, proj.Organisation)
				}
			}
		})
	}
}

func TestService_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	if !hasGitCommand() {
		t.Skip("git command not available")
	}

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	tempDir := t.TempDir()
	svc := NewService(logger, tempDir)

	// Create a test git repository
	repoDir := filepath.Join(tempDir, "testorg", "testproject")
	if err := os.MkdirAll(repoDir, 0755); err != nil {
		t.Fatalf("failed to create repo dir: %v", err)
	}

	// Initialize git repo with main as default branch
	if err := runGitCommand(repoDir, "init", "-b", "main"); err != nil {
		// Fallback for older git versions
		if err := runGitCommand(repoDir, "init"); err != nil {
			t.Fatalf("failed to init git repo: %v", err)
		}
	}

	// Configure git user (required for commits)
	if err := runGitCommand(repoDir, "config", "user.email", "test@example.com"); err != nil {
		t.Fatalf("failed to set git email: %v", err)
	}
	if err := runGitCommand(repoDir, "config", "user.name", "Test User"); err != nil {
		t.Fatalf("failed to set git name: %v", err)
	}

	// Create initial commit
	testFile := filepath.Join(repoDir, "README.md")
	if err := os.WriteFile(testFile, []byte("# Test Project\n"), 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}
	if err := runGitCommand(repoDir, "add", "README.md"); err != nil {
		t.Fatalf("failed to add file: %v", err)
	}
	if err := runGitCommand(repoDir, "commit", "-m", "Initial commit"); err != nil {
		t.Fatalf("failed to create initial commit: %v", err)
	}

	proj := project.Project{
		Name:         "testproject",
		Organisation: "testorg",
		Path:         repoDir,
	}

	ctx := context.Background()

	// Create the branch first for all tests
	if err := runGitCommand(repoDir, "checkout", "-b", "feature"); err != nil {
		t.Fatalf("failed to create feature branch: %v", err)
	}
	if err := runGitCommand(repoDir, "checkout", "main"); err != nil {
		// If main doesn't exist, try master
		if err := runGitCommand(repoDir, "checkout", "master"); err != nil {
			t.Fatalf("failed to checkout main/master: %v", err)
		}
	}

	// Test Add workspace
	t.Run("Add", func(t *testing.T) {
		err := svc.Add(ctx, proj, "feature")
		if err != nil {
			t.Fatalf("Add() error = %v", err)
		}

		// Verify workspace directory exists
		workspacePath := svc.WorkspacePath(proj, "feature")
		if _, err := os.Stat(workspacePath); os.IsNotExist(err) {
			t.Errorf("workspace directory not created: %s", workspacePath)
		}

		// Test List workspaces immediately after add
		workspaces, err := svc.List(ctx, proj)
		if err != nil {
			t.Fatalf("List() error = %v", err)
		}

		if len(workspaces) != 1 {
			// Debug: show actual git worktree list output
			cmd := exec.CommandContext(ctx, "git", "worktree", "list", "--porcelain")
			cmd.Dir = repoDir
			if output, err := cmd.CombinedOutput(); err == nil {
				t.Logf("git worktree list output:\n%s", string(output))
				t.Logf("workspace dir: %s", svc.WorkspaceDir())
			}
			t.Errorf("List() got %d workspaces, want 1", len(workspaces))
		}

		if len(workspaces) > 0 && workspaces[0].Branch != "feature" {
			t.Errorf("workspace branch = %q, want %q", workspaces[0].Branch, "feature")
		}
	})

	// Test Remove workspace
	t.Run("Remove", func(t *testing.T) {
		err := svc.Remove(ctx, proj, "feature", false)
		if err != nil {
			t.Fatalf("Remove() error = %v", err)
		}

		// Verify workspace directory is removed
		workspacePath := svc.WorkspacePath(proj, "feature")
		if _, err := os.Stat(workspacePath); !os.IsNotExist(err) {
			t.Errorf("workspace directory still exists: %s", workspacePath)
		}

		// Test List workspaces after remove
		workspaces, err := svc.List(ctx, proj)
		if err != nil {
			t.Fatalf("List() error after remove = %v", err)
		}

		if len(workspaces) != 0 {
			t.Errorf("List() after remove got %d workspaces, want 0", len(workspaces))
		}
	})

	// Test Add with new branch creation
	t.Run("AddNewBranch", func(t *testing.T) {
		err := svc.Add(ctx, proj, "newbranch")
		if err != nil {
			t.Fatalf("Add() with new branch error = %v", err)
		}

		// Verify workspace directory exists
		workspacePath := svc.WorkspacePath(proj, "newbranch")
		if _, err := os.Stat(workspacePath); os.IsNotExist(err) {
			t.Errorf("workspace directory not created: %s", workspacePath)
		}

		// Verify branch was created
		cmd := exec.CommandContext(ctx, "git", "branch", "--list", "newbranch")
		cmd.Dir = repoDir
		if output, err := cmd.CombinedOutput(); err != nil || len(strings.TrimSpace(string(output))) == 0 {
			t.Error("new branch was not created")
		}

		// Cleanup
		_ = svc.Remove(ctx, proj, "newbranch", false)
	})

	// Test Add duplicate workspace
	t.Run("AddDuplicate", func(t *testing.T) {
		// First add should succeed with new branch
		if err := svc.Add(ctx, proj, "duplicate"); err != nil {
			t.Fatalf("first Add() error = %v", err)
		}

		// Second add should fail
		err := svc.Add(ctx, proj, "duplicate")
		if err == nil {
			t.Error("second Add() should have failed but didn't")
		}

		// Cleanup
		_ = svc.Remove(ctx, proj, "duplicate", false)
	})

	// Test Remove non-existent workspace
	t.Run("RemoveNonExistent", func(t *testing.T) {
		err := svc.Remove(ctx, proj, "nonexistent", false)
		if err == nil {
			t.Error("Remove() should have failed for non-existent workspace")
		}
	})

	// Test Remove with delete branch
	t.Run("RemoveWithDeleteBranch", func(t *testing.T) {
		// First create a workspace with new branch
		err := svc.Add(ctx, proj, "delete-test")
		if err != nil {
			t.Fatalf("Add() error = %v", err)
		}

		// Remove workspace and delete branch
		err = svc.Remove(ctx, proj, "delete-test", true)
		if err != nil {
			t.Fatalf("Remove() with delete branch error = %v", err)
		}

		// Verify workspace directory is removed
		workspacePath := svc.WorkspacePath(proj, "delete-test")
		if _, err := os.Stat(workspacePath); !os.IsNotExist(err) {
			t.Errorf("workspace directory still exists: %s", workspacePath)
		}

		// Verify branch was deleted
		cmd := exec.CommandContext(ctx, "git", "branch", "--list", "delete-test")
		cmd.Dir = repoDir
		if output, err := cmd.CombinedOutput(); err == nil && len(strings.TrimSpace(string(output))) > 0 {
			t.Error("branch was not deleted")
		}
	})
}

func hasGitCommand() bool {
	_, err := exec.LookPath("git")
	return err == nil
}

func runGitCommand(dir string, args ...string) error {
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	return cmd.Run()
}
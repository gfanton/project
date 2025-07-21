package project

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/go-git/go-git/v5"
)

const (
	// GitHubProvider is the default Git provider.
	GitHubProvider = "github.com"
	// DefaultProvider is the default Git provider used for cloning.
	DefaultProvider = GitHubProvider
	// WalkDepth is the depth at which we walk project directories (user/project).
	WalkDepth = 1
)

// Project represents a project with its organization and name.
type Project struct {
	Path         string
	Name         string
	Organisation string
}

// ParseProject parses a project name into a Project struct.
// Supports formats: "project" (uses default user), "user/project".
func ParseProject(rootDir, defaultUser, name string) (*Project, error) {
	split := strings.Split(name, string(os.PathSeparator))

	switch len(split) {
	case 1:
		if defaultUser == "" {
			return nil, fmt.Errorf("no default user defined and project name '%s' doesn't include user", name)
		}
		projectName := split[0]
		projectPath := filepath.Join(rootDir, defaultUser, projectName)
		return &Project{
			Path:         projectPath,
			Name:         projectName,
			Organisation: defaultUser,
		}, nil

	case 2:
		user, projectName := split[0], split[1]
		projectPath := filepath.Join(rootDir, user, projectName)
		return &Project{
			Path:         projectPath,
			Name:         projectName,
			Organisation: user,
		}, nil

	default:
		return nil, fmt.Errorf("malformed project name '%s' (expected 'project' or 'user/project')", name)
	}
}

// String returns the string representation of the project (user/project).
func (p *Project) String() string {
	return fmt.Sprintf("%s/%s", p.Organisation, p.Name)
}

// GitHTTPURL returns the HTTP URL for cloning the project.
func (p *Project) GitHTTPURL() string {
	return fmt.Sprintf("https://%s/%s/%s.git", GitHubProvider, p.Organisation, p.Name)
}

// GitSSHURL returns the SSH URL for cloning the project.
func (p *Project) GitSSHURL() string {
	return fmt.Sprintf("git@%s:%s/%s.git", GitHubProvider, p.Organisation, p.Name)
}

// GitDir returns the path to the .git directory.
func (p *Project) GitDir() string {
	return filepath.Join(p.Path, ".git")
}

// IsGitRepository checks if the project is a Git repository.
func (p *Project) IsGitRepository() bool {
	_, err := os.Stat(p.GitDir())
	return err == nil
}

// OpenRepository opens the Git repository.
func (p *Project) OpenRepository() (*git.Repository, error) {
	return git.PlainOpen(p.Path)
}

// GitStatus represents the Git status of a project.
type GitStatus string

const (
	// GitStatusValid indicates a valid Git repository.
	GitStatusValid GitStatus = "valid"
	// GitStatusInvalid indicates an invalid Git repository.
	GitStatusInvalid GitStatus = "invalid"
	// GitStatusNotGit indicates the directory is not a Git repository.
	GitStatusNotGit GitStatus = "not a git"
)

// GetGitStatus returns the Git status of the project.
func (p *Project) GetGitStatus() GitStatus {
	_, err := p.OpenRepository()
	switch err {
	case git.ErrRepositoryNotExists:
		return GitStatusNotGit
	case nil:
		return GitStatusValid
	default:
		return GitStatusInvalid
	}
}

// WalkFunc is the function called for each project during traversal.
type WalkFunc func(d fs.DirEntry, project *Project) error

// Walk traverses the root directory and calls fn for each project found.
func Walk(rootDir string, fn WalkFunc) error {
	return filepath.WalkDir(rootDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if !d.IsDir() {
			return nil
		}

		relPath, err := filepath.Rel(rootDir, path)
		if err != nil {
			return err
		}

		sepCount := strings.Count(relPath, string(os.PathSeparator))
		if sepCount < WalkDepth {
			return nil
		}

		if sepCount > WalkDepth {
			return fs.SkipDir
		}

		// Skip any directory that starts with a dot (like .workspace, .git, .vscode, etc.)
		for _, part := range strings.Split(relPath, string(os.PathSeparator)) {
			if strings.HasPrefix(part, ".") {
				return fs.SkipDir
			}
		}

		split := strings.Split(relPath, string(os.PathSeparator))
		if len(split) != 2 {
			return nil
		}

		project := &Project{
			Path:         path,
			Name:         split[1],
			Organisation: split[0],
		}

		return fn(d, project)
	})
}

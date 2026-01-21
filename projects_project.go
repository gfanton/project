package projects

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/gfanton/projects/internal/project"
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

// ProjectService provides project operations.
type ProjectService struct {
	logger Logger
	config *Config
}

// NewProjectService creates a new project service.
func NewProjectService(config *Config, logger Logger) *ProjectService {
	return &ProjectService{
		logger: logger,
		config: config,
	}
}

// ParseProject parses a project name into a Project struct.
// Supports formats: "project" (uses default user), "user/project".
func (s *ProjectService) ParseProject(name string) (*Project, error) {
	p, err := project.ParseProject(s.config.RootDir, s.config.RootUser, name)
	if err != nil {
		return nil, err
	}
	return &Project{
		Path:         p.Path,
		Name:         p.Name,
		Organisation: p.Organisation,
	}, nil
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

// GetGitStatus returns the Git status of the project.
func (p *Project) GetGitStatus() GitStatus {
	_, err := p.OpenRepository()
	switch {
	case err == nil:
		return GitStatusValid
	case errors.Is(err, git.ErrRepositoryNotExists):
		return GitStatusNotGit
	default:
		return GitStatusInvalid
	}
}

// WalkFunc is the function called for each project during traversal.
type WalkFunc func(d fs.DirEntry, project *Project) error

// ListProjects walks the root directory and returns all projects found.
func (s *ProjectService) ListProjects() ([]*Project, error) {
	var projects []*Project

	err := s.Walk(func(d fs.DirEntry, project *Project) error {
		projects = append(projects, project)
		return nil
	})

	return projects, err
}

// Walk traverses the root directory and calls fn for each project found.
// It follows symlinks to directories to support projects added via symlinks.
func (s *ProjectService) Walk(fn WalkFunc) error {
	return project.Walk(s.config.RootDir, func(d fs.DirEntry, p *project.Project) error {
		return fn(d, &Project{
			Path:         p.Path,
			Name:         p.Name,
			Organisation: p.Organisation,
		})
	})
}

// FindFromPath finds a project from a given path by checking if it's within the root directory
// and follows the organization/project structure.
// Also handles paths inside .workspace directory.
func (s *ProjectService) FindFromPath(path string) (*Project, error) {
	p, err := project.FindFromPath(s.config.RootDir, path)
	if err != nil {
		return nil, err
	}
	return &Project{
		Path:         p.Path,
		Name:         p.Name,
		Organisation: p.Organisation,
	}, nil
}

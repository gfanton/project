package projects

import "log/slog"

// Config holds the global configuration for the project tool.
type Config struct {
	ConfigFile string
	Debug      bool
	RootDir    string
	RootUser   string
}

// Project represents a project with its organization and name.
type Project struct {
	Path         string
	Name         string
	Organisation string
}

// String returns the string representation of the project (user/project).
func (p *Project) String() string {
	return p.Organisation + "/" + p.Name
}

// GitHTTPURL returns the HTTP URL for cloning the project.
func (p *Project) GitHTTPURL() string {
	return "https://github.com/" + p.Organisation + "/" + p.Name + ".git"
}

// GitSSHURL returns the SSH URL for cloning the project.
func (p *Project) GitSSHURL() string {
	return "git@github.com:" + p.Organisation + "/" + p.Name + ".git"
}

// Workspace represents a workspace with its project and branch.
type Workspace struct {
	Project Project
	Branch  string
	Path    string
}

// SearchResult represents a search result.
type SearchResult struct {
	Project   *Project
	Workspace string // Empty for project results, branch name for workspace results
	Distance  int
}

// SearchOptions holds configuration for project queries.
type SearchOptions struct {
	Query          string
	Exclude        []string
	AbsPath        bool
	Separator      string
	Limit          int
	ShowDistance   bool
	CurrentProject *Project // When set, workspace queries without project prefix are limited to this project
}

// Logger interface for dependency injection
type Logger interface {
	Debug(msg string, args ...any)
	Info(msg string, args ...any)
	Warn(msg string, args ...any)
	Error(msg string, args ...any)
}

// SlogAdapter adapts slog.Logger to our Logger interface
type SlogAdapter struct {
	logger *slog.Logger
}

// NewSlogAdapter creates a new slog adapter
func NewSlogAdapter(logger *slog.Logger) Logger {
	return &SlogAdapter{logger: logger}
}

func (s *SlogAdapter) Debug(msg string, args ...any) {
	s.logger.Debug(msg, args...)
}

func (s *SlogAdapter) Info(msg string, args ...any) {
	s.logger.Info(msg, args...)
}

func (s *SlogAdapter) Warn(msg string, args ...any) {
	s.logger.Warn(msg, args...)
}

func (s *SlogAdapter) Error(msg string, args ...any) {
	s.logger.Error(msg, args...)
}

package query

import (
	"context"
	"fmt"
	"io/fs"
	"log/slog"
	"path/filepath"
	"sort"
	"strings"

	"github.com/lithammer/fuzzysearch/fuzzy"
	"projects/internal/project"
	"projects/internal/workspace"
)

// Options holds configuration for project queries.
type Options struct {
	Query          string
	Exclude        []string
	AbsPath        bool
	Separator      string
	Limit          int
	ShowDistance   bool
	CurrentProject *project.Project // When set, workspace queries without project prefix are limited to this project
}

// Result represents a search result.
type Result struct {
	Project   *project.Project
	Workspace string // Empty for project results, branch name for workspace results
	Distance  int
}

// Service provides project querying functionality.
type Service struct {
	logger           *slog.Logger
	rootDir          string
	workspaceService *workspace.Service
}

// NewService creates a new query service.
func NewService(logger *slog.Logger, rootDir string) *Service {
	return &Service{
		logger:           logger,
		rootDir:          rootDir,
		workspaceService: workspace.NewService(logger, rootDir),
	}
}

// Search searches for projects and workspaces matching the given options.
func (s *Service) Search(ctx context.Context, opts Options) ([]*Result, error) {
	s.logger.Debug("searching projects and workspaces",
		"query", opts.Query,
		"exclude", opts.Exclude,
	)

	excludeMap := make(map[string]bool)

	// Build exclude map
	for _, exclude := range opts.Exclude {
		exclude = strings.TrimSpace(exclude)
		if exclude == "" {
			continue
		}

		abs, err := filepath.Abs(exclude)
		if err != nil {
			return nil, fmt.Errorf("invalid exclude path '%s': %w", exclude, err)
		}
		excludeMap[abs] = true
	}

	// Check if query contains workspace syntax (contains ':')
	isWorkspaceQuery := strings.Contains(opts.Query, ":")

	if isWorkspaceQuery {
		return s.searchWorkspaces(ctx, opts, excludeMap)
	}

	return s.searchProjects(ctx, opts, excludeMap)
}

func (s *Service) searchProjects(ctx context.Context, opts Options, excludeMap map[string]bool) ([]*Result, error) {
	var results []*Result

	qLower := strings.ToLower(opts.Query)
	qOrg, qName, qHasOrg := strings.Cut(qLower, "/")

	err := project.Walk(s.rootDir, func(d fs.DirEntry, p *project.Project) error {
		// Check if project should be excluded
		if excludeMap[p.Path] {
			s.logger.Debug("excluding project", "path", p.Path)
			return filepath.SkipDir
		}

		if opts.Query == "" {
			results = append(results, &Result{
				Project:   p,
				Workspace: "",
				Distance:  1,
			})
			return nil
		}

		// Calculate match distance
		projectName := p.String()
		distance := fuzzy.RankMatchFold(opts.Query, projectName)
		if distance < 0 {
			return nil
		}

		projectLower := strings.ToLower(projectName)

		// Split project name into parts (org/name)
		pOrg, pName, _ := strings.Cut(projectLower, "/")

		if qHasOrg {
			if qOrg != pOrg {
				return nil
			}

			if qName == pName {
				distance = 0
			} else {
				distance = fuzzy.RankMatchFold(qName, pName)
			}
		} else {
			// TODO: improve this part to avoid multiple useless compare
			if !strings.Contains(pOrg, qLower) {
				distance += 100
			}

			if !strings.Contains(pName, qLower) {
				distance += 50
			}

			switch qLower {
			case pOrg:
				distance += 1
			case pName:
				distance += 2
			default:
				distance += 3 + fuzzy.RankMatchFold(qLower, projectLower)
			}
		}

		results = append(results, &Result{
			Project:   p,
			Workspace: "",
			Distance:  distance,
		})

		s.logger.Debug("found matching project",
			"name", projectName,
			"distance", distance,
		)

		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to walk projects: %w", err)
	}

	return s.sortAndLimitResults(results, opts), nil
}

func (s *Service) searchWorkspaces(ctx context.Context, opts Options, excludeMap map[string]bool) ([]*Result, error) {
	var results []*Result

	// Parse workspace query: project_part:branch_part
	projectPart, branchPart, _ := strings.Cut(opts.Query, ":")
	projectPart = strings.TrimSpace(projectPart)
	branchPart = strings.TrimSpace(branchPart)

	s.logger.Debug("searching workspaces", "projectPart", projectPart, "branchPart", branchPart)

	err := project.Walk(s.rootDir, func(d fs.DirEntry, p *project.Project) error {
		// Check if project should be excluded
		if excludeMap[p.Path] {
			s.logger.Debug("excluding project", "path", p.Path)
			return filepath.SkipDir
		}

		// If project part is specified, check if this project matches
		if projectPart != "" {
			projectName := strings.ToLower(p.String())
			if !s.matchesProject(projectPart, projectName) {
				return nil
			}
		} else if opts.CurrentProject != nil {
			// If no project part specified but we have a current project context,
			// only search within the current project
			if p.Path != opts.CurrentProject.Path {
				return nil
			}
		}

		// Get workspaces for this project
		workspaces, err := s.workspaceService.List(ctx, *p)
		if err != nil {
			s.logger.Debug("failed to list workspaces for project", "project", p.String(), "error", err)
			return nil // Continue with other projects
		}

		// Match workspaces against branch part
		for _, ws := range workspaces {
			if branchPart == "" || s.matchesBranch(branchPart, ws.Branch) {
				distance := s.calculateWorkspaceDistance(projectPart, branchPart, p.String(), ws.Branch)
				results = append(results, &Result{
					Project:   p,
					Workspace: ws.Branch,
					Distance:  distance,
				})

				s.logger.Debug("found matching workspace",
					"project", p.String(),
					"branch", ws.Branch,
					"distance", distance,
				)
			}
		}

		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to walk projects: %w", err)
	}

	return s.sortAndLimitResults(results, opts), nil
}

func (s *Service) matchesProject(query, projectName string) bool {
	queryLower := strings.ToLower(query)

	// Exact match
	if projectName == queryLower {
		return true
	}

	// Fuzzy match
	return fuzzy.MatchFold(queryLower, projectName)
}

func (s *Service) matchesBranch(query, branchName string) bool {
	queryLower := strings.ToLower(query)
	branchLower := strings.ToLower(branchName)

	// Exact match
	if branchLower == queryLower {
		return true
	}

	// Substring match
	if strings.Contains(branchLower, queryLower) {
		return true
	}

	// Fuzzy match
	return fuzzy.MatchFold(queryLower, branchName)
}

func (s *Service) calculateWorkspaceDistance(projectQuery, branchQuery, projectName, branchName string) int {
	distance := 0

	// Project matching distance
	if projectQuery != "" {
		projectLower := strings.ToLower(projectName)
		queryLower := strings.ToLower(projectQuery)

		if projectLower == queryLower {
			distance += 0 // Exact match
		} else if strings.Contains(projectLower, queryLower) {
			distance += 10 // Substring match
		} else {
			distance += 50 + fuzzy.RankMatchFold(projectQuery, projectName) // Fuzzy match
		}
	}

	// Branch matching distance
	if branchQuery != "" {
		branchLower := strings.ToLower(branchName)
		queryLower := strings.ToLower(branchQuery)

		if branchLower == queryLower {
			distance += 0 // Exact match
		} else if strings.Contains(branchLower, queryLower) {
			distance += 5 // Substring match
		} else {
			distance += 20 + fuzzy.RankMatchFold(branchQuery, branchName) // Fuzzy match
		}
	}

	return distance
}

func (s *Service) sortAndLimitResults(results []*Result, opts Options) []*Result {
	// Sort by distance (lower is better), then by project name, then by workspace
	sort.Slice(results, func(i, j int) bool {
		if results[i].Distance == results[j].Distance {
			projectCompare := results[i].Project.String()
			if projectCompare == results[j].Project.String() {
				return results[i].Workspace < results[j].Workspace
			}
			return projectCompare < results[j].Project.String()
		}
		return results[i].Distance < results[j].Distance
	})

	// Apply limit
	if opts.Limit > 0 && opts.Limit < len(results) {
		results = results[:opts.Limit]
	}

	return results
}

// Format formats the search results according to the options.
func (s *Service) Format(results []*Result, opts Options) string {
	if len(results) == 0 {
		return ""
	}

	getPath := func(result *Result) string {
		var path string
		if opts.AbsPath {
			if result.Workspace != "" {
				// For workspace results, return the workspace path
				workspacePath := s.workspaceService.WorkspacePath(*result.Project, result.Workspace)
				path = workspacePath
			} else {
				path = result.Project.Path
			}
		} else {
			if result.Workspace != "" {
				// For workspace results, return project:branch format
				path = result.Project.String() + ":" + result.Workspace
			} else {
				path = result.Project.String()
			}
		}

		if opts.ShowDistance {
			path += fmt.Sprintf(" - %d", result.Distance)
		}

		return path
	}

	var parts []string
	for _, result := range results {
		parts = append(parts, getPath(result))
	}

	return strings.Join(parts, opts.Separator)
}

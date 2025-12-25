package projects

import (
	"context"
	"fmt"
	"io/fs"
	"path/filepath"
	"runtime"
	"sort"
	"strings"

	"github.com/lithammer/fuzzysearch/fuzzy"
)

// pathsEqual compares paths with case-insensitivity on macOS/Windows.
func pathsEqual(a, b string) bool {
	if runtime.GOOS == "darwin" || runtime.GOOS == "windows" {
		return strings.EqualFold(a, b)
	}
	return a == b
}

// QueryService provides project querying functionality.
type QueryService struct {
	logger           Logger
	projectService   *ProjectService
	workspaceService *WorkspaceService
}

// NewQueryService creates a new query service.
func NewQueryService(config *Config, logger Logger) *QueryService {
	projectSvc := NewProjectService(config, logger)
	workspaceSvc := NewWorkspaceService(config, logger)

	return &QueryService{
		logger:           logger,
		projectService:   projectSvc,
		workspaceService: workspaceSvc,
	}
}

// Search searches for projects and workspaces matching the given options.
func (s *QueryService) Search(ctx context.Context, opts SearchOptions) ([]*SearchResult, error) {
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

func (s *QueryService) searchProjects(ctx context.Context, opts SearchOptions, excludeMap map[string]bool) ([]*SearchResult, error) {
	var results []*SearchResult

	qLower := strings.ToLower(opts.Query)
	qOrg, qName, qHasOrg := strings.Cut(qLower, "/")

	err := s.projectService.Walk(func(d fs.DirEntry, p *Project) error {
		// Check if project should be excluded
		if excludeMap[p.Path] {
			s.logger.Debug("excluding project", "path", p.Path)
			return filepath.SkipDir
		}

		if opts.Query == "" {
			results = append(results, &SearchResult{
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
			switch {
			case qLower == pName:
				distance = 1
			case qLower == pOrg:
				distance = 2
			case strings.Contains(pName, qLower):
				distance = 10 + fuzzy.RankMatchFold(qLower, pName)
			case strings.Contains(pOrg, qLower):
				distance = 20 + fuzzy.RankMatchFold(qLower, pOrg)
			default:
				distance = 50 + fuzzy.RankMatchFold(qLower, projectLower)
			}
		}

		results = append(results, &SearchResult{
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

func (s *QueryService) searchWorkspaces(ctx context.Context, opts SearchOptions, excludeMap map[string]bool) ([]*SearchResult, error) {
	var results []*SearchResult

	// Parse workspace query: project_part:branch_part
	projectPart, branchPart, _ := strings.Cut(opts.Query, ":")
	projectPart = strings.TrimSpace(projectPart)
	branchPart = strings.TrimSpace(branchPart)

	s.logger.Debug("searching workspaces", "projectPart", projectPart, "branchPart", branchPart)

	err := s.projectService.Walk(func(d fs.DirEntry, p *Project) error {
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
			if !pathsEqual(p.Path, opts.CurrentProject.Path) {
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
				results = append(results, &SearchResult{
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

func (s *QueryService) matchesProject(query, projectName string) bool {
	queryLower := strings.ToLower(query)

	// Exact match
	if projectName == queryLower {
		return true
	}

	// Fuzzy match
	return fuzzy.MatchFold(queryLower, projectName)
}

func (s *QueryService) matchesBranch(query, branchName string) bool {
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

func (s *QueryService) calculateWorkspaceDistance(projectQuery, branchQuery, projectName, branchName string) int {
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

func (s *QueryService) sortAndLimitResults(results []*SearchResult, opts SearchOptions) []*SearchResult {
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
func (s *QueryService) Format(results []*SearchResult, opts SearchOptions) string {
	if len(results) == 0 {
		return ""
	}

	getPath := func(result *SearchResult) string {
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

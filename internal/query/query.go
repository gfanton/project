package query

import (
	"context"
	"fmt"
	"io/fs"
	"log/slog"
	"path/filepath"
	"sort"
	"strings"

	"github.com/gfanton/project/internal/project"
	"github.com/lithammer/fuzzysearch/fuzzy"
)

// Options holds configuration for project queries.
type Options struct {
	Query     string
	Exclude   []string
	AbsPath   bool
	Separator string
	Limit     int
}

// Result represents a search result.
type Result struct {
	Project  *project.Project
	Distance int
}

// Service provides project querying functionality.
type Service struct {
	logger  *slog.Logger
	rootDir string
}

// NewService creates a new query service.
func NewService(logger *slog.Logger, rootDir string) *Service {
	return &Service{
		logger:  logger,
		rootDir: rootDir,
	}
}

// Search searches for projects matching the given options.
func (s *Service) Search(ctx context.Context, opts Options) ([]*Result, error) {
	s.logger.Debug("searching projects",
		"query", opts.Query,
		"exclude", opts.Exclude,
	)

	var results []*Result
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

	err := project.Walk(s.rootDir, func(d fs.DirEntry, p *project.Project) error {
		// Check if project should be excluded
		if excludeMap[p.Path] {
			s.logger.Debug("excluding project", "path", p.Path)
			return filepath.SkipDir
		}

		// Calculate match distance
		distance := 1
		projectName := p.String()

		if opts.Query == "" {
			return nil
		}

		if distance = fuzzy.RankMatchFold(opts.Query, projectName); distance < 0 {
			return nil // No match
		}

		// Prioritize exact substring matches
		queryLower := strings.ToLower(opts.Query)
		projectLower := strings.ToLower(projectName)

		// Split project name into parts
		parts := strings.Split(projectName, "/")
		projectNamePart := ""
		orgPart := ""
		if len(parts) == 2 {
			orgPart = strings.ToLower(parts[0])
			projectNamePart = strings.ToLower(parts[1])
		}

		// Exact match gets highest priority (distance 0)
		if projectLower == queryLower {
			distance = 0
		} else if projectNamePart == queryLower || orgPart == queryLower {
			// Exact match on project name or org gets very high priority
			distance = -1000
		} else if strings.Contains(projectLower, queryLower) {
			// Substring match gets higher priority than fuzzy match
			// Use negative distance to ensure it ranks higher
			distance = -distance
		}

		results = append(results, &Result{
			Project:  p,
			Distance: distance,
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

	// Sort by distance (lower is better, negative distances rank highest)
	sort.Slice(results, func(i, j int) bool {
		// Negative distances (substring matches) should come first
		if results[i].Distance < 0 && results[j].Distance >= 0 {
			return true
		}
		if results[i].Distance >= 0 && results[j].Distance < 0 {
			return false
		}
		// For both negative or both positive, sort normally
		return results[i].Distance < results[j].Distance
	})

	// Apply limit
	if opts.Limit > 0 && opts.Limit < len(results) {
		results = results[:opts.Limit]
	}

	return results, nil
}

// Format formats the search results according to the options.
func (s *Service) Format(results []*Result, opts Options) string {
	if len(results) == 0 {
		return ""
	}

	getPath := func(p *project.Project) string {
		if opts.AbsPath {
			return p.Path
		}
		return p.String()
	}

	var parts []string
	for _, result := range results {
		parts = append(parts, getPath(result.Project))
	}

	return strings.Join(parts, opts.Separator)
}

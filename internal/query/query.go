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
	Query        string
	Exclude      []string
	AbsPath      bool
	Separator    string
	Limit        int
	ShowDistance bool
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
				Project:  p,
				Distance: 1,
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
				// continue with initial distance
			}
		}

		results = append(results, &Result{
			Project:  p,
			Distance: distance,
		})

		s.logger.Debug("found matching project",
			"name", projectName,
			"distance", distance,
		)

		// // parts := strings.Split(projectName, "/")
		// // projectNamePart := ""
		// // orgPart := ""
		// // if len(parts) == 2 {
		// // 	orgPart = strings.ToLower(parts[0])
		// // 	projectNamePart = strings.ToLower(parts[1])
		// // }

		// // Priority ranking (lower distance = higher priority):

		// // 1. Exact full match: "foobar/foo" matches "foobar/foo"
		// if projectLower == queryLower {
		// 	distance = 0
		// 	// 2. Exact project name match: "foo" matches "foobar/foo"
		// } else if projectNamePart == queryLower {
		// 	distance = 1
		// 	// 3. Exact org match: "foobar" matches "foobar/foo"
		// } else if orgPart == queryLower {
		// 	distance = 2
		// 	// 4. Project name substring: "foo" in "foo-by-example" (prioritized over full substring)
		// } else if strings.Contains(projectNamePart, queryLower) {
		// 	distance = 5 + len(projectNamePart) - len(opts.Query) // Shorter project names rank higher
		// 	// 5. Org substring: "foo" in "foobar"
		// } else if strings.Contains(orgPart, queryLower) {
		// 	distance = 50 + len(orgPart) - len(opts.Query) // Shorter org names rank higher
		// 	// 6. Full substring match: "foo" in "foobar/foo-test" (when not in project name)
		// } else if strings.Contains(projectLower, queryLower) {
		// 	distance = 100 + len(projectName) - len(opts.Query) // Shorter strings rank higher
		// 	// 7. Fuzzy match as fallback
		// } else {
		// 	fuzzyDistance := fuzzy.RankMatchFold(opts.Query, projectName)
		// 	if fuzzyDistance < 0 {
		// 		return nil // No match
		// 	}
		// 	distance = 1000 + fuzzyDistance // Low priority for fuzzy matches
		// }

		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to walk projects: %w", err)
	}

	// Sort by distance (lower is better), then alphabetically by project name
	sort.Slice(results, func(i, j int) bool {
		if results[i].Distance == results[j].Distance {
			return results[i].Project.String() < results[j].Project.String()
		}
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
		path := getPath(result.Project)
		if opts.ShowDistance {
			path += fmt.Sprintf(" - %d", result.Distance)
		}
		parts = append(parts, path)
	}

	return strings.Join(parts, opts.Separator)
}

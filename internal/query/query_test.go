package query

import (
	"context"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gfanton/project/internal/project"
	"github.com/go-git/go-git/v5"
)

func setupTestProjects(t *testing.T) (string, func()) {
	// Create temporary directory structure for testing
	tempDir, err := os.MkdirTemp("", "query-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}

	// Create test project structure
	testProjects := []struct {
		path string
		git  bool
	}{
		{"user1/webapp", true},
		{"user1/mobile-app", true},
		{"user2/backend", true},
		{"user2/frontend", false},
		{"org/awesome-project", true},
		{"org/test-app", false},
		{"alice/my-blog", true},
		{"bob/game-engine", true},
	}

	for _, p := range testProjects {
		projectPath := filepath.Join(tempDir, p.path)
		err := os.MkdirAll(projectPath, 0755)
		if err != nil {
			t.Fatalf("Failed to create project directory %s: %v", projectPath, err)
		}

		if p.git {
			_, err := git.PlainInit(projectPath, false)
			if err != nil {
				t.Fatalf("Failed to init git repo in %s: %v", projectPath, err)
			}
		}
	}

	cleanup := func() {
		os.RemoveAll(tempDir)
	}

	return tempDir, cleanup
}

func TestNewService(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	rootDir := "/test/root"

	service := NewService(logger, rootDir)

	if service == nil {
		t.Fatal("NewService() returned nil")
	}

	if service.logger == nil {
		t.Error("NewService() should set logger")
	}

	if service.rootDir != rootDir {
		t.Errorf("NewService() rootDir = %s, want %s", service.rootDir, rootDir)
	}
}

func TestSearch(t *testing.T) {
	rootDir, cleanup := setupTestProjects(t)
	defer cleanup()

	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	service := NewService(logger, rootDir)

	tests := []struct {
		name           string
		opts           Options
		expectedCount  int
		expectedFirst  string
		shouldContain  []string
		shouldExclude  []string
	}{
		{
			name: "search for 'app'",
			opts: Options{
				Query: "app",
				Limit: 0,
			},
			expectedCount: 3, // webapp, mobile-app, test-app (direct matches)
			shouldContain: []string{"user1/webapp", "user1/mobile-app", "org/test-app"},
		},
		{
			name: "search for 'web'",
			opts: Options{
				Query: "web",
				Limit: 1,
			},
			expectedCount: 1,
			expectedFirst: "user1/webapp",
		},
		{
			name: "search for 'backend'",
			opts: Options{
				Query: "backend",
				Limit: 0,
			},
			expectedCount: 1,
			expectedFirst: "user2/backend",
		},
		{
			name: "search with no query (all projects)",
			opts: Options{
				Query: "",
				Limit: 0,
			},
			expectedCount: 8, // All projects
		},
		{
			name: "search with limit",
			opts: Options{
				Query: "",
				Limit: 3,
			},
			expectedCount: 3,
		},
		{
			name: "search with exclusion",
			opts: Options{
				Query:   "app",
				Exclude: []string{filepath.Join(rootDir, "user1/webapp")},
				Limit:   0,
			},
			expectedCount: 2, // Should exclude webapp, leaving mobile-app and test-app
			shouldExclude: []string{"user1/webapp"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			results, err := service.Search(ctx, tt.opts)

			if err != nil {
				t.Fatalf("Search() failed: %v", err)
			}

			if len(results) != tt.expectedCount {
				t.Errorf("Search() returned %d results, want %d", len(results), tt.expectedCount)
			}

			if tt.expectedFirst != "" && len(results) > 0 {
				first := results[0].Project.String()
				if first != tt.expectedFirst {
					t.Errorf("Search() first result = %s, want %s", first, tt.expectedFirst)
				}
			}

			// Check that expected projects are included
			resultNames := make(map[string]bool)
			for _, result := range results {
				resultNames[result.Project.String()] = true
			}

			for _, expected := range tt.shouldContain {
				if !resultNames[expected] {
					t.Errorf("Search() should contain %s but didn't", expected)
				}
			}

			for _, excluded := range tt.shouldExclude {
				if resultNames[excluded] {
					t.Errorf("Search() should exclude %s but didn't", excluded)
				}
			}
		})
	}
}

func TestSearchWithValidExcludePath(t *testing.T) {
	rootDir, cleanup := setupTestProjects(t)
	defer cleanup()

	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	service := NewService(logger, rootDir)

	ctx := context.Background()
	
	// Test with a valid but non-existent exclude path (should work fine)
	opts := Options{
		Query:   "app",
		Exclude: []string{"/non/existent/path"},
	}

	results, err := service.Search(ctx, opts)
	if err != nil {
		t.Fatalf("Search() failed with valid exclude path: %v", err)
	}

	// Should find all 'app' projects since the exclude path doesn't match anything
	if len(results) != 3 {
		t.Errorf("Search() with non-matching exclude should return 3 results, got %d", len(results))
	}
}

func TestSearchWithNonExistentRoot(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	service := NewService(logger, "/non/existent/path")

	ctx := context.Background()
	opts := Options{
		Query: "test",
	}

	_, err := service.Search(ctx, opts)
	if err == nil {
		t.Error("Search() should fail with non-existent root directory")
	}

	if !strings.Contains(err.Error(), "failed to walk projects") {
		t.Errorf("Error should mention walk failure, got: %v", err)
	}
}

func TestSearchResultSorting(t *testing.T) {
	rootDir, cleanup := setupTestProjects(t)
	defer cleanup()

	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	service := NewService(logger, rootDir)

	ctx := context.Background()
	opts := Options{
		Query: "app",
	}

	results, err := service.Search(ctx, opts)
	if err != nil {
		t.Fatalf("Search() failed: %v", err)
	}

	if len(results) < 2 {
		t.Skip("Need at least 2 results to test sorting")
	}

	// Verify that results are sorted by distance (lower distance = better match)
	for i := 1; i < len(results); i++ {
		if results[i-1].Distance > results[i].Distance {
			t.Errorf("Results not sorted by distance: result[%d].Distance=%d > result[%d].Distance=%d",
				i-1, results[i-1].Distance, i, results[i].Distance)
		}
	}
}

func TestFormat(t *testing.T) {
	// Create mock projects for testing formatting
	projects := []*Result{
		{
			Project: &project.Project{
				Path:         "/root/user1/webapp",
				Name:         "webapp",
				Organisation: "user1",
			},
			Distance: 1,
		},
		{
			Project: &project.Project{
				Path:         "/root/user2/backend",
				Name:         "backend",
				Organisation: "user2",
			},
			Distance: 2,
		},
	}

	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	service := NewService(logger, "/root")

	tests := []struct {
		name     string
		opts     Options
		expected string
	}{
		{
			name: "default format with newline separator",
			opts: Options{
				Separator: "\n",
				AbsPath:   false,
			},
			expected: "user1/webapp\nuser2/backend",
		},
		{
			name: "absolute path format",
			opts: Options{
				Separator: "\n",
				AbsPath:   true,
			},
			expected: "/root/user1/webapp\n/root/user2/backend",
		},
		{
			name: "custom separator",
			opts: Options{
				Separator: " | ",
				AbsPath:   false,
			},
			expected: "user1/webapp | user2/backend",
		},
		{
			name: "comma separated",
			opts: Options{
				Separator: ",",
				AbsPath:   false,
			},
			expected: "user1/webapp,user2/backend",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := service.Format(projects, tt.opts)
			if result != tt.expected {
				t.Errorf("Format() = %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestFormatEmpty(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	service := NewService(logger, "/root")

	opts := Options{
		Separator: "\n",
		AbsPath:   false,
	}

	result := service.Format([]*Result{}, opts)
	if result != "" {
		t.Errorf("Format() with empty results should return empty string, got %q", result)
	}
}

func TestSearchFuzzyMatching(t *testing.T) {
	rootDir, cleanup := setupTestProjects(t)
	defer cleanup()

	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	service := NewService(logger, rootDir)

	tests := []struct {
		name          string
		query         string
		shouldMatch   []string
		shouldNotFind []string
	}{
		{
			name:        "exact match",
			query:       "webapp",
			shouldMatch: []string{"user1/webapp"},
		},
		{
			name:        "partial match",
			query:       "web",
			shouldMatch: []string{"user1/webapp"},
		},
		{
			name:        "hyphenated project",
			query:       "mobile",
			shouldMatch: []string{"user1/mobile-app"},
		},
		{
			name:        "organization match",
			query:       "alice",
			shouldMatch: []string{"alice/my-blog"},
		},
		{
			name:          "no match",
			query:         "nonexistent",
			shouldNotFind: []string{"user1/webapp", "user2/backend"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			opts := Options{
				Query: tt.query,
			}

			results, err := service.Search(ctx, opts)
			if err != nil {
				t.Fatalf("Search() failed: %v", err)
			}

			resultNames := make(map[string]bool)
			for _, result := range results {
				resultNames[result.Project.String()] = true
			}

			for _, expected := range tt.shouldMatch {
				if !resultNames[expected] {
					t.Errorf("Query '%s' should match %s", tt.query, expected)
				}
			}

			for _, notExpected := range tt.shouldNotFind {
				if resultNames[notExpected] {
					t.Errorf("Query '%s' should not match %s", tt.query, notExpected)
				}
			}
		})
	}
}

func setupRankingTestProjects(t *testing.T) (string, func()) {
	// Create temporary directory structure for ranking tests
	tempDir, err := os.MkdirTemp("", "ranking-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}

	// Create test project structure using foo/foobar pattern
	testProjects := []struct {
		path string
		git  bool
	}{
		// Exact matches
		{"foobar/foo", true},
		{"foo/bar", true},
		{"foobar/baz", true},
		
		// Substring matches
		{"foobar/foo-by-example", true},
		{"foobar/foo-test", true},
		{"foobar/awesome-foo", true},
		{"foo/foo-lib", true},
		{"otherfoo/bar", true},
		
		// Fuzzy matches
		{"foobar/project", true},
		{"company/fooish", true},
		{"dev/foobaz", true},
		
		// Non-matches that shouldn't appear
		{"bar/baz", true},
		{"company/project", true},
	}

	for _, p := range testProjects {
		projectPath := filepath.Join(tempDir, p.path)
		err := os.MkdirAll(projectPath, 0755)
		if err != nil {
			t.Fatalf("Failed to create project directory %s: %v", projectPath, err)
		}

		if p.git {
			_, err := git.PlainInit(projectPath, false)
			if err != nil {
				t.Fatalf("Failed to init git repo in %s: %v", projectPath, err)
			}
		}
	}

	cleanup := func() {
		os.RemoveAll(tempDir)
	}

	return tempDir, cleanup
}

func TestSearchRankingAlgorithm(t *testing.T) {
	rootDir, cleanup := setupRankingTestProjects(t)
	defer cleanup()

	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	service := NewService(logger, rootDir)

	tests := []struct {
		name           string
		query          string
		expectedOrder  []string // Expected order of results (first = highest priority)
		expectedFirst  string   // Must be the first result
		minResults     int      // Minimum number of results expected
	}{
		{
			name:  "exact full match takes priority",
			query: "foobar/foo",
			expectedOrder: []string{
				"foobar/foo",           // distance: 0 (exact full match)
				"foobar/foo-by-example", // distance: 100+ (project name substring)
				"foobar/foo-test",      // distance: 100+ (project name substring)
			},
			expectedFirst: "foobar/foo",
			minResults:    3,
		},
		{
			name:  "exact project name match",
			query: "foo",
			expectedOrder: []string{
				"foobar/foo",           // distance: 1 (exact project name)
				"foo/bar",              // distance: 2 (exact org name) 
				"foo/foo-lib",          // distance: 2 (exact org name)
				"foobar/foo-test",      // distance: 5+8-3=10 (project name substring)
				"foobar/awesome-foo",   // distance: 5+11-3=13 (project name substring)
				"foobar/foo-by-example", // distance: 5+14-3=16 (project name substring)
			},
			expectedFirst: "foobar/foo", // Should prioritize exact project name over org name
			minResults:    6,
		},
		{
			name:  "exact org match",
			query: "foobar",
			expectedOrder: []string{
				"foobar/awesome-foo",   // distance: 2 (exact org match) - alphabetically first
				"foobar/baz",           // distance: 2 (exact org match)
				"foobar/foo",           // distance: 2 (exact org match)
				"foobar/foo-by-example", // distance: 2 (exact org match)
				"foobar/foo-test",      // distance: 2 (exact org match)
				"foobar/project",       // distance: 2 (exact org match)
			},
			expectedFirst: "foobar/awesome-foo", // First alphabetically among same distance
			minResults:    6,
		},
		{
			name:  "substring match prioritizes shorter strings",
			query: "foo",
			expectedOrder: []string{
				"foobar/foo",      // distance: 1 (exact project name - highest priority)
				"foo/bar",         // distance: 2 (exact org name)
				"foo/foo-lib",     // distance: 2 (exact org name)
			},
			expectedFirst: "foobar/foo",
			minResults:    3,
		},
		{
			name:  "project name substring vs org substring",
			query: "foo",
			// Project name substring should rank higher than org substring
			expectedOrder: []string{
				"foobar/foo",           // distance: 1 (exact project name)
				"foo/bar",              // distance: 2 (exact org)
				"foo/foo-lib",          // distance: 2 (exact org)
				"foobar/foo-test",      // distance: 5+8-3=10 (project name substring)
				"foobar/awesome-foo",   // distance: 5+11-3=13 (project name substring)
				"foobar/foo-by-example", // distance: 5+14-3=16 (project name substring)
				"otherfoo/bar",         // distance: 50+8-3=55 (org substring)
			},
			expectedFirst: "foobar/foo",
			minResults:    7,
		},
		{
			name:  "single character should work",
			query: "f",
			// Should match projects containing 'f'
			minResults: 8, // Most projects should match
		},
		{
			name:  "case insensitive matching",
			query: "FOO",
			expectedOrder: []string{
				"foobar/foo",    // Should match despite case difference
				"foo/bar",
				"foo/foo-lib",
			},
			expectedFirst: "foobar/foo",
			minResults:    3,
		},
		{
			name:       "no match should return empty",
			query:      "nonexistent",
			minResults: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			opts := Options{
				Query: tt.query,
				Limit: 0, // Get all results
			}

			results, err := service.Search(ctx, opts)
			if err != nil {
				t.Fatalf("Search() failed: %v", err)
			}

			// Check minimum results
			if len(results) < tt.minResults {
				t.Errorf("Search() returned %d results, want at least %d", len(results), tt.minResults)
			}

			// Check first result if specified
			if tt.expectedFirst != "" && len(results) > 0 {
				first := results[0].Project.String()
				if first != tt.expectedFirst {
					t.Errorf("Search() first result = %s, want %s", first, tt.expectedFirst)
					// Print all results for debugging
					t.Logf("All results for query '%s':", tt.query)
					for i, result := range results {
						t.Logf("  %d: %s (distance: %d)", i, result.Project.String(), result.Distance)
					}
				}
			}

			// Check expected order for the first N results
			if len(tt.expectedOrder) > 0 {
				for i, expected := range tt.expectedOrder {
					if i >= len(results) {
						break // Not enough results to check this position
					}
					actual := results[i].Project.String()
					if actual != expected {
						t.Errorf("Search() result[%d] = %s, want %s", i, actual, expected)
						// Print all results for debugging
						t.Logf("All results for query '%s':", tt.query)
						for j, result := range results {
							t.Logf("  %d: %s (distance: %d)", j, result.Project.String(), result.Distance)
						}
						break
					}
				}
			}

			// Verify results are sorted by distance
			for i := 1; i < len(results); i++ {
				if results[i-1].Distance > results[i].Distance {
					t.Errorf("Results not sorted by distance: result[%d].Distance=%d > result[%d].Distance=%d",
						i-1, results[i-1].Distance, i, results[i].Distance)
					// Print results for debugging
					t.Logf("Results for query '%s':", tt.query)
					for j, result := range results {
						t.Logf("  %d: %s (distance: %d)", j, result.Project.String(), result.Distance)
					}
					break
				}
			}
		})
	}
}
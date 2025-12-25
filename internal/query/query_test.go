package query

import (
	"context"
	"log/slog"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/gfanton/projects/internal/project"
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
		name          string
		opts          Options
		expectedCount int
		expectedFirst string
		shouldContain []string
		shouldExclude []string
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
		name          string
		query         string
		expectedOrder []string // Expected order of results (first = highest priority)
		expectedFirst string   // Must be the first result
		minResults    int      // Minimum number of results expected
	}{
		{
			name:  "exact full match takes priority",
			query: "foobar/foo",
			expectedOrder: []string{
				"foobar/foo",            // distance: 0 (exact full match)
				"foobar/foo-test",       // distance: 5 (fuzzy match on name part)
				"foobar/awesome-foo",    // distance: 8 (fuzzy match on name part)
				"foobar/foo-by-example", // distance: 11 (fuzzy match on name part)
			},
			expectedFirst: "foobar/foo",
			minResults:    4,
		},
		{
			name:  "exact project name match",
			query: "foo",
			// With new algorithm: exact project name match (1) > exact org match (2) > substring
			expectedOrder: []string{
				"foobar/foo",  // distance: 1 (exact project name match)
				"foo/bar",     // distance: 2 (exact org match)
				"foo/foo-lib", // distance: 2 (exact org match)
			},
			expectedFirst: "foobar/foo",
			minResults:    3,
		},
		{
			name:  "exact org match",
			query: "foobar",
			// All foobar/* projects get distance=2 (exact org match), sorted alphabetically
			expectedOrder: []string{
				"foobar/awesome-foo",
				"foobar/baz",
				"foobar/foo",
				"foobar/foo-by-example",
				"foobar/foo-test",
				"foobar/project",
			},
			expectedFirst: "foobar/awesome-foo", // First alphabetically among same distance
			minResults:    6,
		},
		{
			name:  "exact project name match takes priority",
			query: "foo",
			// "foobar/foo" has exact project name match (distance=1)
			// "foo/bar" and "foo/foo-lib" have exact org match (distance=2)
			expectedOrder: []string{
				"foobar/foo",  // distance: 1 (exact project name match)
				"foo/bar",     // distance: 2 (exact org match, alphabetically first)
				"foo/foo-lib", // distance: 2 (exact org match)
			},
			expectedFirst: "foobar/foo",
			minResults:    3,
		},
		{
			name:  "project name substring vs org substring",
			query: "foo",
			// Exact project name match > exact org match > substring in name > substring in org
			expectedOrder: []string{
				"foobar/foo",  // distance: 1 (exact project name match)
				"foo/bar",     // distance: 2 (exact org match)
				"foo/foo-lib", // distance: 2 (exact org match)
			},
			expectedFirst: "foobar/foo",
			minResults:    3,
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
			// Case insensitive: exact project name match > exact org match
			expectedOrder: []string{
				"foobar/foo",  // Exact project name match (case insensitive)
				"foo/bar",     // Exact org match (case insensitive)
				"foo/foo-lib", // Exact org match (case insensitive)
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

func TestWorkspaceQuerying(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	tempDir, cleanup := setupTestProjects(t)
	defer cleanup()

	// Create test workspaces for user1/webapp project
	webappPath := filepath.Join(tempDir, "user1", "webapp")

	// Create workspace service and add some test workspaces
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelWarn}))
	service := NewService(logger, tempDir)
	ctx := context.Background()

	// Create a project instance
	webappProject := &project.Project{
		Path:         webappPath,
		Name:         "webapp",
		Organisation: "user1",
	}

	// Add test workspaces using the workspace service directly
	err := service.workspaceService.Add(ctx, *webappProject, "feature-auth")
	if err != nil {
		t.Fatalf("Failed to add workspace: %v", err)
	}

	err = service.workspaceService.Add(ctx, *webappProject, "dev-branch")
	if err != nil {
		t.Fatalf("Failed to add workspace: %v", err)
	}

	err = service.workspaceService.Add(ctx, *webappProject, "bugfix-123")
	if err != nil {
		t.Fatalf("Failed to add workspace: %v", err)
	}

	tests := []struct {
		name     string
		query    string
		expected []string // Expected results in format "project:workspace"
		minCount int
	}{
		{
			name:     "Query specific project workspace",
			query:    "user1/webapp:feature",
			expected: []string{"user1/webapp:feature-auth"},
			minCount: 1,
		},
		{
			name:     "Query workspace by branch name only",
			query:    ":dev",
			expected: []string{"user1/webapp:dev-branch"},
			minCount: 1,
		},
		{
			name:     "Query all workspaces for project",
			query:    "webapp:",
			expected: []string{"user1/webapp:bugfix-123", "user1/webapp:dev-branch", "user1/webapp:feature-auth"},
			minCount: 3,
		},
		{
			name:     "Query workspaces with partial project match",
			query:    "user1:",
			expected: []string{"user1/webapp:bugfix-123", "user1/webapp:dev-branch", "user1/webapp:feature-auth"},
			minCount: 3,
		},
		{
			name:     "Query workspace substring match",
			query:    ":bug",
			expected: []string{"user1/webapp:bugfix-123"},
			minCount: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			results, err := service.Search(ctx, Options{
				Query:     tt.query,
				Separator: "\n",
			})

			if err != nil {
				t.Fatalf("Search() error = %v", err)
			}

			if len(results) < tt.minCount {
				t.Errorf("Search() returned %d results, want at least %d", len(results), tt.minCount)
				t.Logf("Results:")
				for i, result := range results {
					workspace := result.Workspace
					if workspace == "" {
						t.Logf("  %d: %s (project)", i, result.Project.String())
					} else {
						t.Logf("  %d: %s:%s (workspace)", i, result.Project.String(), workspace)
					}
				}
				return
			}

			// Check that all results are workspace results (have workspace set)
			for _, result := range results {
				if result.Workspace == "" {
					t.Errorf("Expected workspace result but got project result: %s", result.Project.String())
				}
			}

			// Check expected results
			if len(tt.expected) > 0 {
				actualResults := make([]string, len(results))
				for i, result := range results {
					actualResults[i] = result.Project.String() + ":" + result.Workspace
				}

				for i, expected := range tt.expected {
					if i >= len(actualResults) {
						t.Errorf("Missing expected result: %s", expected)
						continue
					}
					if actualResults[i] != expected {
						t.Errorf("Result[%d] = %s, want %s", i, actualResults[i], expected)
					}
				}
			}
		})
	}

	// Clean up workspaces
	_ = service.workspaceService.Remove(ctx, *webappProject, "feature-auth", false)
	_ = service.workspaceService.Remove(ctx, *webappProject, "dev-branch", false)
	_ = service.workspaceService.Remove(ctx, *webappProject, "bugfix-123", false)
}

func TestQueryExcludesDotDirectories(t *testing.T) {
	// Create temporary directory structure for testing
	tempDir, err := os.MkdirTemp("", "query-dot-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create test directory structure including dot directories
	testStructure := []struct {
		path string
		git  bool
	}{
		{"user1/normal-project", true},
		{"user2/another-project", true},
		{".workspace/user1/project.feature", false}, // Should be excluded from project search
		{".vscode/settings", false},                 // Should be excluded
		{".git/hooks", false},                       // Should be excluded
		{"user1/.hidden-project", false},            // Should be excluded
	}

	for _, item := range testStructure {
		projectPath := filepath.Join(tempDir, item.path)
		err := os.MkdirAll(projectPath, 0755)
		if err != nil {
			t.Fatalf("Failed to create project directory %s: %v", projectPath, err)
		}

		if item.git {
			_, err := git.PlainInit(projectPath, false)
			if err != nil {
				t.Fatalf("Failed to init git repo in %s: %v", projectPath, err)
			}
		}
	}

	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	service := NewService(logger, tempDir)

	ctx := context.Background()
	// Search for all projects (empty query)
	results, err := service.Search(ctx, Options{
		Query: "",
		Limit: 0,
	})

	if err != nil {
		t.Fatalf("Search() failed: %v", err)
	}

	// Should only find normal projects, not those in dot directories
	expectedCount := 2 // user1/normal-project and user2/another-project
	if len(results) != expectedCount {
		t.Errorf("Expected %d projects, found %d", expectedCount, len(results))
		for i, result := range results {
			t.Logf("Found project %d: %s", i, result.Project.String())
		}
	}

	// Verify no results contain dot directories
	for _, result := range results {
		projectName := result.Project.String()
		if strings.Contains(projectName, "/.") || strings.HasPrefix(projectName, ".") {
			t.Errorf("Found project in dot directory: %s", projectName)
		}

		// Ensure the results are the expected normal projects
		if projectName != "user1/normal-project" && projectName != "user2/another-project" {
			t.Errorf("Unexpected project found: %s", projectName)
		}
	}
}

func TestWorkspaceFormat(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelWarn}))
	tempDir := t.TempDir()
	service := NewService(logger, tempDir)

	testProject := &project.Project{
		Path:         "/test/user1/webapp",
		Name:         "webapp",
		Organisation: "user1",
	}

	tests := []struct {
		name     string
		results  []*Result
		opts     Options
		expected string
	}{
		{
			name: "Format workspace results",
			results: []*Result{
				{Project: testProject, Workspace: "feature-auth", Distance: 0},
				{Project: testProject, Workspace: "dev-branch", Distance: 5},
			},
			opts:     Options{Separator: "\n"},
			expected: "user1/webapp:feature-auth\nuser1/webapp:dev-branch",
		},
		{
			name: "Format mixed results",
			results: []*Result{
				{Project: testProject, Workspace: "", Distance: 0},
				{Project: testProject, Workspace: "feature-auth", Distance: 5},
			},
			opts:     Options{Separator: "\n"},
			expected: "user1/webapp\nuser1/webapp:feature-auth",
		},
		{
			name: "Format with distance",
			results: []*Result{
				{Project: testProject, Workspace: "feature-auth", Distance: 10},
			},
			opts:     Options{Separator: "\n", ShowDistance: true},
			expected: "user1/webapp:feature-auth - 10",
		},
		{
			name: "Format absolute paths",
			results: []*Result{
				{Project: testProject, Workspace: "feature-auth", Distance: 0},
			},
			opts:     Options{Separator: "\n", AbsPath: true},
			expected: tempDir + "/.workspace/user1/webapp.feature-auth",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actual := service.Format(tt.results, tt.opts)
			if actual != tt.expected {
				t.Errorf("Format() = %q, want %q", actual, tt.expected)
			}
		})
	}
}

func TestWorkspaceQueryWithCurrentProject(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	tempDir, cleanup := setupTestProjects(t)
	defer cleanup()

	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelWarn}))
	service := NewService(logger, tempDir)
	ctx := context.Background()

	// Create projects
	webappProject := &project.Project{
		Path:         filepath.Join(tempDir, "user1", "webapp"),
		Name:         "webapp",
		Organisation: "user1",
	}

	mobileAppProject := &project.Project{
		Path:         filepath.Join(tempDir, "user1", "mobile-app"),
		Name:         "mobile-app",
		Organisation: "user1",
	}

	backendProject := &project.Project{
		Path:         filepath.Join(tempDir, "user2", "backend"),
		Name:         "backend",
		Organisation: "user2",
	}

	// Add workspaces to different projects
	// webapp workspaces
	_ = service.workspaceService.Add(ctx, *webappProject, "feature-branch")
	_ = service.workspaceService.Add(ctx, *webappProject, "dev-workspace")
	_ = service.workspaceService.Add(ctx, *webappProject, "main")

	// mobile-app workspaces
	_ = service.workspaceService.Add(ctx, *mobileAppProject, "feature-branch")
	_ = service.workspaceService.Add(ctx, *mobileAppProject, "prod-workspace")

	// backend workspaces
	_ = service.workspaceService.Add(ctx, *backendProject, "feature-branch")
	_ = service.workspaceService.Add(ctx, *backendProject, "staging")

	tests := []struct {
		name           string
		query          string
		currentProject *project.Project
		expected       []string // Expected results in format "project:workspace"
		description    string
	}{
		{
			name:           "Workspace query without project prefix from webapp",
			query:          ":feature",
			currentProject: webappProject,
			expected:       []string{"user1/webapp:feature-branch"},
			description:    "Should only find feature-branch in current project (webapp)",
		},
		{
			name:           "Workspace query without project prefix from mobile-app",
			query:          ":feature",
			currentProject: mobileAppProject,
			expected:       []string{"user1/mobile-app:feature-branch"},
			description:    "Should only find feature-branch in current project (mobile-app)",
		},
		{
			name:           "Workspace query without project prefix from backend",
			query:          ":feature",
			currentProject: backendProject,
			expected:       []string{"user2/backend:feature-branch"},
			description:    "Should only find feature-branch in current project (backend)",
		},
		{
			name:           "Workspace query without current project context",
			query:          ":feature",
			currentProject: nil,
			expected:       []string{"user1/mobile-app:feature-branch", "user1/webapp:feature-branch", "user2/backend:feature-branch"},
			description:    "Should find feature-branch across all projects when no current project",
		},
		{
			name:           "Explicit project overrides current project",
			query:          "user2/backend:feature",
			currentProject: webappProject,
			expected:       []string{"user2/backend:feature-branch"},
			description:    "Explicit project in query should override current project context",
		},
		{
			name:           "List all workspaces in current project",
			query:          ":",
			currentProject: webappProject,
			expected:       []string{"user1/webapp:dev-workspace", "user1/webapp:feature-branch", "user1/webapp:main"},
			description:    "Empty branch query should list all workspaces in current project",
		},
		{
			name:           "Workspace not in current project",
			query:          ":staging",
			currentProject: webappProject,
			expected:       []string{},
			description:    "Should not find workspaces that don't exist in current project",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			results, err := service.Search(ctx, Options{
				Query:          tt.query,
				CurrentProject: tt.currentProject,
				Separator:      "\n",
			})

			if err != nil {
				t.Fatalf("Search() error = %v", err)
			}

			// Format results for comparison
			actualResults := make([]string, len(results))
			for i, result := range results {
				if result.Workspace != "" {
					actualResults[i] = result.Project.String() + ":" + result.Workspace
				} else {
					actualResults[i] = result.Project.String()
				}
			}

			// Sort both expected and actual for comparison
			sort.Strings(tt.expected)
			sort.Strings(actualResults)

			if len(actualResults) != len(tt.expected) {
				t.Errorf("%s\nExpected %d results, got %d\nExpected: %v\nActual: %v",
					tt.description, len(tt.expected), len(actualResults), tt.expected, actualResults)
				return
			}

			for i, expected := range tt.expected {
				if actualResults[i] != expected {
					t.Errorf("%s\nResult[%d] = %s, want %s",
						tt.description, i, actualResults[i], expected)
				}
			}
		})
	}

	// Clean up workspaces
	_ = service.workspaceService.Remove(ctx, *webappProject, "feature-branch", false)
	_ = service.workspaceService.Remove(ctx, *webappProject, "dev-workspace", false)
	_ = service.workspaceService.Remove(ctx, *webappProject, "main", false)
	_ = service.workspaceService.Remove(ctx, *mobileAppProject, "feature-branch", false)
	_ = service.workspaceService.Remove(ctx, *mobileAppProject, "prod-workspace", false)
	_ = service.workspaceService.Remove(ctx, *backendProject, "feature-branch", false)
	_ = service.workspaceService.Remove(ctx, *backendProject, "staging", false)
}

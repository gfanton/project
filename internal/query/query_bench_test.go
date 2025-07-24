package query

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"testing"

	"github.com/go-git/go-git/v5"
	"projects/internal/project"
)

func setupBenchmarkProjects(b *testing.B, numProjects int) (string, func()) {
	// Create temporary directory structure for benchmarking
	tempDir, err := os.MkdirTemp("", "query-bench-*")
	if err != nil {
		b.Fatalf("Failed to create temp dir: %v", err)
	}

	// Create many test projects for benchmarking
	for i := 0; i < numProjects; i++ {
		org := fmt.Sprintf("org%d", i/10)
		project := fmt.Sprintf("project%d", i)
		projectPath := filepath.Join(tempDir, org, project)

		err := os.MkdirAll(projectPath, 0755)
		if err != nil {
			b.Fatalf("Failed to create project directory %s: %v", projectPath, err)
		}

		// Make some of them Git repositories for realism
		if i%3 == 0 {
			_, err := git.PlainInit(projectPath, false)
			if err != nil {
				b.Fatalf("Failed to init git repo in %s: %v", projectPath, err)
			}
		}
	}

	cleanup := func() {
		os.RemoveAll(tempDir)
	}

	return tempDir, cleanup
}

func BenchmarkSearchEmpty(b *testing.B) {
	rootDir, cleanup := setupBenchmarkProjects(b, 100)
	defer cleanup()

	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	service := NewService(logger, rootDir)
	ctx := context.Background()

	opts := Options{
		Query: "",
		Limit: 0,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := service.Search(ctx, opts)
		if err != nil {
			b.Fatalf("Search failed: %v", err)
		}
	}
}

func BenchmarkSearchWithQuery(b *testing.B) {
	rootDir, cleanup := setupBenchmarkProjects(b, 100)
	defer cleanup()

	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	service := NewService(logger, rootDir)
	ctx := context.Background()

	opts := Options{
		Query: "project",
		Limit: 0,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := service.Search(ctx, opts)
		if err != nil {
			b.Fatalf("Search failed: %v", err)
		}
	}
}

func BenchmarkSearchWithQueryAndLimit(b *testing.B) {
	rootDir, cleanup := setupBenchmarkProjects(b, 100)
	defer cleanup()

	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	service := NewService(logger, rootDir)
	ctx := context.Background()

	opts := Options{
		Query: "project",
		Limit: 10,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := service.Search(ctx, opts)
		if err != nil {
			b.Fatalf("Search failed: %v", err)
		}
	}
}

func BenchmarkSearchLargeDataset(b *testing.B) {
	rootDir, cleanup := setupBenchmarkProjects(b, 1000)
	defer cleanup()

	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	service := NewService(logger, rootDir)
	ctx := context.Background()

	opts := Options{
		Query: "proj",
		Limit: 5,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := service.Search(ctx, opts)
		if err != nil {
			b.Fatalf("Search failed: %v", err)
		}
	}
}

func BenchmarkFormat(b *testing.B) {
	// Create mock results for benchmarking format performance
	results := make([]*Result, 100)
	for i := 0; i < 100; i++ {
		results[i] = &Result{
			Project: &project.Project{
				Path:         "/root/user/project" + string(rune('0'+(i%10))),
				Name:         "project" + string(rune('0'+(i%10))),
				Organisation: "user" + string(rune('a'+(i%26))),
			},
			Distance: i,
		}
	}

	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	service := NewService(logger, "/root")

	opts := Options{
		Separator: "\n",
		AbsPath:   false,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = service.Format(results, opts)
	}
}

func BenchmarkFormatAbsPath(b *testing.B) {
	// Create mock results for benchmarking format performance
	results := make([]*Result, 100)
	for i := 0; i < 100; i++ {
		results[i] = &Result{
			Project: &project.Project{
				Path:         "/root/user/project" + string(rune('0'+(i%10))),
				Name:         "project" + string(rune('0'+(i%10))),
				Organisation: "user" + string(rune('a'+(i%26))),
			},
			Distance: i,
		}
	}

	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	service := NewService(logger, "/root")

	opts := Options{
		Separator: "\n",
		AbsPath:   true,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = service.Format(results, opts)
	}
}

package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"strings"

	"github.com/gfanton/project/internal/config"
	"github.com/gfanton/project/internal/query"
	"github.com/peterbourgon/ff/v3/ffcli"
)

type queryConfig struct {
	exclude      []string
	all          bool
	absPath      bool
	separator    string
	limit        int
	showDistance bool
}

type excludeValue struct {
	values *[]string
}

func (e excludeValue) Set(value string) error {
	*e.values = append(*e.values, value)
	return nil
}

func (e excludeValue) String() string {
	if e.values == nil {
		return ""
	}
	return strings.Join(*e.values, ",")
}

func newQueryCommand(logger *slog.Logger, cfg *config.Config) *ffcli.Command {
	var queryCfg queryConfig

	fs := flag.NewFlagSet("query", flag.ExitOnError)
	fs.Var(excludeValue{&queryCfg.exclude}, "exclude", "exclude project path (can be used multiple times)")
	fs.BoolVar(&queryCfg.absPath, "abspath", false, "return absolute paths instead of project names")
	fs.StringVar(&queryCfg.separator, "sep", "\n", "separator between results")
	fs.IntVar(&queryCfg.limit, "limit", 20, "limit number of results (0 = no limit)")
	fs.BoolVar(&queryCfg.showDistance, "v", false, "show distance with matching projects")

	return &ffcli.Command{
		Name:       "query",
		ShortUsage: "project query [flags] [search]",
		ShortHelp:  "Search for projects using fuzzy matching",
		LongHelp: `Search for projects using fuzzy matching.

Examples:
  project query myapp
  project query --exclude $(pwd) myapp
  project query --abspath --limit 5 app`,
		FlagSet: fs,
		Exec: func(ctx context.Context, args []string) error {
			return runQuery(ctx, logger, cfg, queryCfg, args)
		},
	}
}

func runQuery(ctx context.Context, logger *slog.Logger, cfg *config.Config, queryCfg queryConfig, args []string) error {
	searchQuery := strings.Join(args, " ")

	queryService := query.NewService(logger, cfg.RootDir)

	opts := query.Options{
		Query:        searchQuery,
		Exclude:      queryCfg.exclude,
		AbsPath:      queryCfg.absPath,
		Separator:    queryCfg.separator,
		Limit:        queryCfg.limit,
		ShowDistance: queryCfg.showDistance,
	}

	results, err := queryService.Search(ctx, opts)
	if err != nil {
		return fmt.Errorf("search failed: %w", err)
	}

	if len(results) == 0 {
		return fmt.Errorf("no matching projects found")
	}

	output := queryService.Format(results, opts)
	fmt.Print(output)

	// Add newline if not already present and we have output
	if output != "" && !strings.HasSuffix(output, "\n") {
		fmt.Println()
	}

	return nil
}

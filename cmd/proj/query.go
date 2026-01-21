package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"strings"

	"github.com/gfanton/projects"
	"github.com/gfanton/projects/internal/config"
	"github.com/peterbourgon/ff/v4"
)

type queryConfig struct {
	Exclude      []string
	AbsPath      bool
	Separator    string
	Limit        int
	ShowDistance bool
}

func newQueryCommand(logger *slog.Logger, cfg *config.Config, projectsCfg *projects.Config, projectsLogger projects.Logger) *ff.Command {
	queryCfg := &queryConfig{}
	fs := ff.NewFlagSet("query")
	fs.StringSetVar(&queryCfg.Exclude, 0, "exclude", "exclude project path (repeatable)")
	fs.BoolVar(&queryCfg.AbsPath, 0, "abspath", "return absolute paths instead of project names")
	fs.StringVar(&queryCfg.Separator, 0, "sep", "\n", "separator between results")
	fs.IntVar(&queryCfg.Limit, 0, "limit", 20, "limit number of results (0 = no limit)")
	fs.BoolVar(&queryCfg.ShowDistance, 'v', "", "show distance with matching projects")

	return &ff.Command{
		Name:      "query",
		Usage:     "proj query [flags] [search]",
		ShortHelp: "Search for projects and workspaces using fuzzy matching",
		LongHelp: `Search for projects and workspaces using fuzzy matching.

Project search:
  proj query myapp                    # Search projects matching "myapp"
  proj query foo/bar                  # Search for "foo/bar" project

Workspace search (requires ':' syntax):
  proj query foo/bar:feature          # Search workspace "feature" in "foo/bar" project
  proj query :feature                 # Search workspaces named "feature" in all projects
  proj query foo:                     # List all workspaces in projects matching "foo"

Examples:
  proj query myapp
  proj query --exclude $(pwd) myapp
  proj query --abspath --limit 5 app
  proj query gfanton/projects:main
  proj query :dev`,
		Flags: fs,
		Exec: func(ctx context.Context, args []string) error {
			return runQuery(ctx, logger, cfg, projectsCfg, projectsLogger, *queryCfg, args)
		},
	}
}

func runQuery(ctx context.Context, logger *slog.Logger, cfg *config.Config, projectsCfg *projects.Config, projectsLogger projects.Logger, queryCfg queryConfig, args []string) error {
	searchQuery := strings.Join(args, " ")

	queryService := projects.NewQueryService(projectsCfg, projectsLogger)
	projectService := projects.NewProjectService(projectsCfg, projectsLogger)

	// Detect current project if query starts with ':' (workspace query without project prefix)
	var currentProject *projects.Project
	if strings.HasPrefix(searchQuery, ":") {
		wd, err := os.Getwd()
		if err == nil {
			if proj, err := projectService.FindFromPath(wd); err == nil {
				currentProject = proj
				logger.Debug("detected current project for workspace query", "project", proj.String())
			}
		}
	}

	opts := projects.SearchOptions{
		Query:          searchQuery,
		Exclude:        queryCfg.Exclude,
		AbsPath:        queryCfg.AbsPath,
		Separator:      queryCfg.Separator,
		Limit:          queryCfg.Limit,
		ShowDistance:   queryCfg.ShowDistance,
		CurrentProject: currentProject,
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

package main

import (
	"context"
	"flag"
	"fmt"
	"io/fs"
	"path/filepath"
	"sort"
	"strings"

	"github.com/lithammer/fuzzysearch/fuzzy"
	"github.com/peterbourgon/ff/v3/ffcli"
	"go.uber.org/zap"
)

type ExcludeValue struct {
	elems []string
}

func (e *ExcludeValue) Set(value string) error {
	if e.elems == nil {
		e.elems = []string{}
	}

	e.elems = append(e.elems, value)
	return nil
}

func (e *ExcludeValue) String() string {
	return strings.Join(e.elems, ":")
}

type QueryConfig struct {
	*RootConfig

	All     bool
	Exclude ExcludeValue
}

func ProjectsQuery(ctx context.Context, cfg *QueryConfig, values ...string) error {
	value := strings.Join(values, "")
	excludes := strings.Split(cfg.Exclude.String(), ":")
	logger.Debug("query",
		zap.String("lookup", value),
		zap.Strings("exclude", excludes),
		zap.Bool("all", cfg.All))

	projects := []*Project{}
	dist := map[string]int{}

	err := WalkProject(cfg.RootDir, func(d fs.DirEntry, p *Project) error {
		for _, expath := range excludes {
			abs, err := filepath.Abs(expath)
			if err != nil {
				return fmt.Errorf("invalid path(%s): %w", expath, err)
			}

			if strings.Contains(p.Path, abs) {
				return filepath.SkipDir
			}
		}

		name := p.String()
		if val := fuzzy.RankMatchFold(value, name); val >= 0 {
			dist[name] = val
			projects = append(projects, p)
			logger.Debug("matching project",
				zap.String("name", name), zap.Int("distance", val))
		}

		return nil
	})

	if err != nil {
		return fmt.Errorf("unable to process: %w", err)
	}

	if len(projects) == 0 {
		return fmt.Errorf("no match found")
	}

	sort.Slice(projects, func(i, j int) bool {
		valI, valJ := projects[i].String(), projects[j].String()
		return dist[valI] < dist[valJ]
	})

	until := 1
	if cfg.All {
		until = len(projects)
	}

	for i := 0; i < until; i++ {
		fmt.Println(projects[i].Path)
	}

	return nil
}

func queryCommand(rcfg *RootConfig) *ffcli.Command {
	var cfg QueryConfig
	cfg.RootConfig = rcfg

	flagSet := flag.NewFlagSet("query", flag.ExitOnError)
	flagSet.Var(&cfg.Exclude, "exclude", "exclude project")
	flagSet.BoolVar(&cfg.All, "all", false, "print all project ranked from the highest to lowest match")

	return &ffcli.Command{
		Name:        "query",
		ShortUsage:  "projects query",
		ShortHelp:   "query projects",
		FlagSet:     flagSet,
		Subcommands: []*ffcli.Command{},
		Exec: func(ctx context.Context, args []string) error {
			if len(args) >= 1 {
				return ProjectsQuery(ctx, &cfg, args...)
			}

			return fmt.Errorf("invalid argument(s)")
		},
	}
}

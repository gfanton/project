package main

import (
	"context"
	"flag"
	"fmt"
	"io/fs"
	"log"
	"path/filepath"
	"sort"
	"strings"

	"github.com/lithammer/fuzzysearch/fuzzy"
	"github.com/peterbourgon/ff/v3/ffcli"
)

type QueryResultFormat string

const (
	String  QueryResultFormat = "string"
	Orga    QueryResultFormat = "orga"
	Name    QueryResultFormat = "name"
	compdef QueryResultFormat = "compdef"
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
	Sep     string
	Limit   int
	AbsPath bool
}

func ProjectQuery(ctx context.Context, logger *log.Logger, cfg *QueryConfig, values ...string) error {
	value := strings.Join(values, "")
	excludes := strings.Split(cfg.Exclude.String(), ":")
	logger.Printf("query: lookup=`%s`, excludes=`%+v`, all=%t", value, excludes, cfg.All)

	project := []*Project{}
	distances := map[string]int{}
	err := WalkProject(cfg.RootDir, func(d fs.DirEntry, p *Project) error {
		for _, expath := range excludes {
			expath = strings.TrimSpace(expath)
			if expath == "" {
				continue
			}

			abs, err := filepath.Abs(expath)
			if err != nil {
				return fmt.Errorf("invalid path(%s): %w", expath, err)
			}

			if p.Path == abs {
				logger.Printf("skipping: %s (exclude: %s)", p.Path, abs)
				return filepath.SkipDir
			}
		}

		dist, name := 1, p.String()
		if value != "" {
			if dist = fuzzy.RankMatchFold(value, name); dist < 0 {
				return nil
			}
		}

		distances[name] = dist
		project = append(project, p)
		logger.Printf("matching project: name=`%s`, dist=%d", name, dist)

		return nil
	})

	if err != nil {
		return fmt.Errorf("unable to process: %w", err)
	}

	if len(project) == 0 {
		return fmt.Errorf("no match found")
	}

	sort.Slice(project, func(i, j int) bool {
		valI, valJ := project[i].String(), project[j].String()
		return distances[valI] < distances[valJ]
	})

	until := cfg.Limit
	if cfg.Limit == 0 || cfg.All {
		until = len(project)
	}

	getpath := func(project *Project) string {
		if cfg.AbsPath {
			return project.Path
		}
		return project.String()
	}

	var b strings.Builder
	b.Grow(until)
	fmt.Print(getpath(project[0]))
	for i := 1; i < until; i++ {
		fmt.Print(cfg.Sep)
		fmt.Print(getpath(project[i]))
	}
	fmt.Print("\n")

	return nil
}

func queryCommand(logger *log.Logger, rcfg *RootConfig) *ffcli.Command {
	var cfg QueryConfig
	cfg.RootConfig = rcfg

	flagSet := flag.NewFlagSet("query", flag.ExitOnError)
	flagSet.Var(&cfg.Exclude, "exclude", "exclude project")
	flagSet.BoolVar(&cfg.All, "all", false, "print all project ranked from the highest to lowest match")
	flagSet.BoolVar(&cfg.AbsPath, "abspath", false, "print abs path")
	flagSet.StringVar(&cfg.Sep, "sep", "\n", "separator between result")
	flagSet.IntVar(&cfg.Limit, "limit", 0, "limit the query result")
	return &ffcli.Command{
		Name:        "query",
		ShortUsage:  "project query",
		ShortHelp:   "query project",
		FlagSet:     flagSet,
		Subcommands: []*ffcli.Command{},
		Exec: func(ctx context.Context, args []string) error {
			switch {
			case len(args) > 0 && cfg.Limit == 0:
				cfg.Limit = 1
			}

			return ProjectQuery(ctx, logger, &cfg, args...)
		},
	}
}

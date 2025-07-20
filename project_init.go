package main

import (
	"context"
	"embed"
	"flag"
	"fmt"
	"io/fs"
	"log"
	"os"
	"strings"
	"text/template"

	"github.com/peterbourgon/ff/v3/ffcli"
)

//go:embed template

var templateContent embed.FS

type InitConfig struct {
	*RootConfig

	Capital bool
}

type templates map[string]*template.Template

func (f templates) PrintAvailable() {
	fmt.Println("init config available:")
	for name := range f {
		fmt.Printf("- %s\n", name)
	}
}

func parseInitFiles(logger *log.Logger) templates {
	initf := make(templates)

	err := fs.WalkDir(templateContent, "template", func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if entry.IsDir() {
			return nil
		}

		if strings.HasSuffix(entry.Name(), ".init") {
			logger.Printf("parsing embed file: %s", path)

			pt, err := template.ParseFS(templateContent, path, "template/*.init")
			if err != nil {
				return err
			}

			name := strings.TrimSuffix(entry.Name(), ".init")
			initf[name] = pt
		}

		return nil
	})

	if err != nil {
		panic(fmt.Errorf("walk template embed directory failed:  %w", err))
	}

	return initf
}

func ProjectInit(ctx context.Context, logger *log.Logger, rcfg *InitConfig, args ...string) error {
	initf := parseInitFiles(logger)
	if len(args) != 1 {
		initf.PrintAvailable()
		return nil
	}

	config := args[0]
	tpt, exist := initf[config]
	if !exist {
		initf.PrintAvailable()
		return fmt.Errorf("config `%s` not available", config)
	}

	ex, err := os.Executable()
	if err != nil {
		return fmt.Errorf("unable to get path executable: %w", err)
	}

	data := map[string]string{
		"Exec": ex,
	}

	return tpt.Execute(os.Stdout, data)
}

func initCommand(logger *log.Logger, rcfg *RootConfig) *ffcli.Command {
	var cfg InitConfig
	cfg.RootConfig = rcfg

	flagSet := flag.NewFlagSet("init", flag.ExitOnError)
	flagSet.BoolVar(&cfg.Capital, "c", false, "display in capital")

	return &ffcli.Command{
		Name:        "init",
		ShortUsage:  "project init <name>",
		ShortHelp:   "init project",
		FlagSet:     flagSet,
		Subcommands: []*ffcli.Command{},
		Exec: func(ctx context.Context, args []string) error {
			return ProjectInit(ctx, logger, &cfg, args...)
		},
	}
}

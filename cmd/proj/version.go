package main

import (
	"context"
	"flag"
	"fmt"
	"runtime"

	"github.com/peterbourgon/ff/v3/ffcli"
)

var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
	builtBy = "unknown"
)

type versionConfig struct {
	verbose bool
}

func NewVersionCommand(parent *rootConfig) *ffcli.Command {
	cfg := &versionConfig{}
	fs := flag.NewFlagSet("proj version", flag.ContinueOnError)
	fs.BoolVar(&cfg.verbose, "v", false, "show verbose version information")
	fs.BoolVar(&cfg.verbose, "verbose", false, "show verbose version information")

	return &ffcli.Command{
		Name:       "version",
		ShortUsage: "proj version [-v]",
		ShortHelp:  "Show version information",
		FlagSet:    fs,
		Exec: func(ctx context.Context, args []string) error {
			return runVersion(ctx, parent, cfg)
		},
	}
}

func runVersion(ctx context.Context, rootCfg *rootConfig, cfg *versionConfig) error {
	if cfg.verbose {
		fmt.Printf("proj version %s\n", version)
		fmt.Printf("  commit: %s\n", commit)
		fmt.Printf("  built at: %s\n", date)
		fmt.Printf("  built by: %s\n", builtBy)
		fmt.Printf("  go version: %s\n", runtime.Version())
		fmt.Printf("  platform: %s/%s\n", runtime.GOOS, runtime.GOARCH)
	} else {
		fmt.Println(version)
	}

	return nil
}

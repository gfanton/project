package main

import (
	"context"
	"fmt"
	"runtime"

	"github.com/peterbourgon/ff/v4"
)

var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
	builtBy = "unknown"
)

type versionConfig struct {
	Verbose bool
}

func NewVersionCommand(parent *rootConfig) *ff.Command {
	cfg := &versionConfig{}
	fs := ff.NewFlagSet("proj version")
	fs.BoolVar(&cfg.Verbose, 'v', "verbose", "show verbose version information")

	return &ff.Command{
		Name:      "version",
		Usage:     "proj version [-v]",
		ShortHelp: "Show version information",
		Flags:     fs,
		Exec: func(ctx context.Context, args []string) error {
			return runVersion(ctx, parent, cfg)
		},
	}
}

func runVersion(_ context.Context, _ *rootConfig, cfg *versionConfig) error {
	if cfg.Verbose {
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

package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"

	"github.com/gfanton/projects/internal/config"
	"github.com/gfanton/projects/pkg/template"
	"github.com/peterbourgon/ff/v3/ffcli"
)

func newInitCommand(logger *slog.Logger, cfg *config.Config) *ffcli.Command {
	return &ffcli.Command{
		Name:       "init",
		ShortUsage: "proj init <shell>",
		ShortHelp:  "Generate shell integration script",
		LongHelp: `Generate shell integration script for the specified shell.

Supported shells:
  zsh    Generate zsh integration script

Example:
  eval "$(proj init zsh)"`,
		Exec: func(ctx context.Context, args []string) error {
			return runInit(ctx, logger, cfg, args)
		},
	}
}

func runInit(ctx context.Context, logger *slog.Logger, cfg *config.Config, args []string) error {
	if len(args) != 1 {
		return fmt.Errorf("exactly one shell argument required")
	}

	shell := args[0]
	switch shell {
	case "zsh":
		return generateZshInit(logger, cfg)
	default:
		return fmt.Errorf("unsupported shell: %s", shell)
	}
}

func generateZshInit(logger *slog.Logger, cfg *config.Config) error {
	execPath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("failed to get executable path: %w", err)
	}

	data := template.Data{
		Exec: execPath,
	}

	output, err := template.Render("zsh", data)
	if err != nil {
		return fmt.Errorf("failed to render zsh template: %w", err)
	}

	fmt.Print(output)
	return nil
}

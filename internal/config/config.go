package config

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/user"
	"path/filepath"
	"strings"

	"github.com/peterbourgon/ff/v4"
	"github.com/peterbourgon/ff/v4/fftoml"
)

// Config holds the global configuration for the project tool.
type Config struct {
	ConfigFile string `ff:"long=config,  usage='configuration file path'"`
	Debug      bool   `ff:"long=debug,   usage='enable debug logging'"`
	RootDir    string `ff:"long=root,    usage='root directory for projects'"`
	RootUser   string `ff:"long=user,    usage='default user for projects'"`
}

// NewConfig creates a new configuration with default values.
func NewConfig() (*Config, error) {
	u, err := user.Current()
	if err != nil {
		return nil, fmt.Errorf("failed to get current user: %w", err)
	}

	return &Config{
		ConfigFile: filepath.Join(u.HomeDir, ".projectrc"),
		RootDir:    filepath.Join(u.HomeDir, "code"),
		Debug:      false,
	}, nil
}

// Load loads configuration from flags, environment variables, and config file.
// Note: This only parses global config flags (--debug, --root, --user, --config).
// Subcommand flags and help are handled by the main command parser.
func (c *Config) Load(args []string) error {
	// Filter args to only extract global config flags
	// This is necessary because args may contain subcommands and their flags
	filteredArgs := filterGlobalFlags(args)

	fs := ff.NewFlagSet("project")
	if err := fs.AddStruct(c); err != nil {
		return fmt.Errorf("failed to add config struct: %w", err)
	}

	err := ff.Parse(fs, filteredArgs,
		ff.WithEnvVarPrefix("PROJECT"),
		ff.WithConfigFileFlag("config"),
		ff.WithConfigAllowMissingFile(),
		ff.WithConfigFileParser(fftoml.Parse),
	)
	if err != nil {
		// Ignore help requests - those are handled by the main command parser
		if errors.Is(err, ff.ErrHelp) || errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return fmt.Errorf("failed to parse configuration: %w", err)
	}

	// Expand paths
	c.RootDir = expandPath(c.RootDir)
	c.ConfigFile = expandPath(c.ConfigFile)

	// Ensure root directory exists
	if err := c.ensureRootDir(); err != nil {
		return fmt.Errorf("failed to ensure root directory: %w", err)
	}

	return nil
}

// filterGlobalFlags extracts only global config flags from args.
// Global flags are: --debug, --root, --user, --config (and their values)
func filterGlobalFlags(args []string) []string {
	var filtered []string
	globalFlags := map[string]bool{
		"--debug":  false, // bool flag, no value
		"--root":   true,  // string flag, has value
		"--user":   true,  // string flag, has value
		"--config": true,  // string flag, has value
	}

	for i := 0; i < len(args); i++ {
		arg := args[i]

		// Check for --flag=value format
		if strings.HasPrefix(arg, "--") {
			parts := strings.SplitN(arg, "=", 2)
			flagName := parts[0]

			if hasValue, ok := globalFlags[flagName]; ok {
				filtered = append(filtered, arg)
				// If flag needs a value and wasn't provided with =, get next arg
				if hasValue && len(parts) == 1 && i+1 < len(args) && !strings.HasPrefix(args[i+1], "-") {
					i++
					filtered = append(filtered, args[i])
				}
			}
		}
	}

	return filtered
}

// Logger creates a structured logger based on the debug configuration.
func (c *Config) Logger() *slog.Logger {
	level := slog.LevelInfo
	if c.Debug {
		level = slog.LevelDebug
	}

	handler := NewToolHandler(os.Stderr, level)
	return slog.New(handler)
}

// ToolHandler is a custom slog handler optimized for CLI tools
type ToolHandler struct {
	writer io.Writer
	level  slog.Level
}

// NewToolHandler creates a new tool-friendly handler
func NewToolHandler(w io.Writer, level slog.Level) *ToolHandler {
	return &ToolHandler{
		writer: w,
		level:  level,
	}
}

// Enabled returns true if the handler should handle the given level
func (h *ToolHandler) Enabled(_ context.Context, level slog.Level) bool {
	return level >= h.level
}

// Handle formats and writes the log record
func (h *ToolHandler) Handle(_ context.Context, r slog.Record) error {
	var prefix string
	switch r.Level {
	case slog.LevelDebug:
		prefix = "D: "
	case slog.LevelInfo:
		prefix = "" // No prefix for info messages
	case slog.LevelWarn:
		prefix = "!W: "
	case slog.LevelError:
		prefix = "ERROR: "
	default:
		prefix = ""
	}

	// Build the message
	msg := prefix + r.Message

	// Add attributes if any
	if r.NumAttrs() > 0 {
		var attrs []string
		r.Attrs(func(a slog.Attr) bool {
			attrs = append(attrs, fmt.Sprintf("%s=%v", a.Key, a.Value))
			return true
		})
		if len(attrs) > 0 {
			msg += " (" + strings.Join(attrs, " ") + ")"
		}
	}

	_, err := fmt.Fprintln(h.writer, msg)
	return err
}

// WithAttrs returns a new handler with the given attributes
func (h *ToolHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	// For simplicity, we don't support persistent attributes in this tool handler
	return h
}

// WithGroup returns a new handler with the given group name
func (h *ToolHandler) WithGroup(name string) slog.Handler {
	// For simplicity, we don't support groups in this tool handler
	return h
}

// ensureRootDir creates the root directory if it doesn't exist.
func (c *Config) ensureRootDir() error {
	if _, err := os.Stat(c.RootDir); os.IsNotExist(err) {
		slog.Info("creating root directory", "path", c.RootDir)
		if err := os.MkdirAll(c.RootDir, 0755); err != nil {
			return fmt.Errorf("failed to create root directory %s: %w", c.RootDir, err)
		}
	}
	return nil
}

// expandPath expands environment variables and ~ in paths.
func expandPath(path string) string {
	path = os.ExpandEnv(path)
	if strings.HasPrefix(path, "~") {
		if u, err := user.Current(); err == nil {
			return strings.Replace(path, "~", u.HomeDir, 1)
		}
	}
	return path
}

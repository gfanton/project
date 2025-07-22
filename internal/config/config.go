package config

import (
	"context"
	"flag"
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/user"
	"path/filepath"
	"strings"

	"github.com/peterbourgon/ff/v3"
	"github.com/peterbourgon/ff/v3/fftoml"
)

// Config holds the global configuration for the project tool.
type Config struct {
	ConfigFile string
	Debug      bool
	RootDir    string
	RootUser   string
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
func (c *Config) Load(args []string) error {
	fs := createFlagSet(c)

	err := ff.Parse(fs, args,
		ff.WithEnvVarPrefix("PROJECT"),
		ff.WithConfigFileFlag("config"),
		ff.WithAllowMissingConfigFile(true),
		ff.WithConfigFileParser(fftoml.Parser),
	)
	if err != nil {
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

// createFlagSet creates a flag set with the configuration options.
func createFlagSet(cfg *Config) *flag.FlagSet {
	fs := flag.NewFlagSet("project", flag.ExitOnError)
	fs.StringVar(&cfg.RootDir, "root", cfg.RootDir, "root directory for projects")
	fs.StringVar(&cfg.RootUser, "user", cfg.RootUser, "default user for projects")
	fs.StringVar(&cfg.ConfigFile, "config", cfg.ConfigFile, "configuration file path")
	fs.BoolVar(&cfg.Debug, "debug", cfg.Debug, "enable debug logging")
	return fs
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

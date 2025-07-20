package config

import (
	"flag"
	"fmt"
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
	opts := &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}

	if c.Debug {
		opts.Level = slog.LevelDebug
		opts.AddSource = true
	}

	handler := slog.NewTextHandler(os.Stderr, opts)
	return slog.New(handler)
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

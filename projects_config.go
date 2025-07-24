package projects

import (
	"fmt"
	"os"
	"os/user"
	"path/filepath"
	"strings"
)

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

// EnsureRootDir creates the root directory if it doesn't exist.
func (c *Config) EnsureRootDir() error {
	if _, err := os.Stat(c.RootDir); os.IsNotExist(err) {
		if err := os.MkdirAll(c.RootDir, 0755); err != nil {
			return fmt.Errorf("failed to create root directory %s: %w", c.RootDir, err)
		}
	}
	return nil
}

// ExpandPath expands environment variables and ~ in paths.
func (c *Config) ExpandPath(path string) string {
	path = os.ExpandEnv(path)
	if strings.HasPrefix(path, "~") {
		if u, err := user.Current(); err == nil {
			return strings.Replace(path, "~", u.HomeDir, 1)
		}
	}
	return path
}
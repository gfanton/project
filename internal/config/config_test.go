package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestNewConfig(t *testing.T) {
	cfg, err := NewConfig()
	if err != nil {
		t.Fatalf("NewConfig() failed: %v", err)
	}

	if cfg == nil {
		t.Fatal("NewConfig() returned nil config")
	}

	// Check that default values are set
	if cfg.ConfigFile == "" {
		t.Error("ConfigFile should have a default value")
	}

	if cfg.RootDir == "" {
		t.Error("RootDir should have a default value")
	}

	if cfg.Debug != false {
		t.Error("Debug should default to false")
	}

	// Check that paths contain expected patterns
	if !strings.Contains(cfg.ConfigFile, ".projectrc") {
		t.Errorf("ConfigFile should contain '.projectrc', got: %s", cfg.ConfigFile)
	}

	if !strings.Contains(cfg.RootDir, "code") {
		t.Errorf("RootDir should contain 'code', got: %s", cfg.RootDir)
	}
}

func TestConfigLoad(t *testing.T) {
	tests := []struct {
		name    string
		args    []string
		want    func(*Config) bool
		wantErr bool
	}{
		{
			name: "empty args",
			args: []string{},
			want: func(c *Config) bool {
				return c.Debug == false && c.RootUser == ""
			},
			wantErr: false,
		},
		{
			name: "debug flag",
			args: []string{"--debug"},
			want: func(c *Config) bool {
				return c.Debug == true
			},
			wantErr: false,
		},
		{
			name: "user flag",
			args: []string{"--user", "testuser"},
			want: func(c *Config) bool {
				return c.RootUser == "testuser"
			},
			wantErr: false,
		},
		{
			name: "multiple flags",
			args: []string{"--debug", "--user", "testuser"},
			want: func(c *Config) bool {
				return c.Debug == true && c.RootUser == "testuser"
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg, err := NewConfig()
			if err != nil {
				t.Fatalf("NewConfig() failed: %v", err)
			}

			// Use a temporary directory for testing
			tempDir, err := os.MkdirTemp("", "project-test-*")
			if err != nil {
				t.Fatalf("Failed to create temp dir: %v", err)
			}
			defer os.RemoveAll(tempDir)

			cfg.RootDir = tempDir
			cfg.ConfigFile = filepath.Join(tempDir, ".projectrc")

			err = cfg.Load(tt.args)
			if (err != nil) != tt.wantErr {
				t.Errorf("Load() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && !tt.want(cfg) {
				t.Errorf("Load() result doesn't match expectations for args: %v", tt.args)
			}
		})
	}
}

func TestConfigLogger(t *testing.T) {
	tests := []struct {
		name  string
		debug bool
	}{
		{
			name:  "debug disabled",
			debug: false,
		},
		{
			name:  "debug enabled",
			debug: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &Config{Debug: tt.debug}
			logger := cfg.Logger()

			if logger == nil {
				t.Fatal("Logger() returned nil")
			}

			// Test that logger is functional
			logger.Info("test message")
			logger.Debug("debug message")
		})
	}
}

func TestExpandPath(t *testing.T) {
	tests := []struct {
		name     string
		path     string
		expected string
	}{
		{
			name:     "no expansion needed",
			path:     "/absolute/path",
			expected: "/absolute/path",
		},
		{
			name:     "env var expansion",
			path:     "$HOME/Documents",
			expected: "", // Will be set in test
		},
	}

	// Set up environment for testing
	originalHome := os.Getenv("HOME")
	testHome := "/test/home"
	os.Setenv("HOME", testHome)
	defer os.Setenv("HOME", originalHome)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set expected values based on test environment
			if strings.Contains(tt.path, "$HOME") {
				tt.expected = strings.Replace(tt.path, "$HOME", testHome, 1)
			}

			result := expandPath(tt.path)
			if result != tt.expected {
				t.Errorf("expandPath(%s) = %s, want %s", tt.path, result, tt.expected)
			}
		})
	}

	// Test tilde expansion separately with real home directory
	t.Run("tilde expansion", func(t *testing.T) {
		result := expandPath("~/Documents")
		// The function uses user.Current() which gets the actual home directory
		// So we just verify that it starts with / and contains Documents
		if !strings.HasPrefix(result, "/") {
			t.Errorf("expandPath(~/Documents) should return absolute path, got %s", result)
		}
		if !strings.HasSuffix(result, "/Documents") {
			t.Errorf("expandPath(~/Documents) should end with /Documents, got %s", result)
		}
		if strings.Contains(result, "~") {
			t.Errorf("expandPath(~/Documents) should not contain ~, got %s", result)
		}
	})
}

func TestConfigEnsureRootDir(t *testing.T) {
	// Test directory creation
	tempDir, err := os.MkdirTemp("", "project-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	cfg := &Config{
		RootDir: filepath.Join(tempDir, "new-dir"),
	}

	// Directory shouldn't exist yet
	if _, err := os.Stat(cfg.RootDir); err == nil {
		t.Fatal("Test directory already exists")
	}

	// ensureRootDir should create it
	err = cfg.ensureRootDir()
	if err != nil {
		t.Fatalf("ensureRootDir() failed: %v", err)
	}

	// Directory should now exist
	if _, err := os.Stat(cfg.RootDir); os.IsNotExist(err) {
		t.Fatal("ensureRootDir() didn't create directory")
	}

	// Running again should not error
	err = cfg.ensureRootDir()
	if err != nil {
		t.Fatalf("ensureRootDir() failed on existing directory: %v", err)
	}
}

func TestConfigWithEnvironmentVariables(t *testing.T) {
	// Save original environment
	originalVars := map[string]string{
		"PROJECT_ROOT":  os.Getenv("PROJECT_ROOT"),
		"PROJECT_USER":  os.Getenv("PROJECT_USER"),
		"PROJECT_DEBUG": os.Getenv("PROJECT_DEBUG"),
	}

	// Restore environment after test
	defer func() {
		for key, value := range originalVars {
			if value == "" {
				os.Unsetenv(key)
			} else {
				os.Setenv(key, value)
			}
		}
	}()

	// Create temp directory for testing
	tempDir, err := os.MkdirTemp("", "project-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Set test environment variables
	testUser := "testuser"
	os.Setenv("PROJECT_ROOT", tempDir) // Use temp dir for root
	os.Setenv("PROJECT_USER", testUser)
	os.Setenv("PROJECT_DEBUG", "true")

	cfg, err := NewConfig()
	if err != nil {
		t.Fatalf("NewConfig() failed: %v", err)
	}

	cfg.ConfigFile = filepath.Join(tempDir, ".projectrc")

	err = cfg.Load([]string{})
	if err != nil {
		t.Fatalf("Load() failed: %v", err)
	}

	// Check that environment variables were used
	if cfg.RootUser != testUser {
		t.Errorf("Expected RootUser=%s from env var, got %s", testUser, cfg.RootUser)
	}

	if cfg.Debug != true {
		t.Errorf("Expected Debug=true from env var, got %t", cfg.Debug)
	}

	if cfg.RootDir != tempDir {
		t.Errorf("Expected RootDir=%s from env var, got %s", tempDir, cfg.RootDir)
	}
}

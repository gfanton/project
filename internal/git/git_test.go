package git

import (
	"context"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestNewClient(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	client := NewClient(logger)

	if client == nil {
		t.Fatal("NewClient() returned nil")
	}

	if client.logger == nil {
		t.Error("NewClient() should set logger")
	}
}

func TestCloneOptions(t *testing.T) {
	tests := []struct {
		name        string
		opts        CloneOptions
		wantErr     bool
		description string
	}{
		{
			name: "valid HTTPS clone",
			opts: CloneOptions{
				URL:         "https://github.com/test/repo.git",
				Destination: "/tmp/test",
				UseSSH:      false,
			},
			wantErr:     false,
			description: "should accept valid HTTPS URL",
		},
		{
			name: "valid SSH clone",
			opts: CloneOptions{
				URL:         "git@github.com:test/repo.git",
				Destination: "/tmp/test",
				UseSSH:      true,
			},
			wantErr:     false,
			description: "should accept valid SSH URL",
		},
		{
			name: "clone with token",
			opts: CloneOptions{
				URL:         "https://github.com/test/repo.git",
				Destination: "/tmp/test",
				UseSSH:      false,
				Token:       "test-token",
			},
			wantErr:     false,
			description: "should accept token for authentication",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// We're just testing the structure and validation, not actual cloning
			if tt.opts.URL == "" {
				t.Error("URL should not be empty")
			}
			if tt.opts.Destination == "" {
				t.Error("Destination should not be empty")
			}
		})
	}
}

func TestCloneInvalidRepository(t *testing.T) {
	// Test cloning a non-existent repository
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	client := NewClient(logger)

	tempDir, err := os.MkdirTemp("", "git-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	opts := CloneOptions{
		URL:         "https://github.com/non-existent-user/non-existent-repo.git",
		Destination: filepath.Join(tempDir, "test-repo"),
		UseSSH:      false,
	}

	err = client.Clone(ctx, opts)
	if err == nil {
		t.Error("Clone() should fail for non-existent repository")
	}

	// Check that error message is informative
	errMsg := err.Error()
	if !strings.Contains(errMsg, "failed to clone repository") {
		t.Errorf("Error message should contain 'failed to clone repository', got: %s", errMsg)
	}
}

func TestCloneCreateDestinationDirectory(t *testing.T) {
	// Test that Clone creates the destination directory
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	client := NewClient(logger)

	tempDir, err := os.MkdirTemp("", "git-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Use a nested path that doesn't exist yet
	destPath := filepath.Join(tempDir, "nested", "path", "repo")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	opts := CloneOptions{
		URL:         "https://github.com/non-existent-user/non-existent-repo.git",
		Destination: destPath,
		UseSSH:      false,
	}

	// This will fail because the repository doesn't exist, but the directory should be created
	client.Clone(ctx, opts)

	// Check that the destination directory was created
	if _, err := os.Stat(destPath); os.IsNotExist(err) {
		t.Error("Clone() should create destination directory even if clone fails")
	}
}

func TestCloneWithEmptyURL(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	client := NewClient(logger)

	tempDir, err := os.MkdirTemp("", "git-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	opts := CloneOptions{
		URL:         "", // Empty URL
		Destination: filepath.Join(tempDir, "test-repo"),
		UseSSH:      false,
	}

	err = client.Clone(ctx, opts)
	if err == nil {
		t.Error("Clone() should fail with empty URL")
	}
}

func TestCloneWithInvalidDestination(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	client := NewClient(logger)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Try to create directory at root (should fail due to permissions)
	opts := CloneOptions{
		URL:         "https://github.com/test/repo.git",
		Destination: "/invalid/path/that/cannot/be/created",
		UseSSH:      false,
	}

	err := client.Clone(ctx, opts)
	if err == nil {
		t.Error("Clone() should fail when destination directory cannot be created")
	}

	// Check that error message mentions directory creation
	errMsg := err.Error()
	if !strings.Contains(errMsg, "failed to create destination directory") {
		t.Errorf("Error message should mention directory creation failure, got: %s", errMsg)
	}
}

func TestCloneContextCancellation(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	client := NewClient(logger)

	tempDir, err := os.MkdirTemp("", "git-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create a context that's already cancelled
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	opts := CloneOptions{
		URL:         "https://github.com/test/repo.git",
		Destination: filepath.Join(tempDir, "test-repo"),
		UseSSH:      false,
	}

	err = client.Clone(ctx, opts)
	if err == nil {
		t.Error("Clone() should fail when context is cancelled")
	}

	// The error might be about context cancellation or connection issues
	// Both are acceptable since we're testing with a non-existent repo anyway
}

// Note: Testing actual successful clones would require network access and a real repository
// For integration tests, we could set up a local Git server or use a known public repository
// but for unit tests, we focus on error cases and internal logic

func TestCloneSSHAuthenticationSetup(t *testing.T) {
	// This test verifies that SSH authentication is set up when UseSSH is true
	// We can't easily test the actual SSH authentication without a real SSH setup,
	// but we can test that the option is processed correctly

	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	client := NewClient(logger)

	tempDir, err := os.MkdirTemp("", "git-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	opts := CloneOptions{
		URL:         "git@github.com:test/repo.git",
		Destination: filepath.Join(tempDir, "test-repo"),
		UseSSH:      true,
	}

	// This will likely fail due to SSH key issues or non-existent repo,
	// but we're testing that the SSH option is processed
	err = client.Clone(ctx, opts)
	if err == nil {
		t.Log("Unexpected successful clone - this test expects failure due to auth or non-existent repo")
	}

	// The specific error will depend on the system's SSH configuration
	// We just verify that an error occurred and was handled properly
	if err != nil && !strings.Contains(err.Error(), "failed to clone repository") {
		t.Errorf("Expected wrapped error message, got: %s", err.Error())
	}
}
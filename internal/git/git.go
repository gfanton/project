package git

import (
	"context"
	"fmt"
	"log/slog"
	"os"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/transport/http"
	"github.com/go-git/go-git/v5/plumbing/transport/ssh"
)

const defaultDirPerms = 0755

// Client provides Git operations.
type Client struct {
	logger *slog.Logger
}

// NewClient creates a new Git client.
func NewClient(logger *slog.Logger) *Client {
	return &Client{
		logger: logger,
	}
}

// CloneOptions holds options for cloning repositories.
type CloneOptions struct {
	URL         string
	Destination string
	UseSSH      bool
	Token       string
}

// Clone clones a repository to the specified destination.
func (c *Client) Clone(ctx context.Context, opts CloneOptions) error {
	c.logger.Debug("cloning repository",
		"url", opts.URL,
		"destination", opts.Destination,
		"use_ssh", opts.UseSSH,
	)

	// Ensure destination directory exists
	if err := os.MkdirAll(opts.Destination, defaultDirPerms); err != nil {
		return fmt.Errorf("create destination directory: %w", err)
	}

	cloneOpts := &git.CloneOptions{
		URL:      opts.URL,
		Progress: os.Stdout,
	}

	// Set up authentication if needed
	if opts.UseSSH {
		auth, err := ssh.NewSSHAgentAuth("git")
		if err != nil {
			return fmt.Errorf("failed to create SSH auth: %w", err)
		}
		cloneOpts.Auth = auth
	} else if opts.Token != "" {
		cloneOpts.Auth = &http.BasicAuth{
			Username: "git",
			Password: opts.Token,
		}
	}

	_, err := git.PlainCloneContext(ctx, opts.Destination, false, cloneOpts)
	if err != nil {
		return fmt.Errorf("failed to clone repository: %w", err)
	}

	c.logger.Info("repository cloned successfully",
		"url", opts.URL,
		"destination", opts.Destination,
	)

	return nil
}

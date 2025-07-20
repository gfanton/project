package project

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/go-git/go-git/v5"
)

func TestParseProject(t *testing.T) {
	tests := []struct {
		name        string
		rootDir     string
		defaultUser string
		projectName string
		expected    *Project
		wantErr     bool
	}{
		{
			name:        "project with explicit user",
			rootDir:     "/root",
			defaultUser: "defaultuser",
			projectName: "user/project",
			expected: &Project{
				Path:         "/root/user/project",
				Name:         "project",
				Organisation: "user",
			},
			wantErr: false,
		},
		{
			name:        "project with default user",
			rootDir:     "/root",
			defaultUser: "defaultuser",
			projectName: "project",
			expected: &Project{
				Path:         "/root/defaultuser/project",
				Name:         "project",
				Organisation: "defaultuser",
			},
			wantErr: false,
		},
		{
			name:        "project without default user",
			rootDir:     "/root",
			defaultUser: "",
			projectName: "project",
			expected:    nil,
			wantErr:     true,
		},
		{
			name:        "malformed project name",
			rootDir:     "/root",
			defaultUser: "defaultuser",
			projectName: "user/project/extra",
			expected:    nil,
			wantErr:     true,
		},
		{
			name:        "empty project name",
			rootDir:     "/root",
			defaultUser: "defaultuser",
			projectName: "",
			expected: &Project{
				Path:         "/root/defaultuser",
				Name:         "",
				Organisation: "defaultuser",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ParseProject(tt.rootDir, tt.defaultUser, tt.projectName)

			if (err != nil) != tt.wantErr {
				t.Errorf("ParseProject() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr {
				return
			}

			if result == nil {
				t.Fatal("ParseProject() returned nil without error")
			}

			if result.Path != tt.expected.Path {
				t.Errorf("ParseProject() Path = %v, want %v", result.Path, tt.expected.Path)
			}

			if result.Name != tt.expected.Name {
				t.Errorf("ParseProject() Name = %v, want %v", result.Name, tt.expected.Name)
			}

			if result.Organisation != tt.expected.Organisation {
				t.Errorf("ParseProject() Organisation = %v, want %v", result.Organisation, tt.expected.Organisation)
			}
		})
	}
}

func TestProjectString(t *testing.T) {
	p := &Project{
		Path:         "/root/user/project",
		Name:         "project",
		Organisation: "user",
	}

	expected := "user/project"
	result := p.String()

	if result != expected {
		t.Errorf("Project.String() = %v, want %v", result, expected)
	}
}

func TestProjectGitURLs(t *testing.T) {
	p := &Project{
		Path:         "/root/user/project",
		Name:         "project",
		Organisation: "user",
	}

	tests := []struct {
		name     string
		method   func() string
		expected string
	}{
		{
			name:     "HTTP URL",
			method:   p.GitHTTPURL,
			expected: "https://github.com/user/project.git",
		},
		{
			name:     "SSH URL",
			method:   p.GitSSHURL,
			expected: "git@github.com:user/project.git",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.method()
			if result != tt.expected {
				t.Errorf("%s = %v, want %v", tt.name, result, tt.expected)
			}
		})
	}
}

func TestProjectGitDir(t *testing.T) {
	p := &Project{
		Path:         "/root/user/project",
		Name:         "project",
		Organisation: "user",
	}

	expected := "/root/user/project/.git"
	result := p.GitDir()

	if result != expected {
		t.Errorf("Project.GitDir() = %v, want %v", result, expected)
	}
}

func TestProjectIsGitRepository(t *testing.T) {
	// Create temporary directory structure for testing
	tempDir, err := os.MkdirTemp("", "project-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Test non-Git project
	nonGitProject := &Project{
		Path:         tempDir,
		Name:         "test",
		Organisation: "user",
	}

	if nonGitProject.IsGitRepository() {
		t.Error("IsGitRepository() should return false for non-Git directory")
	}

	// Create .git directory
	gitDir := filepath.Join(tempDir, ".git")
	err = os.Mkdir(gitDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create .git dir: %v", err)
	}

	// Test Git project
	gitProject := &Project{
		Path:         tempDir,
		Name:         "test",
		Organisation: "user",
	}

	if !gitProject.IsGitRepository() {
		t.Error("IsGitRepository() should return true for Git directory")
	}
}

func TestProjectGetGitStatus(t *testing.T) {
	// Create temporary directory structure for testing
	tempDir, err := os.MkdirTemp("", "project-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Test non-Git project
	nonGitProject := &Project{
		Path:         tempDir,
		Name:         "test",
		Organisation: "user",
	}

	status := nonGitProject.GetGitStatus()
	if status != GitStatusNotGit {
		t.Errorf("GetGitStatus() = %v, want %v for non-Git directory", status, GitStatusNotGit)
	}

	// Initialize a real Git repository
	_, err = git.PlainInit(tempDir, false)
	if err != nil {
		t.Fatalf("Failed to init Git repo: %v", err)
	}

	status = nonGitProject.GetGitStatus()
	if status != GitStatusValid {
		t.Errorf("GetGitStatus() = %v, want %v for valid Git repository", status, GitStatusValid)
	}
}

func TestWalk(t *testing.T) {
	// Create temporary directory structure for testing
	tempDir, err := os.MkdirTemp("", "project-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create test directory structure
	testStructure := []string{
		"user1/project1",
		"user1/project2",
		"user2/project3",
		"user1/project1/.git",
		"user2/project3/.git",
		"not-a-project",    // Should be ignored (wrong depth)
		"user3",            // Should be ignored (wrong depth)
	}

	for _, dir := range testStructure {
		fullPath := filepath.Join(tempDir, dir)
		err := os.MkdirAll(fullPath, 0755)
		if err != nil {
			t.Fatalf("Failed to create test directory %s: %v", fullPath, err)
		}
	}

	// Collect projects found by Walk
	var foundProjects []*Project
	err = Walk(tempDir, func(d fs.DirEntry, p *Project) error {
		foundProjects = append(foundProjects, p)
		return nil
	})

	if err != nil {
		t.Fatalf("Walk() failed: %v", err)
	}

	// Verify expected projects were found
	expectedProjects := map[string]bool{
		"user1/project1": false,
		"user1/project2": false,
		"user2/project3": false,
	}

	for _, p := range foundProjects {
		key := p.String()
		if _, exists := expectedProjects[key]; exists {
			expectedProjects[key] = true
		} else {
			t.Errorf("Unexpected project found: %s", key)
		}
	}

	// Check that all expected projects were found
	for project, found := range expectedProjects {
		if !found {
			t.Errorf("Expected project not found: %s", project)
		}
	}

	// Verify project details
	for _, p := range foundProjects {
		if p.Name == "" {
			t.Error("Project Name should not be empty")
		}
		if p.Organisation == "" {
			t.Error("Project Organisation should not be empty")
		}
		if p.Path == "" {
			t.Error("Project Path should not be empty")
		}
		if !strings.HasPrefix(p.Path, tempDir) {
			t.Errorf("Project Path should start with tempDir, got: %s", p.Path)
		}
	}
}

func TestWalkWithError(t *testing.T) {
	// Test walking a non-existent directory
	err := Walk("/non-existent-directory", func(d fs.DirEntry, p *Project) error {
		return nil
	})

	if err == nil {
		t.Error("Walk() should return error for non-existent directory")
	}
}

func TestWalkWithCallbackError(t *testing.T) {
	// Create temporary directory
	tempDir, err := os.MkdirTemp("", "project-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create test project
	projectPath := filepath.Join(tempDir, "user", "project")
	err = os.MkdirAll(projectPath, 0755)
	if err != nil {
		t.Fatalf("Failed to create test project: %v", err)
	}

	// Test callback that returns an error
	expectedError := "callback error"
	err = Walk(tempDir, func(d fs.DirEntry, p *Project) error {
		return errors.New(expectedError)
	})

	if err == nil {
		t.Error("Walk() should return error when callback returns error")
	}

	if !strings.Contains(err.Error(), expectedError) {
		t.Errorf("Walk() error should contain callback error, got: %v", err)
	}
}
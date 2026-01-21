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
			expected:    nil,
			wantErr:     true, // Empty project names are now rejected
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
		"not-a-project", // Should be ignored (wrong depth)
		"user3",         // Should be ignored (wrong depth)
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

func TestWalkExcludesDotDirectories(t *testing.T) {
	// Create temporary directory structure for testing
	tempDir, err := os.MkdirTemp("", "project-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create test directory structure including dot directories
	testStructure := []string{
		"user1/project1",
		"user2/project2",
		".workspace/user1/project1.feature", // Should be excluded
		".vscode/settings",                  // Should be excluded
		".git/hooks",                        // Should be excluded
		"user1/.hidden-project",             // Should be excluded
		"user3/normal-project",              // Should be included
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

	// Should only find normal projects, not those in dot directories
	expectedProjects := []string{"user1/project1", "user2/project2", "user3/normal-project"}

	if len(foundProjects) != len(expectedProjects) {
		t.Errorf("Expected %d projects, found %d", len(expectedProjects), len(foundProjects))
		for i, p := range foundProjects {
			t.Logf("Found project %d: %s", i, p.String())
		}
	}

	// Create a map of found projects for easy checking
	foundMap := make(map[string]bool)
	for _, p := range foundProjects {
		foundMap[p.String()] = true
	}

	// Verify each expected project was found
	for _, expected := range expectedProjects {
		if !foundMap[expected] {
			t.Errorf("Expected project %s was not found", expected)
		}
	}

	// Verify no dot directories were included
	for _, p := range foundProjects {
		if strings.HasPrefix(p.Organisation, ".") || strings.HasPrefix(p.Name, ".") {
			t.Errorf("Found project in dot directory: %s", p.String())
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

func TestFindFromPath(t *testing.T) {
	// Create temporary directory structure
	tempDir, err := os.MkdirTemp("", "project-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create test projects
	testProjects := []string{
		"user1/project1",
		"user2/project2",
		"org/deep-project/subdirectory",
		".workspace/user1/project1/feature-branch",
		".workspace/user2/project2",
		".workspace/user1",
	}

	for _, project := range testProjects {
		err := os.MkdirAll(filepath.Join(tempDir, project), 0755)
		if err != nil {
			t.Fatalf("Failed to create test directory: %v", err)
		}
	}

	tests := []struct {
		name        string
		path        string
		expected    *Project
		expectError bool
		errorMsg    string
	}{
		{
			name: "find project from exact project path",
			path: filepath.Join(tempDir, "user1/project1"),
			expected: &Project{
				Path:         filepath.Join(tempDir, "user1/project1"),
				Name:         "project1",
				Organisation: "user1",
			},
			expectError: false,
		},
		{
			name: "find project from subdirectory",
			path: filepath.Join(tempDir, "org/deep-project/subdirectory"),
			expected: &Project{
				Path:         filepath.Join(tempDir, "org/deep-project"),
				Name:         "deep-project",
				Organisation: "org",
			},
			expectError: false,
		},
		{
			name:        "path outside root directory",
			path:        "/tmp/outside",
			expectError: true,
			errorMsg:    "not inside projects root directory",
		},
		{
			name:        "path is root directory",
			path:        tempDir,
			expectError: true,
			errorMsg:    "path is the root directory",
		},
		{
			name:        "path with only organization",
			path:        filepath.Join(tempDir, "user1"),
			expectError: true,
			errorMsg:    "does not contain organization/project structure",
		},
		{
			name: "relative path resolution",
			path: filepath.Join(tempDir, "user1/project1/../../user2/project2"),
			expected: &Project{
				Path:         filepath.Join(tempDir, "user2/project2"),
				Name:         "project2",
				Organisation: "user2",
			},
			expectError: false,
		},
		{
			name: "find project from workspace directory",
			path: filepath.Join(tempDir, ".workspace/user1/project1/feature-branch"),
			expected: &Project{
				Path:         filepath.Join(tempDir, "user1/project1"),
				Name:         "project1",
				Organisation: "user1",
			},
			expectError: false,
		},
		{
			name: "find project from workspace root (no branch)",
			path: filepath.Join(tempDir, ".workspace/user2/project2"),
			expected: &Project{
				Path:         filepath.Join(tempDir, "user2/project2"),
				Name:         "project2",
				Organisation: "user2",
			},
			expectError: false,
		},
		{
			name:        "incomplete workspace path (org only)",
			path:        filepath.Join(tempDir, ".workspace/user1"),
			expectError: true,
			errorMsg:    "does not contain organization/project structure",
		},
		{
			name:        "workspace directory alone",
			path:        filepath.Join(tempDir, ".workspace"),
			expectError: true,
			errorMsg:    "does not contain organization/project structure",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			project, err := FindFromPath(tempDir, tt.path)

			if tt.expectError {
				if err == nil {
					t.Errorf("expected error but got none")
					return
				}
				if tt.errorMsg != "" && !strings.Contains(err.Error(), tt.errorMsg) {
					t.Errorf("expected error containing %q, got %q", tt.errorMsg, err.Error())
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if project == nil {
				t.Fatal("expected project but got nil")
			}

			if project.Path != tt.expected.Path {
				t.Errorf("path = %q, want %q", project.Path, tt.expected.Path)
			}
			if project.Name != tt.expected.Name {
				t.Errorf("name = %q, want %q", project.Name, tt.expected.Name)
			}
			if project.Organisation != tt.expected.Organisation {
				t.Errorf("organisation = %q, want %q", project.Organisation, tt.expected.Organisation)
			}
		})
	}
}

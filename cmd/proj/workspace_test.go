package main

import (
	"path/filepath"
	"testing"

	"github.com/gfanton/projects"
)

// mockLogger implements projects.Logger for testing
type mockLogger struct{}

func (m *mockLogger) Debug(msg string, args ...any) {}
func (m *mockLogger) Info(msg string, args ...any)  {}
func (m *mockLogger) Warn(msg string, args ...any)  {}
func (m *mockLogger) Error(msg string, args ...any) {}

func TestResolveProject(t *testing.T) {
	tempDir := t.TempDir()

	projectsCfg := &projects.Config{
		RootDir:  tempDir,
		RootUser: "testuser",
	}

	logger := &mockLogger{}

	tests := []struct {
		name        string
		projectStr  string
		expectedErr bool
		expected    projects.Project
	}{
		{
			name:       "explicit project with org",
			projectStr: "testorg/testproject",
			expected: projects.Project{
				Name:         "testproject",
				Organisation: "testorg",
				Path:         filepath.Join(tempDir, "testorg", "testproject"),
			},
		},
		{
			name:       "explicit project without org",
			projectStr: "testproject",
			expected: projects.Project{
				Name:         "testproject",
				Organisation: "testuser",
				Path:         filepath.Join(tempDir, "testuser", "testproject"),
			},
		},
		{
			name:        "invalid project name",
			projectStr:  "",
			expectedErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			proj, err := resolveProject(projectsCfg, logger, tt.projectStr)

			if tt.expectedErr {
				if err == nil {
					t.Error("expected error but got none")
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if proj.Name != tt.expected.Name {
				t.Errorf("project name = %q, want %q", proj.Name, tt.expected.Name)
			}
			if proj.Organisation != tt.expected.Organisation {
				t.Errorf("project org = %q, want %q", proj.Organisation, tt.expected.Organisation)
			}
		})
	}
}

func TestFindProjectFromPath(t *testing.T) {
	tempDir := t.TempDir()
	projectsCfg := &projects.Config{
		RootDir: tempDir,
	}

	logger := &mockLogger{}
	projectSvc := projects.NewProjectService(projectsCfg, logger)

	tests := []struct {
		name        string
		path        string
		expectedErr bool
		expected    projects.Project
	}{
		{
			name: "valid project path",
			path: filepath.Join(tempDir, "testorg", "testproject"),
			expected: projects.Project{
				Name:         "testproject",
				Organisation: "testorg",
				Path:         filepath.Join(tempDir, "testorg", "testproject"),
			},
		},
		{
			name: "nested project path",
			path: filepath.Join(tempDir, "testorg", "testproject", "subdir"),
			expected: projects.Project{
				Name:         "testproject",
				Organisation: "testorg",
				Path:         filepath.Join(tempDir, "testorg", "testproject"),
			},
		},
		{
			name:        "outside root directory",
			path:        "/tmp",
			expectedErr: true,
		},
		{
			name:        "incomplete path",
			path:        filepath.Join(tempDir, "testorg"),
			expectedErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			proj, err := projectSvc.FindFromPath(tt.path)

			if tt.expectedErr {
				if err == nil {
					t.Error("expected error but got none")
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if proj.Name != tt.expected.Name {
				t.Errorf("project name = %q, want %q", proj.Name, tt.expected.Name)
			}
			if proj.Organisation != tt.expected.Organisation {
				t.Errorf("project org = %q, want %q", proj.Organisation, tt.expected.Organisation)
			}
			if proj.Path != tt.expected.Path {
				t.Errorf("project path = %q, want %q", proj.Path, tt.expected.Path)
			}
		})
	}
}

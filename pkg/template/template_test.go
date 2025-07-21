package template

import (
	"strings"
	"testing"
)

func TestRender(t *testing.T) {
	tests := []struct {
		name         string
		templateName string
		data         Data
		expectError  bool
	}{
		{
			name:         "render zsh template",
			templateName: "zsh",
			data: Data{
				Exec: "/usr/local/bin/project",
			},
			expectError: false,
		},
		{
			name:         "render zsh with different exec path",
			templateName: "zsh",
			data: Data{
				Exec: "/custom/path/to/project",
			},
			expectError: false,
		},
		{
			name:         "render non-existent template",
			templateName: "nonexistent",
			data: Data{
				Exec: "/usr/local/bin/project",
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := Render(tt.templateName, tt.data)

			if tt.expectError {
				if err == nil {
					t.Errorf("Render() expected error but got none")
				}
				return
			}

			if err != nil {
				t.Fatalf("Render() failed: %v", err)
			}

			if result == "" {
				t.Error("Render() returned empty result")
			}

			// Verify exec path substitution
			if !strings.Contains(result, tt.data.Exec) {
				t.Errorf("Template should contain exec path %q", tt.data.Exec)
			}
		})
	}
}

func TestRenderBasicStructure(t *testing.T) {
	data := Data{
		Exec: "/test/bin/project",
	}

	result, err := Render("zsh", data)
	if err != nil {
		t.Fatalf("Render() failed: %v", err)
	}

	// Test that basic functions exist (without being overly prescriptive)
	basicElements := []string{
		"function __project_pwd()",
		"function __project_cd()",  
		"function __project_p()",
		"function p()",
		"function _p()",
	}

	for _, element := range basicElements {
		if !strings.Contains(result, element) {
			t.Errorf("Template should contain: %s", element)
		}
	}
}

func TestRenderWithEmptyData(t *testing.T) {
	data := Data{
		Exec: "", // Empty exec path
	}

	result, err := Render("zsh", data)
	if err != nil {
		t.Fatalf("Render() should handle empty data, got error: %v", err)
	}

	if result == "" {
		t.Error("Render() should not return empty result even with empty data")
	}
}

func TestRenderErrorHandling(t *testing.T) {
	tests := []struct {
		name         string
		templateName string
		expectError  string
	}{
		{
			name:         "template not found",
			templateName: "nonexistent",
			expectError:  "failed to read template",
		},
		{
			name:         "empty template name",
			templateName: "",
			expectError:  "failed to read template",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data := Data{Exec: "/test/bin/project"}
			_, err := Render(tt.templateName, data)

			if err == nil {
				t.Error("Render() should return error for invalid template")
			}

			if !strings.Contains(err.Error(), tt.expectError) {
				t.Errorf("Error should contain %q, got: %v", tt.expectError, err)
			}
		})
	}
}

func TestRenderSpecialCharacters(t *testing.T) {
	// Test rendering with special characters in the exec path
	data := Data{
		Exec: "/path with spaces/project-tool",
	}

	result, err := Render("zsh", data)
	if err != nil {
		t.Fatalf("Render() failed with special characters: %v", err)
	}

	if !strings.Contains(result, "/path with spaces/project-tool") {
		t.Error("Template should handle paths with special characters")
	}
}


package template

import (
	"strings"
	"testing"
)

func TestRender(t *testing.T) {
	tests := []struct {
		name           string
		templateName   string
		data           Data
		expectError    bool
		expectContains []string
	}{
		{
			name:         "render zsh template",
			templateName: "zsh",
			data: Data{
				Exec: "/usr/local/bin/project",
			},
			expectError: false,
			expectContains: []string{
				"# compdef p",
				"__project_p_prefix",
				"__project_pwd",
				"__project_cd",
				"__project_p",
				"/usr/local/bin/project",
				"alias p=__project_p",
			},
		},
		{
			name:         "render zsh with different exec path",
			templateName: "zsh",
			data: Data{
				Exec: "/custom/path/to/project",
			},
			expectError: false,
			expectContains: []string{
				"/custom/path/to/project",
				"# compdef p",
			},
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

			// Check that expected content is present
			for _, expected := range tt.expectContains {
				if !strings.Contains(result, expected) {
					t.Errorf("Render() result should contain %q", expected)
				}
			}
		})
	}
}

func TestRenderZshTemplateStructure(t *testing.T) {
	data := Data{
		Exec: "/test/bin/project",
	}

	result, err := Render("zsh", data)
	if err != nil {
		t.Fatalf("Render() failed: %v", err)
	}

	// Verify the template structure has the key components
	requiredFunctions := []string{
		"function __project_pwd()",
		"function __project_cd()",
		"function __project_p()",
		"function __project_p_complete()",
	}

	for _, fn := range requiredFunctions {
		if !strings.Contains(result, fn) {
			t.Errorf("Template should contain function definition: %s", fn)
		}
	}

	// Verify the exec path is properly substituted
	if !strings.Contains(result, "/test/bin/project") {
		t.Error("Template should contain the provided exec path")
	}

	// Verify zsh-specific elements
	zshElements := []string{
		"# compdef p",
		"[[ -o zle ]]",
		"compdef __project_p_complete __project_p",
		"alias p=__project_p",
	}

	for _, element := range zshElements {
		if !strings.Contains(result, element) {
			t.Errorf("Template should contain zsh element: %s", element)
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

	// Should still contain template structure
	if !strings.Contains(result, "function __project_p()") {
		t.Error("Template should still contain function definitions with empty data")
	}
}

func TestDataStruct(t *testing.T) {
	// Test that Data struct can be created and fields accessed
	data := Data{
		Exec: "/test/path",
	}

	if data.Exec != "/test/path" {
		t.Errorf("Data.Exec = %s, want /test/path", data.Exec)
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

func TestRenderConsistency(t *testing.T) {
	// Test that rendering the same template with the same data produces consistent results
	data := Data{
		Exec: "/usr/bin/project",
	}

	result1, err1 := Render("zsh", data)
	if err1 != nil {
		t.Fatalf("First render failed: %v", err1)
	}

	result2, err2 := Render("zsh", data)
	if err2 != nil {
		t.Fatalf("Second render failed: %v", err2)
	}

	if result1 != result2 {
		t.Error("Render() should produce consistent results for the same input")
	}
}
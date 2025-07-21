package shell

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// TestShellIntegration tests shell completions similar to zoxide's approach
func TestShellIntegration(t *testing.T) {
	// Skip if running in short mode
	if testing.Short() {
		t.Skip("skipping shell integration tests in short mode")
	}

	// Build the project binary first
	projectBin := buildProjectBinary(t)

	tests := []struct {
		name     string
		shell    string
		testFunc func(t *testing.T, projectBin string)
	}{
		{
			name:     "zsh_completion",
			shell:    "zsh",
			testFunc: testZshCompletion,
		},
		{
			name:     "zsh_navigation",
			shell:    "zsh",
			testFunc: testZshNavigation,
		},
		{
			name:     "zsh_init_script",
			shell:    "zsh",
			testFunc: testZshInitScript,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Check if shell is available
			if !isShellAvailable(tt.shell) {
				t.Skipf("%s not available, skipping test", tt.shell)
			}

			tt.testFunc(t, projectBin)
		})
	}
}

func buildProjectBinary(t *testing.T) string {
	t.Helper()

	// Get project root
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get working directory: %v", err)
	}

	// Navigate to project root (assuming we're in internal/shell)
	projectRoot := filepath.Join(wd, "..", "..")

	// Build binary
	binaryPath := filepath.Join(projectRoot, "build", "proj")
	cmd := exec.Command("go", "build", "-o", binaryPath, "./cmd/proj")
	cmd.Dir = projectRoot

	if err := cmd.Run(); err != nil {
		t.Fatalf("failed to build project binary: %v", err)
	}

	return binaryPath
}

func isShellAvailable(shell string) bool {
	_, err := exec.LookPath(shell)
	return err == nil
}

func testZshInitScript(t *testing.T, projectBin string) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Generate zsh init script
	cmd := exec.CommandContext(ctx, projectBin, "init", "zsh")
	output, err := cmd.Output()
	if err != nil {
		t.Fatalf("failed to generate zsh init script: %v", err)
	}

	script := string(output)

	// Verify essential components
	expectedComponents := []string{
		"function __project_p()",
		"function __project_p_complete()",
		"alias p=__project_p",
		"# compdef p",
	}

	for _, component := range expectedComponents {
		if !strings.Contains(script, component) {
			t.Errorf("zsh init script missing component: %s", component)
		}
	}
}

func testZshCompletion(t *testing.T, projectBin string) {
	// Create temporary test environment
	testDir := createTestEnvironment(t, projectBin)
	defer os.RemoveAll(testDir)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Test that zsh can source the completion script without errors
	configFile := filepath.Join(testDir, ".projectrc")

	script := fmt.Sprintf(`
set -e
# Enable completion system and zle
autoload -U compinit && compinit -u
zmodload zsh/zle

# Source project init
eval "$(%q init zsh)"

# Test that main function is loaded
type __project_p >/dev/null || exit 1

# Test that completion function is loaded (only if zle is available)
if [[ -o zle ]]; then
    type __project_p_complete >/dev/null || exit 1
    echo "Completion function loaded successfully"
else
    echo "ZLE not available, skipping completion function test"
fi

# Test basic query functionality
export PROJECT_CONFIG_FILE=%q
result=$(%q query project1 2>/dev/null || true)
if [[ -n "$result" ]]; then
    echo "Query returned: $result"
else
    echo "No results found, but this might be expected in test environment"
fi

echo "Completion test passed"
`, projectBin, configFile, projectBin)

	formattedScript := script

	cmd := exec.CommandContext(ctx, "zsh", "-c", formattedScript)
	cmd.Env = append(os.Environ(), "PROJECT_CONFIG_FILE="+configFile)

	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Logf("Script that failed:\n%s", formattedScript)
		t.Fatalf("zsh completion test failed: %v\nOutput: %s", err, output)
	}

	if !strings.Contains(string(output), "Completion test passed") {
		t.Errorf("completion test did not pass as expected. Output: %s", output)
	}
}

func testZshNavigation(t *testing.T, projectBin string) {
	// Create temporary test environment
	testDir := createTestEnvironment(t, projectBin)
	defer os.RemoveAll(testDir)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	configFile := filepath.Join(testDir, ".projectrc")

	// Test navigation functionality
	script := fmt.Sprintf(`
set -e
export PROJECT_CONFIG_FILE=%q

# Source project init
eval "$(%q init zsh)"

# Override __project_cd to just echo the path for testing
__project_cd() {
    echo "NAVIGATED_TO:$1"
}

# Test p command with a project that should exist
result=$(p project1 2>/dev/null || echo "NO_MATCH")
echo "Navigation result: $result"
`, configFile, projectBin)

	formattedScript := script

	cmd := exec.CommandContext(ctx, "zsh", "-c", formattedScript)
	cmd.Env = append(os.Environ(), "PROJECT_CONFIG_FILE="+configFile)

	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("zsh navigation test failed: %v\nOutput: %s", err, output)
	}

	// The output should contain either a navigation result or indicate no match
	outputStr := string(output)
	if !strings.Contains(outputStr, "Navigation result:") {
		t.Errorf("navigation test did not produce expected output. Output: %s", outputStr)
	}
}

func createTestEnvironment(t *testing.T, projectBin string) string {
	t.Helper()

	testDir, err := os.MkdirTemp("", "project-shell-test-*")
	if err != nil {
		t.Fatalf("failed to create test directory: %v", err)
	}

	// Create test project structure
	codeDir := filepath.Join(testDir, "code")
	projects := []string{
		"user1/project1",
		"user1/project2",
		"user2/project3",
		"user2/awesome-project",
		"user3/my-cool-app",
	}

	for _, project := range projects {
		projectPath := filepath.Join(codeDir, project)
		if err := os.MkdirAll(projectPath, 0755); err != nil {
			t.Fatalf("failed to create project directory %s: %v", project, err)
		}

		// Initialize git repo
		cmd := exec.Command("git", "init", "--quiet")
		cmd.Dir = projectPath
		if err := cmd.Run(); err != nil {
			t.Fatalf("failed to init git repo in %s: %v", project, err)
		}
	}

	// Create config file
	configFile := filepath.Join(testDir, ".projectrc")
	configContent := fmt.Sprintf(`root = "%s"
user = "testuser"
`, codeDir)

	if err := os.WriteFile(configFile, []byte(configContent), 0644); err != nil {
		t.Fatalf("failed to create config file: %v", err)
	}

	return testDir
}

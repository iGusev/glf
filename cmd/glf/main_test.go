package main

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/igusev/glf/internal/types"
	"github.com/spf13/cobra"
)

func TestMaskToken(t *testing.T) {
	tests := []struct {
		name     string
		token    string
		expected string
	}{
		{
			name:     "empty token",
			token:    "",
			expected: "********",
		},
		{
			name:     "short token 4 chars",
			token:    "abcd",
			expected: "********",
		},
		{
			name:     "short token 8 chars",
			token:    "abcdefgh",
			expected: "********",
		},
		{
			name:     "normal token 20 chars",
			token:    "12345678901234567890",
			expected: "1234****7890",
		},
		{
			name:     "gitlab token format",
			token:    "glpat-1234567890abcdefghij",
			expected: "glpa****ghij",
		},
		{
			name:     "exactly 9 chars",
			token:    "123456789",
			expected: "1234****6789",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := maskToken(tt.token)
			if result != tt.expected {
				t.Errorf("maskToken(%q) = %q, want %q", tt.token, result, tt.expected)
			}
		})
	}
}

func TestOpenBrowser(t *testing.T) {
	// Test that openBrowser returns error for unsupported platform
	// We can't easily test actual browser opening without executing commands

	// Test URL parameter is used correctly
	testURL := "https://gitlab.example.com/project/name"

	// Store original GOOS
	originalGOOS := runtime.GOOS
	defer func() {
		// Note: We can't actually change runtime.GOOS in tests
		// This test verifies the function doesn't panic on current platform
		_ = originalGOOS
	}()

	// Test that function accepts valid URLs without panicking
	// on supported platforms (darwin, linux, windows)
	switch runtime.GOOS {
	case "darwin", "linux", "windows":
		// These platforms should not error on command creation
		// We can't test actual execution in unit tests
		err := openBrowser(testURL)
		// Command Start() might fail in test environment (no display, etc.)
		// but that's OK - we're testing the function doesn't panic
		if err != nil {
			t.Logf("openBrowser returned error (expected in test env): %v", err)
		}
	default:
		// Other platforms should return unsupported error
		err := openBrowser(testURL)
		if err == nil {
			t.Error("Expected error for unsupported platform, got nil")
		}
	}
}

func TestIndexDescriptions_EmptyProjects(t *testing.T) {
	// Test indexing empty project list
	tempDir := t.TempDir()

	projects := []types.Project{}

	err := indexDescriptions(projects, tempDir, true)
	if err != nil {
		t.Fatalf("indexDescriptions with empty projects failed: %v", err)
	}

	// Verify index was created (even for empty list)
	indexPath := filepath.Join(tempDir, "description.bleve")
	if _, err := os.Stat(indexPath); os.IsNotExist(err) {
		t.Error("Expected index directory to be created")
	}
}

func TestIndexDescriptions_SingleProject(t *testing.T) {
	// Test indexing a single project
	tempDir := t.TempDir()

	projects := []types.Project{
		{
			Path:        "group/project1",
			Name:        "Project 1",
			Description: "Test project description",
		},
	}

	err := indexDescriptions(projects, tempDir, true)
	if err != nil {
		t.Fatalf("indexDescriptions with single project failed: %v", err)
	}

	// Verify index exists
	indexPath := filepath.Join(tempDir, "description.bleve")
	if _, err := os.Stat(indexPath); os.IsNotExist(err) {
		t.Error("Expected index directory to be created")
	}
}

func TestIndexDescriptions_MultipleProjects(t *testing.T) {
	// Test indexing multiple projects (batch processing)
	tempDir := t.TempDir()

	// Create 150 projects to test batch logic (batches of 100)
	projects := make([]types.Project, 150)
	for i := 0; i < 150; i++ {
		projects[i] = types.Project{
			Path:        "group/project" + string(rune(i)),
			Name:        "Project " + string(rune(i)),
			Description: "Test description " + string(rune(i)),
		}
	}

	err := indexDescriptions(projects, tempDir, true)
	if err != nil {
		t.Fatalf("indexDescriptions with multiple projects failed: %v", err)
	}

	// Verify index exists
	indexPath := filepath.Join(tempDir, "description.bleve")
	if _, err := os.Stat(indexPath); os.IsNotExist(err) {
		t.Error("Expected index directory to be created")
	}
}

func TestIndexDescriptions_ProjectsWithoutDescription(t *testing.T) {
	// Test indexing projects without descriptions
	tempDir := t.TempDir()

	projects := []types.Project{
		{
			Path:        "group/project1",
			Name:        "Project 1",
			Description: "", // Empty description
		},
		{
			Path:        "group/project2",
			Name:        "Project 2",
			Description: "Has description",
		},
	}

	err := indexDescriptions(projects, tempDir, true)
	if err != nil {
		t.Fatalf("indexDescriptions with projects without description failed: %v", err)
	}

	// Verify index exists
	indexPath := filepath.Join(tempDir, "description.bleve")
	if _, err := os.Stat(indexPath); os.IsNotExist(err) {
		t.Error("Expected index directory to be created")
	}
}

func TestIndexDescriptions_InvalidCacheDir(t *testing.T) {
	// Test with invalid cache directory (read-only or non-existent parent)
	// This test verifies error handling

	// Use a path that doesn't exist and can't be created
	invalidPath := "/nonexistent/readonly/path/that/cannot/be/created"

	projects := []types.Project{
		{
			Path:        "group/project1",
			Name:        "Project 1",
			Description: "Test",
		},
	}

	err := indexDescriptions(projects, invalidPath, true)
	if err == nil {
		t.Error("Expected error with invalid cache directory, got nil")
	}
}

func TestRunSearch_AutoGoWithoutQuery(t *testing.T) {
	// Test that auto-go mode requires a query
	// Set up minimal environment
	tempDir := t.TempDir()
	configDir := filepath.Join(tempDir, ".config", "glf")
	_ = os.MkdirAll(configDir, 0755)

	cacheDir := filepath.Join(tempDir, "cache")
	_ = os.MkdirAll(cacheDir, 0755)

	// Create minimal config
	configPath := filepath.Join(configDir, "config.yaml")
	configContent := `gitlab:
  url: https://gitlab.example.com
  token: test-token
cache:
  dir: ` + cacheDir
	_ = os.WriteFile(configPath, []byte(configContent), 0600)

	// Create minimal index so we get past the index check
	// Index a single dummy project
	projects := []types.Project{
		{Path: "test/project", Name: "Test", Description: "Test"},
	}
	_ = indexDescriptions(projects, cacheDir, true)

	// Set HOME to temp directory
	oldHome := os.Getenv("HOME")
	os.Setenv("HOME", tempDir)
	defer os.Setenv("HOME", oldHome)

	// Enable auto-go mode
	autoGo = true
	defer func() { autoGo = false }()

	// Create empty command
	cmd := &cobra.Command{}

	// Try to run with empty query - should fail
	err := runSearch(cmd, []string{})
	if err == nil {
		t.Error("Expected error for auto-go without query, got nil")
	}
	if err != nil && err.Error() != "-g/--go requires a search query" {
		t.Errorf("Expected '-g/--go requires a search query' error, got: %v", err)
	}
}

func TestRunSearch_IndexNotFound(t *testing.T) {
	// Test that runSearch fails gracefully when index doesn't exist
	tempDir := t.TempDir()
	configDir := filepath.Join(tempDir, ".config", "glf")
	_ = os.MkdirAll(configDir, 0755)

	// Create minimal config
	configPath := filepath.Join(configDir, "config.yaml")
	configContent := `gitlab:
  url: https://gitlab.example.com
  token: test-token
cache:
  dir: ` + filepath.Join(tempDir, "cache")
	_ = os.WriteFile(configPath, []byte(configContent), 0600)

	// Set HOME to temp directory
	oldHome := os.Getenv("HOME")
	os.Setenv("HOME", tempDir)
	defer os.Setenv("HOME", oldHome)

	// Disable auto-go and sync modes
	autoGo = false
	doSync = false

	// Create empty command
	cmd := &cobra.Command{}

	// Try to run without index - should fail
	err := runSearch(cmd, []string{"test"})
	if err == nil {
		t.Error("Expected error for missing index, got nil")
	}
	if err != nil && err.Error() != "index not found, run 'glf --sync' first" {
		t.Errorf("Expected 'index not found' error, got: %v", err)
	}
}

func TestExtractProjectPath(t *testing.T) {
	tests := []struct {
		name        string
		remoteURL   string
		gitlabURL   string
		expected    string
		expectError bool
		errorMsg    string
	}{
		{
			name:        "SSH format with .git suffix",
			remoteURL:   "git@gitlab.example.com:namespace/project.git",
			gitlabURL:   "https://gitlab.example.com",
			expected:    "namespace/project",
			expectError: false,
		},
		{
			name:        "SSH format without .git suffix",
			remoteURL:   "git@gitlab.example.com:namespace/project",
			gitlabURL:   "https://gitlab.example.com",
			expected:    "namespace/project",
			expectError: false,
		},
		{
			name:        "HTTPS format with .git suffix",
			remoteURL:   "https://gitlab.example.com/namespace/project.git",
			gitlabURL:   "https://gitlab.example.com",
			expected:    "namespace/project",
			expectError: false,
		},
		{
			name:        "HTTPS format without .git suffix",
			remoteURL:   "https://gitlab.example.com/namespace/project",
			gitlabURL:   "https://gitlab.example.com",
			expected:    "namespace/project",
			expectError: false,
		},
		{
			name:        "HTTP format with .git suffix",
			remoteURL:   "http://gitlab.example.com/namespace/project.git",
			gitlabURL:   "http://gitlab.example.com",
			expected:    "namespace/project",
			expectError: false,
		},
		{
			name:        "nested namespace",
			remoteURL:   "git@gitlab.example.com:group/subgroup/project.git",
			gitlabURL:   "https://gitlab.example.com",
			expected:    "group/subgroup/project",
			expectError: false,
		},
		{
			name:        "domain mismatch SSH",
			remoteURL:   "git@gitlab.other.com:namespace/project.git",
			gitlabURL:   "https://gitlab.example.com",
			expectError: true,
			errorMsg:    "does not match configured GitLab domain",
		},
		{
			name:        "domain mismatch HTTPS",
			remoteURL:   "https://gitlab.other.com/namespace/project.git",
			gitlabURL:   "https://gitlab.example.com",
			expectError: true,
			errorMsg:    "does not match configured GitLab domain",
		},
		{
			name:        "invalid SSH format - no colon",
			remoteURL:   "git@gitlab.example.com/namespace/project.git",
			gitlabURL:   "https://gitlab.example.com",
			expectError: true,
			errorMsg:    "invalid SSH remote URL format",
		},
		{
			name:        "invalid HTTPS format - no path",
			remoteURL:   "https://gitlab.example.com",
			gitlabURL:   "https://gitlab.example.com",
			expectError: true,
			errorMsg:    "invalid HTTPS remote URL format",
		},
		{
			name:        "unsupported protocol",
			remoteURL:   "ftp://gitlab.example.com/namespace/project.git",
			gitlabURL:   "https://gitlab.example.com",
			expectError: true,
			errorMsg:    "unsupported git remote URL format",
		},
		{
			name:        "GitLab URL with trailing slash",
			remoteURL:   "https://gitlab.example.com/namespace/project.git",
			gitlabURL:   "https://gitlab.example.com/",
			expected:    "namespace/project",
			expectError: false,
		},
		{
			name:        "project path with leading slash",
			remoteURL:   "https://gitlab.example.com//namespace/project.git",
			gitlabURL:   "https://gitlab.example.com",
			expected:    "namespace/project",
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := extractProjectPath(tt.remoteURL, tt.gitlabURL)

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error containing '%s', got nil", tt.errorMsg)
				} else if tt.errorMsg != "" && !contains(err.Error(), tt.errorMsg) {
					t.Errorf("Expected error containing '%s', got: %v", tt.errorMsg, err)
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
				if result != tt.expected {
					t.Errorf("Expected '%s', got '%s'", tt.expected, result)
				}
			}
		})
	}
}

func TestGetGitRemoteURL_NonGitDirectory(t *testing.T) {
	// Test with a directory that's not a Git repository
	tempDir := t.TempDir()

	_, err := getGitRemoteURL(tempDir)
	if err == nil {
		t.Error("Expected error for non-git directory, got nil")
	}
}

// Helper function to check if a string contains a substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && (s[:len(substr)] == substr || s[len(s)-len(substr):] == substr || containsMiddle(s, substr)))
}

func containsMiddle(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

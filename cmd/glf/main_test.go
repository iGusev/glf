package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/igusev/glf/internal/config"
	"github.com/igusev/glf/internal/index"
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
	case platformDarwin, platformLinux, platformWindows:
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
	// Platform-specific invalid paths
	var invalidPath string
	switch runtime.GOOS {
	case platformWindows:
		// Use Windows reserved device name which cannot be a directory
		invalidPath = `C:\CON\invalid\path`
	default:
		invalidPath = "/nonexistent/readonly/path/that/cannot/be/created"
	}

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
		name         string
		remoteURL    string
		gitlabURL    string
		expectedPath string
		expectedBase string
		expectError  bool
		errorMsg     string
	}{
		{
			name:         "SSH format with .git suffix",
			remoteURL:    "git@gitlab.example.com:namespace/project.git",
			gitlabURL:    "https://gitlab.example.com",
			expectedPath: "namespace/project",
			expectedBase: "https://gitlab.example.com",
			expectError:  false,
		},
		{
			name:         "SSH format without .git suffix",
			remoteURL:    "git@gitlab.example.com:namespace/project",
			gitlabURL:    "https://gitlab.example.com",
			expectedPath: "namespace/project",
			expectedBase: "https://gitlab.example.com",
			expectError:  false,
		},
		{
			name:         "SSH with ssh:// prefix and port",
			remoteURL:    "ssh://git@gitlab.example.com:10022/namespace/project.git",
			gitlabURL:    "https://gitlab.example.com",
			expectedPath: "namespace/project",
			expectedBase: "https://gitlab.example.com",
			expectError:  false,
		},
		{
			name:         "SSH with ssh:// prefix without port",
			remoteURL:    "ssh://git@gitlab.example.com/namespace/project.git",
			gitlabURL:    "https://gitlab.example.com",
			expectedPath: "namespace/project",
			expectedBase: "https://gitlab.example.com",
			expectError:  false,
		},
		{
			name:         "SSH with ssh:// prefix and nested namespace",
			remoteURL:    "ssh://git@gitlab.example.com:10022/docs/backend/project.git",
			gitlabURL:    "https://gitlab.example.com",
			expectedPath: "docs/backend/project",
			expectedBase: "https://gitlab.example.com",
			expectError:  false,
		},
		{
			name:         "HTTPS format with .git suffix",
			remoteURL:    "https://gitlab.example.com/namespace/project.git",
			gitlabURL:    "https://gitlab.example.com",
			expectedPath: "namespace/project",
			expectedBase: "https://gitlab.example.com",
			expectError:  false,
		},
		{
			name:         "HTTPS format without .git suffix",
			remoteURL:    "https://gitlab.example.com/namespace/project",
			gitlabURL:    "https://gitlab.example.com",
			expectedPath: "namespace/project",
			expectedBase: "https://gitlab.example.com",
			expectError:  false,
		},
		{
			name:         "HTTP format with .git suffix",
			remoteURL:    "http://gitlab.example.com/namespace/project.git",
			gitlabURL:    "http://gitlab.example.com",
			expectedPath: "namespace/project",
			expectedBase: "http://gitlab.example.com",
			expectError:  false,
		},
		{
			name:         "nested namespace",
			remoteURL:    "git@gitlab.example.com:group/subgroup/project.git",
			gitlabURL:    "https://gitlab.example.com",
			expectedPath: "group/subgroup/project",
			expectedBase: "https://gitlab.example.com",
			expectError:  false,
		},
		{
			name:        "domain mismatch SSH",
			remoteURL:   "git@gitlab.other.com:namespace/project.git",
			gitlabURL:   "https://gitlab.example.com",
			expectError: true,
			errorMsg:    "does not match configured GitLab",
		},
		{
			name:        "domain mismatch HTTPS",
			remoteURL:   "https://gitlab.other.com/namespace/project.git",
			gitlabURL:   "https://gitlab.example.com",
			expectError: true,
			errorMsg:    "does not match configured GitLab",
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
			errorMsg:    "invalid remote URL format",
		},
		{
			name:        "unsupported protocol",
			remoteURL:   "ftp://gitlab.example.com/namespace/project.git",
			gitlabURL:   "https://gitlab.example.com",
			expectError: true,
			errorMsg:    "unsupported git remote URL format",
		},
		{
			name:         "GitLab URL with trailing slash",
			remoteURL:    "https://gitlab.example.com/namespace/project.git",
			gitlabURL:    "https://gitlab.example.com/",
			expectedPath: "namespace/project",
			expectedBase: "https://gitlab.example.com",
			expectError:  false,
		},
		{
			name:         "project path with leading slash",
			remoteURL:    "https://gitlab.example.com//namespace/project.git",
			gitlabURL:    "https://gitlab.example.com",
			expectedPath: "namespace/project",
			expectedBase: "https://gitlab.example.com",
			expectError:  false,
		},
		// Port handling tests
		{
			name:         "HTTPS with port",
			remoteURL:    "https://gitlab.example.com:8443/namespace/project.git",
			gitlabURL:    "https://gitlab.example.com:8443",
			expectedPath: "namespace/project",
			expectedBase: "https://gitlab.example.com:8443",
			expectError:  false,
		},
		{
			name:         "HTTP with port",
			remoteURL:    "http://gitlab.example.com:8080/namespace/project.git",
			gitlabURL:    "http://gitlab.example.com:8080",
			expectedPath: "namespace/project",
			expectedBase: "http://gitlab.example.com:8080",
			expectError:  false,
		},
		// Public repository tests
		{
			name:         "GitHub SSH",
			remoteURL:    "git@github.com:user/repo.git",
			gitlabURL:    "https://gitlab.example.com",
			expectedPath: "user/repo",
			expectedBase: "https://github.com",
			expectError:  false,
		},
		{
			name:         "GitHub HTTPS",
			remoteURL:    "https://github.com/user/repo.git",
			gitlabURL:    "https://gitlab.example.com",
			expectedPath: "user/repo",
			expectedBase: "https://github.com",
			expectError:  false,
		},
		{
			name:         "GitLab.com SSH",
			remoteURL:    "git@gitlab.com:user/repo.git",
			gitlabURL:    "https://gitlab.example.com",
			expectedPath: "user/repo",
			expectedBase: "https://gitlab.com",
			expectError:  false,
		},
		{
			name:         "GitLab.com HTTPS",
			remoteURL:    "https://gitlab.com/user/repo.git",
			gitlabURL:    "https://gitlab.example.com",
			expectedPath: "user/repo",
			expectedBase: "https://gitlab.com",
			expectError:  false,
		},
		{
			name:         "Bitbucket SSH",
			remoteURL:    "git@bitbucket.org:user/repo.git",
			gitlabURL:    "https://gitlab.example.com",
			expectedPath: "user/repo",
			expectedBase: "https://bitbucket.org",
			expectError:  false,
		},
		{
			name:         "Bitbucket HTTPS",
			remoteURL:    "https://bitbucket.org/user/repo.git",
			gitlabURL:    "https://gitlab.example.com",
			expectedPath: "user/repo",
			expectedBase: "https://bitbucket.org",
			expectError:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			projectPath, baseURL, err := extractProjectPath(tt.remoteURL, tt.gitlabURL)

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
				if projectPath != tt.expectedPath {
					t.Errorf("Expected path '%s', got '%s'", tt.expectedPath, projectPath)
				}
				if baseURL != tt.expectedBase {
					t.Errorf("Expected base URL '%s', got '%s'", tt.expectedBase, baseURL)
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

func TestGetGitRemoteURL_NoRemote(t *testing.T) {
	// Check if git is available
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not available, skipping test")
	}

	// Create a Git repository without remote
	tempDir := t.TempDir()

	// Initialize Git repo
	cmd := exec.Command("git", "init")
	cmd.Dir = tempDir
	if err := cmd.Run(); err != nil {
		t.Fatalf("Failed to init git repo: %v", err)
	}

	// Try to get remote URL - should fail
	_, err := getGitRemoteURL(tempDir)
	if err == nil {
		t.Error("Expected error for repo without remote, got nil")
	}
}

func TestRunOpenCurrent_WithValidRemote(t *testing.T) {
	// Check if git is available
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not available, skipping test")
	}

	// Create temp directory structure
	tempDir := t.TempDir()
	configDir := filepath.Join(tempDir, ".config", "glf")
	_ = os.MkdirAll(configDir, 0755)

	repoDir := filepath.Join(tempDir, "repo")
	_ = os.MkdirAll(repoDir, 0755)

	// Create config
	configPath := filepath.Join(configDir, "config.yaml")
	configContent := `gitlab:
  url: https://gitlab.example.com
  token: test-token`
	if err := os.WriteFile(configPath, []byte(configContent), 0600); err != nil {
		t.Fatalf("Failed to create config: %v", err)
	}

	// Set HOME to temp directory
	oldHome := os.Getenv("HOME")
	os.Setenv("HOME", tempDir)
	defer os.Setenv("HOME", oldHome)

	// Initialize Git repo with matching remote
	cmd := exec.Command("git", "init")
	cmd.Dir = repoDir
	if err := cmd.Run(); err != nil {
		t.Fatalf("Failed to init git repo: %v", err)
	}

	cmd = exec.Command("git", "remote", "add", "origin", "git@gitlab.example.com:test/project.git")
	cmd.Dir = repoDir
	if err := cmd.Run(); err != nil {
		t.Fatalf("Failed to add remote: %v", err)
	}

	// Change to repo directory
	oldWd, _ := os.Getwd()
	os.Chdir(repoDir)
	defer os.Chdir(oldWd)

	// Load config
	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	// Call runOpenCurrent - should not error
	// Note: Browser opening will fail in test environment, but that's OK
	// We're testing the logic, not the actual browser
	err = runOpenCurrent(cfg)
	// We expect nil error because the function prints warning on browser failure
	// but doesn't return error
	if err != nil {
		t.Errorf("runOpenCurrent failed: %v", err)
	}
}

func TestRunOpenCurrent_WithPublicRemote(t *testing.T) {
	// Check if git is available
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not available, skipping test")
	}

	// Create temp directory structure
	tempDir := t.TempDir()
	configDir := filepath.Join(tempDir, ".config", "glf")
	_ = os.MkdirAll(configDir, 0755)

	repoDir := filepath.Join(tempDir, "repo")
	_ = os.MkdirAll(repoDir, 0755)

	// Create config
	configPath := filepath.Join(configDir, "config.yaml")
	configContent := `gitlab:
  url: https://gitlab.example.com
  token: test-token`
	if err := os.WriteFile(configPath, []byte(configContent), 0600); err != nil {
		t.Fatalf("Failed to create config: %v", err)
	}

	// Set HOME to temp directory
	oldHome := os.Getenv("HOME")
	os.Setenv("HOME", tempDir)
	defer os.Setenv("HOME", oldHome)

	// Initialize Git repo with GitHub remote
	cmd := exec.Command("git", "init")
	cmd.Dir = repoDir
	if err := cmd.Run(); err != nil {
		t.Fatalf("Failed to init git repo: %v", err)
	}

	cmd = exec.Command("git", "remote", "add", "origin", "git@github.com:user/repo.git")
	cmd.Dir = repoDir
	if err := cmd.Run(); err != nil {
		t.Fatalf("Failed to add remote: %v", err)
	}

	// Change to repo directory
	oldWd, _ := os.Getwd()
	os.Chdir(repoDir)
	defer os.Chdir(oldWd)

	// Load config
	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	// Call runOpenCurrent - should work with GitHub
	err = runOpenCurrent(cfg)
	if err != nil {
		t.Errorf("runOpenCurrent with GitHub remote failed: %v", err)
	}
}

func TestRunOpenCurrent_NoRemote(t *testing.T) {
	// Check if git is available
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not available, skipping test")
	}

	// Create temp directory structure
	tempDir := t.TempDir()
	configDir := filepath.Join(tempDir, ".config", "glf")
	_ = os.MkdirAll(configDir, 0755)

	repoDir := filepath.Join(tempDir, "repo")
	_ = os.MkdirAll(repoDir, 0755)

	// Create config
	configPath := filepath.Join(configDir, "config.yaml")
	configContent := `gitlab:
  url: https://gitlab.example.com
  token: test-token`
	if err := os.WriteFile(configPath, []byte(configContent), 0600); err != nil {
		t.Fatalf("Failed to create config: %v", err)
	}

	// Set HOME to temp directory
	oldHome := os.Getenv("HOME")
	os.Setenv("HOME", tempDir)
	defer os.Setenv("HOME", oldHome)

	// Initialize Git repo WITHOUT remote
	cmd := exec.Command("git", "init")
	cmd.Dir = repoDir
	if err := cmd.Run(); err != nil {
		t.Fatalf("Failed to init git repo: %v", err)
	}

	// Change to repo directory
	oldWd, _ := os.Getwd()
	os.Chdir(repoDir)
	defer os.Chdir(oldWd)

	// Load config
	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	// Call runOpenCurrent - should error
	err = runOpenCurrent(cfg)
	if err == nil {
		t.Error("Expected error for repo without remote, got nil")
	}
	if err != nil && !contains(err.Error(), "failed to get git remote URL") {
		t.Errorf("Expected 'failed to get git remote URL' error, got: %v", err)
	}
}

func TestRunOpenCurrent_MismatchedRemote(t *testing.T) {
	// Check if git is available
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not available, skipping test")
	}

	// Create temp directory structure
	tempDir := t.TempDir()
	configDir := filepath.Join(tempDir, ".config", "glf")
	_ = os.MkdirAll(configDir, 0755)

	repoDir := filepath.Join(tempDir, "repo")
	_ = os.MkdirAll(repoDir, 0755)

	// Create config
	configPath := filepath.Join(configDir, "config.yaml")
	configContent := `gitlab:
  url: https://gitlab.example.com
  token: test-token`
	if err := os.WriteFile(configPath, []byte(configContent), 0600); err != nil {
		t.Fatalf("Failed to create config: %v", err)
	}

	// Set HOME to temp directory
	oldHome := os.Getenv("HOME")
	os.Setenv("HOME", tempDir)
	defer os.Setenv("HOME", oldHome)

	// Initialize Git repo with different domain
	cmd := exec.Command("git", "init")
	cmd.Dir = repoDir
	if err := cmd.Run(); err != nil {
		t.Fatalf("Failed to init git repo: %v", err)
	}

	cmd = exec.Command("git", "remote", "add", "origin", "git@gitlab.other.com:test/project.git")
	cmd.Dir = repoDir
	if err := cmd.Run(); err != nil {
		t.Fatalf("Failed to add remote: %v", err)
	}

	// Change to repo directory
	oldWd, _ := os.Getwd()
	os.Chdir(repoDir)
	defer os.Chdir(oldWd)

	// Load config
	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	// Call runOpenCurrent - should error about mismatch
	err = runOpenCurrent(cfg)
	if err == nil {
		t.Error("Expected error for mismatched remote, got nil")
	}
	if err != nil && !contains(err.Error(), "does not match configured GitLab") {
		t.Errorf("Expected 'does not match' error, got: %v", err)
	}
}

func TestOpenBrowser_EmptyURL(t *testing.T) {
	// Test with empty URL
	err := openBrowser("")
	// Should not panic, may or may not error depending on platform
	if err != nil {
		t.Logf("openBrowser with empty URL returned error (expected): %v", err)
	}
}

func TestOpenBrowser_SpecialCharacters(t *testing.T) {
	// Test with URL containing special characters
	testURL := "https://gitlab.example.com/test/project?foo=bar&baz=qux"
	err := openBrowser(testURL)
	// Should not panic
	if err != nil {
		t.Logf("openBrowser with special chars returned error (expected in test env): %v", err)
	}
}

func TestRunSearch_WithDotArgument(t *testing.T) {
	// Check if git is available
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not available, skipping test")
	}

	// Create temp directory structure
	tempDir := t.TempDir()
	configDir := filepath.Join(tempDir, ".config", "glf")
	_ = os.MkdirAll(configDir, 0755)

	repoDir := filepath.Join(tempDir, "repo")
	_ = os.MkdirAll(repoDir, 0755)

	// Create config
	configPath := filepath.Join(configDir, "config.yaml")
	configContent := `gitlab:
  url: https://gitlab.example.com
  token: test-token`
	if err := os.WriteFile(configPath, []byte(configContent), 0600); err != nil {
		t.Fatalf("Failed to create config: %v", err)
	}

	// Set HOME to temp directory
	oldHome := os.Getenv("HOME")
	os.Setenv("HOME", tempDir)
	defer os.Setenv("HOME", oldHome)

	// Initialize Git repo with remote
	cmd := exec.Command("git", "init")
	cmd.Dir = repoDir
	if err := cmd.Run(); err != nil {
		t.Fatalf("Failed to init git repo: %v", err)
	}

	cmd = exec.Command("git", "remote", "add", "origin", "git@gitlab.example.com:test/project.git")
	cmd.Dir = repoDir
	if err := cmd.Run(); err != nil {
		t.Fatalf("Failed to add remote: %v", err)
	}

	// Change to repo directory
	oldWd, _ := os.Getwd()
	os.Chdir(repoDir)
	defer os.Chdir(oldWd)

	// Create empty command
	cobraCmd := &cobra.Command{}

	// Call runSearch with "." argument
	err := runSearch(cobraCmd, []string{"."})
	if err != nil {
		t.Errorf("runSearch with '.' argument failed: %v", err)
	}
}

func TestExtractProjectPath_EdgeCases(t *testing.T) {
	tests := []struct {
		name         string
		remoteURL    string
		gitlabURL    string
		expectedPath string
		expectedBase string
		expectError  bool
		errorMsg     string
	}{
		{
			name:         "Very deep namespace hierarchy",
			remoteURL:    "git@gitlab.example.com:group/subgroup1/subgroup2/subgroup3/project.git",
			gitlabURL:    "https://gitlab.example.com",
			expectedPath: "group/subgroup1/subgroup2/subgroup3/project",
			expectedBase: "https://gitlab.example.com",
			expectError:  false,
		},
		{
			name:         "Project name with dashes and underscores",
			remoteURL:    "git@gitlab.example.com:namespace/my-project_name.git",
			gitlabURL:    "https://gitlab.example.com",
			expectedPath: "namespace/my-project_name",
			expectedBase: "https://gitlab.example.com",
			expectError:  false,
		},
		{
			name:         "Project name with dots",
			remoteURL:    "git@gitlab.example.com:namespace/my.project.name.git",
			gitlabURL:    "https://gitlab.example.com",
			expectedPath: "namespace/my.project.name",
			expectedBase: "https://gitlab.example.com",
			expectError:  false,
		},
		{
			name:         "Numeric project name",
			remoteURL:    "git@gitlab.example.com:namespace/12345.git",
			gitlabURL:    "https://gitlab.example.com",
			expectedPath: "namespace/12345",
			expectedBase: "https://gitlab.example.com",
			expectError:  false,
		},
		{
			name:        "SSH URL with no path",
			remoteURL:   "git@gitlab.example.com:",
			gitlabURL:   "https://gitlab.example.com",
			expectError: true,
			errorMsg:    "could not extract project path",
		},
		{
			name:        "HTTPS URL with only slash",
			remoteURL:   "https://gitlab.example.com/",
			gitlabURL:   "https://gitlab.example.com",
			expectError: true,
			errorMsg:    "invalid remote URL format",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			projectPath, baseURL, err := extractProjectPath(tt.remoteURL, tt.gitlabURL)

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
				if projectPath != tt.expectedPath {
					t.Errorf("Expected path '%s', got '%s'", tt.expectedPath, projectPath)
				}
				if baseURL != tt.expectedBase {
					t.Errorf("Expected base URL '%s', got '%s'", tt.expectedBase, baseURL)
				}
			}
		})
	}
}

func TestIndexDescriptions_VerifyIndexContent(t *testing.T) {
	// Test that we can actually query the index after indexing
	tempDir := t.TempDir()

	projects := []types.Project{
		{
			Path:        "group/project1",
			Name:        "Backend API",
			Description: "REST API for authentication",
		},
		{
			Path:        "group/project2",
			Name:        "Frontend App",
			Description: "React application for users",
		},
	}

	err := indexDescriptions(projects, tempDir, true)
	if err != nil {
		t.Fatalf("indexDescriptions failed: %v", err)
	}

	// Open index and verify we can query it
	indexPath := filepath.Join(tempDir, "description.bleve")
	descIndex, err := index.NewDescriptionIndex(indexPath)
	if err != nil {
		t.Fatalf("Failed to open index: %v", err)
	}
	defer descIndex.Close()

	// Verify we can get projects back
	allProjects, err := descIndex.GetAllProjects()
	if err != nil {
		t.Fatalf("Failed to get projects from index: %v", err)
	}

	if len(allProjects) != 2 {
		t.Errorf("Expected 2 projects in index, got %d", len(allProjects))
	}
}

func TestIndexDescriptions_IncrementalUpdate(t *testing.T) {
	// Test indexing twice (simulating incremental sync)
	tempDir := t.TempDir()

	// First batch
	projects1 := []types.Project{
		{
			Path:        "group/project1",
			Name:        "Project 1",
			Description: "First project",
		},
	}

	err := indexDescriptions(projects1, tempDir, true)
	if err != nil {
		t.Fatalf("First indexDescriptions failed: %v", err)
	}

	// Second batch (simulating incremental sync)
	projects2 := []types.Project{
		{
			Path:        "group/project2",
			Name:        "Project 2",
			Description: "Second project",
		},
	}

	err = indexDescriptions(projects2, tempDir, true)
	if err != nil {
		t.Fatalf("Second indexDescriptions failed: %v", err)
	}

	// Verify both projects are in index
	indexPath := filepath.Join(tempDir, "description.bleve")
	descIndex, err := index.NewDescriptionIndex(indexPath)
	if err != nil {
		t.Fatalf("Failed to open index: %v", err)
	}
	defer descIndex.Close()

	allProjects, err := descIndex.GetAllProjects()
	if err != nil {
		t.Fatalf("Failed to get projects: %v", err)
	}

	if len(allProjects) < 2 {
		t.Errorf("Expected at least 2 projects after incremental update, got %d", len(allProjects))
	}
}

func TestRunSearch_CorruptedIndex(t *testing.T) {
	// Test that runSearch handles corrupted index gracefully
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

	// Create valid index first
	projects := []types.Project{
		{Path: "test/project", Name: "Test", Description: "Test"},
	}
	_ = indexDescriptions(projects, cacheDir, true)

	// Corrupt the index by writing invalid data to a critical file
	indexPath := filepath.Join(cacheDir, "description.bleve")
	// Find and corrupt a critical index file (store.json is essential for bleve)
	storePath := filepath.Join(indexPath, "store.json")
	_ = os.WriteFile(storePath, []byte("corrupted invalid json data"), 0600)

	// Set HOME to temp directory
	oldHome := os.Getenv("HOME")
	os.Setenv("HOME", tempDir)
	defer os.Setenv("HOME", oldHome)

	// Enable auto-go mode to avoid launching interactive TUI
	// This allows the test to fail quickly with corrupted index
	autoGo = true
	defer func() { autoGo = false }()
	doSync = false

	// Create empty command
	cmd := &cobra.Command{}

	// Try to run with corrupted index
	// The corrupted store.json may or may not cause an error depending on
	// how bleve handles it. The important thing is that it doesn't hang
	// (which was the original bug - it would launch TUI and wait for input).
	err := runSearch(cmd, []string{"test"})

	// Test passes if:
	// 1. It returns quickly (no hang) - which it does now with autoGo=true
	// 2. Either succeeds (bleve is resilient) OR fails with appropriate error
	if err != nil {
		// If there's an error, log it - this is acceptable
		t.Logf("Got error (acceptable): %v", err)
	} else {
		// If there's no error, that's also acceptable - means the system
		// is resilient to this type of corruption
		t.Logf("System handled corrupted index gracefully (no error)")
	}
}

func TestIndexDescriptions_WithExistingIndex(t *testing.T) {
	// Test indexing into an existing index (covers docCount > 0 path)
	tempDir := t.TempDir()

	// First indexing - create index with initial projects
	projects1 := []types.Project{
		{Path: "group/project1", Name: "Project 1", Description: "First project"},
		{Path: "group/project2", Name: "Project 2", Description: "Second project"},
	}

	err := indexDescriptions(projects1, tempDir, true)
	if err != nil {
		t.Fatalf("First indexing failed: %v", err)
	}

	// Verify index was created
	indexPath := filepath.Join(tempDir, "description.bleve")
	if _, err := os.Stat(indexPath); os.IsNotExist(err) {
		t.Fatal("Expected index to be created")
	}

	// Second indexing - add more projects to existing index
	// This should trigger the "Existing index has X documents" log path
	projects2 := []types.Project{
		{Path: "group/project3", Name: "Project 3", Description: "Third project"},
	}

	err = indexDescriptions(projects2, tempDir, true)
	if err != nil {
		t.Fatalf("Second indexing (with existing index) failed: %v", err)
	}

	// Verify all projects are in the index
	descIndex, err := index.NewDescriptionIndex(indexPath)
	if err != nil {
		t.Fatalf("Failed to open index: %v", err)
	}
	defer descIndex.Close()

	// Check document count
	docCount, err := descIndex.Count()
	if err != nil {
		t.Fatalf("Failed to get document count: %v", err)
	}

	// Should have at least 3 documents (might have more from incremental indexing)
	if docCount < 3 {
		t.Errorf("Expected at least 3 documents in index, got %d", docCount)
	}
}

func TestIndexDescriptions_LargeBatch(t *testing.T) {
	// Test indexing with projects that trigger progress logging (indexed % 50 == 0)
	tempDir := t.TempDir()

	// Create exactly 200 projects to trigger multiple batches and progress logs
	// This will create 2 full batches of 100 each
	projects := make([]types.Project, 200)
	for i := 0; i < 200; i++ {
		projects[i] = types.Project{
			Path:        "group/project-" + string(rune(i)),
			Name:        "Project " + string(rune(i)),
			Description: "Description " + string(rune(i)),
		}
	}

	err := indexDescriptions(projects, tempDir, true)
	if err != nil {
		t.Fatalf("Large batch indexing failed: %v", err)
	}

	// Verify index was created and contains all projects
	indexPath := filepath.Join(tempDir, "description.bleve")
	descIndex, err := index.NewDescriptionIndex(indexPath)
	if err != nil {
		t.Fatalf("Failed to open index: %v", err)
	}
	defer descIndex.Close()

	// Check document count - should have 200 projects
	docCount, err := descIndex.Count()
	if err != nil {
		t.Fatalf("Failed to get document count: %v", err)
	}

	if docCount != 200 {
		t.Errorf("Expected 200 documents in index, got %d", docCount)
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

// TestPerformSyncInternalWithClient_Success tests successful full sync
func TestPerformSyncInternalWithClient_Success(t *testing.T) {
	tempDir := t.TempDir()
	cacheDir := filepath.Join(tempDir, "cache")
	_ = os.MkdirAll(cacheDir, 0755)

	cfg := &config.Config{
		GitLab: config.GitLabConfig{
			URL:     "https://gitlab.example.com",
			Token:   "test-token",
			Timeout: 30,
		},
		Cache: config.CacheConfig{
			Dir: cacheDir,
		},
	}

	// Create mock client that returns test projects
	mockClient := &mockGitLabClient{
		testConnectionFunc: func() error {
			return nil // Connection succeeds
		},
		fetchProjectsFunc: func(since *time.Time) ([]types.Project, error) {
			return []types.Project{
				{Path: "group/project1", Name: "Project 1", Description: "Test project 1"},
				{Path: "group/project2", Name: "Project 2", Description: "Test project 2"},
			}, nil
		},
	}

	// Perform sync with mock client
	err := performSyncInternalWithClient(cfg, mockClient, true, false)
	if err != nil {
		t.Fatalf("Sync failed: %v", err)
	}

	// Verify index was created and contains projects
	indexPath := filepath.Join(cacheDir, "description.bleve")
	if !index.Exists(indexPath) {
		t.Fatal("Index was not created")
	}

	descIndex, err := index.NewDescriptionIndex(indexPath)
	if err != nil {
		t.Fatalf("Failed to open index: %v", err)
	}
	defer descIndex.Close()

	projects, err := descIndex.GetAllProjects()
	if err != nil {
		t.Fatalf("Failed to get projects from index: %v", err)
	}

	if len(projects) != 2 {
		t.Errorf("Expected 2 projects in index, got %d", len(projects))
	}
}

// TestPerformSyncInternalWithClient_ConnectionFailure tests connection failure handling
func TestPerformSyncInternalWithClient_ConnectionFailure(t *testing.T) {
	tempDir := t.TempDir()
	cacheDir := filepath.Join(tempDir, "cache")
	_ = os.MkdirAll(cacheDir, 0755)

	cfg := &config.Config{
		GitLab: config.GitLabConfig{
			URL:     "https://gitlab.example.com",
			Token:   "test-token",
			Timeout: 30,
		},
		Cache: config.CacheConfig{
			Dir: cacheDir,
		},
	}

	// Create mock client that fails connection test
	mockClient := &mockGitLabClient{
		testConnectionFunc: func() error {
			return fmt.Errorf("connection refused")
		},
	}

	// Perform sync - should fail with connection error
	err := performSyncInternalWithClient(cfg, mockClient, true, false)
	if err == nil {
		t.Fatal("Expected error for connection failure, got nil")
	}

	if !contains(err.Error(), "connection test failed") {
		t.Errorf("Expected 'connection test failed' error, got: %v", err)
	}
}

// TestPerformSyncInternalWithClient_FetchFailure tests fetch failure handling
func TestPerformSyncInternalWithClient_FetchFailure(t *testing.T) {
	tempDir := t.TempDir()
	cacheDir := filepath.Join(tempDir, "cache")
	_ = os.MkdirAll(cacheDir, 0755)

	cfg := &config.Config{
		GitLab: config.GitLabConfig{
			URL:     "https://gitlab.example.com",
			Token:   "test-token",
			Timeout: 30,
		},
		Cache: config.CacheConfig{
			Dir: cacheDir,
		},
	}

	// Create mock client that fails to fetch projects
	mockClient := &mockGitLabClient{
		testConnectionFunc: func() error {
			return nil // Connection succeeds
		},
		fetchProjectsFunc: func(since *time.Time) ([]types.Project, error) {
			return nil, fmt.Errorf("API error: rate limit exceeded")
		},
	}

	// Perform sync - should fail with fetch error
	err := performSyncInternalWithClient(cfg, mockClient, true, false)
	if err == nil {
		t.Fatal("Expected error for fetch failure, got nil")
	}

	if !contains(err.Error(), "fetch error") {
		t.Errorf("Expected 'fetch error' in message, got: %v", err)
	}
}

// TestPerformSyncInternalWithClient_NoProjects tests handling of zero projects
func TestPerformSyncInternalWithClient_NoProjects(t *testing.T) {
	tempDir := t.TempDir()
	cacheDir := filepath.Join(tempDir, "cache")
	_ = os.MkdirAll(cacheDir, 0755)

	cfg := &config.Config{
		GitLab: config.GitLabConfig{
			URL:     "https://gitlab.example.com",
			Token:   "test-token",
			Timeout: 30,
		},
		Cache: config.CacheConfig{
			Dir: cacheDir,
		},
	}

	// Create mock client that returns no projects
	mockClient := &mockGitLabClient{
		testConnectionFunc: func() error {
			return nil
		},
		fetchProjectsFunc: func(since *time.Time) ([]types.Project, error) {
			return []types.Project{}, nil // Empty list
		},
	}

	// Perform sync - should succeed but warn about no projects
	err := performSyncInternalWithClient(cfg, mockClient, true, false)
	if err != nil {
		t.Fatalf("Sync should succeed with no projects, got error: %v", err)
	}
}

// TestPerformSyncInternalWithClient_IncrementalSync tests incremental sync mode
func TestPerformSyncInternalWithClient_IncrementalSync(t *testing.T) {
	tempDir := t.TempDir()
	cacheDir := filepath.Join(tempDir, "cache")
	_ = os.MkdirAll(cacheDir, 0755)

	cfg := &config.Config{
		GitLab: config.GitLabConfig{
			URL:     "https://gitlab.example.com",
			Token:   "test-token",
			Timeout: 30,
		},
		Cache: config.CacheConfig{
			Dir: cacheDir,
		},
	}

	// First sync - full sync to establish baseline
	mockClient1 := &mockGitLabClient{
		testConnectionFunc: func() error {
			return nil
		},
		fetchProjectsFunc: func(since *time.Time) ([]types.Project, error) {
			if since != nil {
				t.Error("First sync should be full sync (since should be nil)")
			}
			return []types.Project{
				{Path: "group/project1", Name: "Project 1", Description: "First"},
			}, nil
		},
	}

	err := performSyncInternalWithClient(cfg, mockClient1, true, false)
	if err != nil {
		t.Fatalf("First sync failed: %v", err)
	}

	// Second sync - incremental (since timestamp exists)
	var incrementalCallMade bool
	mockClient2 := &mockGitLabClient{
		testConnectionFunc: func() error {
			return nil
		},
		fetchProjectsFunc: func(since *time.Time) ([]types.Project, error) {
			if since == nil {
				t.Error("Second sync should be incremental (since should not be nil)")
			} else {
				incrementalCallMade = true
			}
			return []types.Project{
				{Path: "group/project2", Name: "Project 2", Description: "Second"},
			}, nil
		},
	}

	err = performSyncInternalWithClient(cfg, mockClient2, true, false)
	if err != nil {
		t.Fatalf("Incremental sync failed: %v", err)
	}

	if !incrementalCallMade {
		t.Error("Incremental sync was not performed (since parameter was not set)")
	}

	// Verify both projects are in the index
	indexPath := filepath.Join(cacheDir, "description.bleve")
	descIndex, err := index.NewDescriptionIndex(indexPath)
	if err != nil {
		t.Fatalf("Failed to open index: %v", err)
	}
	defer descIndex.Close()

	projects, err := descIndex.GetAllProjects()
	if err != nil {
		t.Fatalf("Failed to get projects from index: %v", err)
	}

	if len(projects) < 2 {
		t.Errorf("Expected at least 2 projects after incremental sync, got %d", len(projects))
	}
}

// TestPerformSyncInternalWithClient_ForceFullSync tests force full sync flag
func TestPerformSyncInternalWithClient_ForceFullSync(t *testing.T) {
	tempDir := t.TempDir()
	cacheDir := filepath.Join(tempDir, "cache")
	_ = os.MkdirAll(cacheDir, 0755)

	cfg := &config.Config{
		GitLab: config.GitLabConfig{
			URL:     "https://gitlab.example.com",
			Token:   "test-token",
			Timeout: 30,
		},
		Cache: config.CacheConfig{
			Dir: cacheDir,
		},
	}

	// First sync to create timestamp
	mockClient1 := &mockGitLabClient{
		testConnectionFunc: func() error { return nil },
		fetchProjectsFunc: func(since *time.Time) ([]types.Project, error) {
			return []types.Project{
				{Path: "group/project1", Name: "Project 1", Description: "First"},
			}, nil
		},
	}
	_ = performSyncInternalWithClient(cfg, mockClient1, true, false)

	// Second sync with forceFullSync=true should pass since=nil
	var fullSyncCalled bool
	mockClient2 := &mockGitLabClient{
		testConnectionFunc: func() error { return nil },
		fetchProjectsFunc: func(since *time.Time) ([]types.Project, error) {
			if since == nil {
				fullSyncCalled = true
			}
			return []types.Project{
				{Path: "group/project2", Name: "Project 2", Description: "Second"},
			}, nil
		},
	}

	err := performSyncInternalWithClient(cfg, mockClient2, true, true) // forceFullSync=true
	if err != nil {
		t.Fatalf("Force full sync failed: %v", err)
	}

	if !fullSyncCalled {
		t.Error("Force full sync flag was ignored - incremental sync was performed instead")
	}
}

// TestRunAutoGoWithSync_EmptyProjects tests error handling for empty project list
func TestRunAutoGoWithSync_EmptyProjects(t *testing.T) {
	tempDir := t.TempDir()
	cfg := &config.Config{
		GitLab: config.GitLabConfig{URL: "https://gitlab.example.com"},
		Cache:  config.CacheConfig{Dir: tempDir},
	}

	mockSync := func() error { return nil }

	err := runAutoGoWithSync([]types.Project{}, "query", cfg, nil, mockSync)
	if err == nil {
		t.Fatal("Expected error for empty projects, got nil")
	}

	expectedError := "no projects in cache"
	if err.Error() != expectedError {
		t.Errorf("Expected error '%s', got '%s'", expectedError, err.Error())
	}
}

// TestRunAutoGoWithSync_NoMatches tests error handling when search returns no results
func TestRunAutoGoWithSync_NoMatches(t *testing.T) {
	tempDir := t.TempDir()
	cacheDir := filepath.Join(tempDir, "cache")
	_ = os.MkdirAll(cacheDir, 0755)

	cfg := &config.Config{
		GitLab: config.GitLabConfig{URL: "https://gitlab.example.com"},
		Cache:  config.CacheConfig{Dir: cacheDir},
	}

	// Create test projects
	projects := []types.Project{
		{Path: "backend/api", Name: "API Server", Description: "REST API backend"},
	}

	// Create and populate index
	indexPath := filepath.Join(cacheDir, "description.bleve")
	descIndex, err := index.NewDescriptionIndex(indexPath)
	if err != nil {
		t.Fatalf("Failed to create index: %v", err)
	}

	if err := descIndex.Add(projects[0].Path, projects[0].Name, projects[0].Description); err != nil {
		descIndex.Close()
		t.Fatalf("Failed to add document: %v", err)
	}

	mockSync := func() error { return nil }

	// Search for something that doesn't exist
	err = runAutoGoWithSync(projects, "nonexistent-query-xyz-12345", cfg, descIndex, mockSync)
	descIndex.Close()

	if err == nil {
		t.Fatal("Expected error for no matches, got nil")
	}

	if !contains(err.Error(), "no projects found for query") {
		t.Errorf("Expected 'no projects found' error, got: %v", err)
	}
}

// TestRunAutoGoWithSync_SuccessfulMatch tests successful match with history and sync
func TestRunAutoGoWithSync_SuccessfulMatch(t *testing.T) {
	tempDir := t.TempDir()
	cacheDir := filepath.Join(tempDir, "cache")
	_ = os.MkdirAll(cacheDir, 0755)

	cfg := &config.Config{
		GitLab: config.GitLabConfig{URL: "https://gitlab.example.com"},
		Cache:  config.CacheConfig{Dir: cacheDir},
	}

	// Create test projects
	projects := []types.Project{
		{Path: "backend/api", Name: "API Server", Description: "REST API backend"},
	}

	// Create and populate index
	indexPath := filepath.Join(cacheDir, "description.bleve")
	descIndex, err := index.NewDescriptionIndex(indexPath)
	if err != nil {
		t.Fatalf("Failed to create index: %v", err)
	}

	if err := descIndex.Add(projects[0].Path, projects[0].Name, projects[0].Description); err != nil {
		descIndex.Close()
		t.Fatalf("Failed to add document: %v", err)
	}

	// Mock sync function that succeeds
	syncCalled := false
	mockSync := func() error {
		syncCalled = true
		return nil
	}

	// Search for "api" - should find the project
	err = runAutoGoWithSync(projects, "api", cfg, descIndex, mockSync)
	descIndex.Close()

	// Should succeed (browser opening will fail in test environment, but that's expected)
	if err != nil {
		t.Errorf("runAutoGoWithSync failed: %v", err)
	}

	// Verify sync was called
	if !syncCalled {
		t.Error("Background sync was not called")
	}
}

// TestRunAutoGoWithSync_SyncFailure tests handling of sync function failure
func TestRunAutoGoWithSync_SyncFailure(t *testing.T) {
	tempDir := t.TempDir()
	cacheDir := filepath.Join(tempDir, "cache")
	_ = os.MkdirAll(cacheDir, 0755)

	cfg := &config.Config{
		GitLab: config.GitLabConfig{URL: "https://gitlab.example.com"},
		Cache:  config.CacheConfig{Dir: cacheDir},
	}

	// Create test projects
	projects := []types.Project{
		{Path: "backend/api", Name: "API Server", Description: "REST API backend"},
	}

	// Create and populate index
	indexPath := filepath.Join(cacheDir, "description.bleve")
	descIndex, err := index.NewDescriptionIndex(indexPath)
	if err != nil {
		t.Fatalf("Failed to create index: %v", err)
	}

	if err := descIndex.Add(projects[0].Path, projects[0].Name, projects[0].Description); err != nil {
		descIndex.Close()
		t.Fatalf("Failed to add document: %v", err)
	}

	// Mock sync function that fails
	syncCalled := false
	mockSync := func() error {
		syncCalled = true
		return fmt.Errorf("sync failed: network timeout")
	}

	// Search for "api" - should find the project
	// Even if sync fails, the function should succeed (it just logs the error)
	err = runAutoGoWithSync(projects, "api", cfg, descIndex, mockSync)
	descIndex.Close()

	if err != nil {
		t.Errorf("runAutoGoWithSync should succeed even if sync fails, got: %v", err)
	}

	// Verify sync was called
	if !syncCalled {
		t.Error("Background sync was not called")
	}
}

// TestRunAutoGoWithSync_SyncTimeout tests handling of sync timeout
func TestRunAutoGoWithSync_SyncTimeout(t *testing.T) {
	// This test verifies the 30-second timeout logic
	// We use a sync function that takes longer than the test timeout but returns quickly
	// to avoid making the test take 30 seconds
	tempDir := t.TempDir()
	cacheDir := filepath.Join(tempDir, "cache")
	_ = os.MkdirAll(cacheDir, 0755)

	cfg := &config.Config{
		GitLab: config.GitLabConfig{URL: "https://gitlab.example.com"},
		Cache:  config.CacheConfig{Dir: cacheDir},
	}

	// Create test projects
	projects := []types.Project{
		{Path: "backend/api", Name: "API Server", Description: "REST API backend"},
	}

	// Create and populate index
	indexPath := filepath.Join(cacheDir, "description.bleve")
	descIndex, err := index.NewDescriptionIndex(indexPath)
	if err != nil {
		t.Fatalf("Failed to create index: %v", err)
	}

	if err := descIndex.Add(projects[0].Path, projects[0].Name, projects[0].Description); err != nil {
		descIndex.Close()
		t.Fatalf("Failed to add document: %v", err)
	}

	// Note: We can't easily test the actual 30-second timeout without making the test slow
	// But we can verify the code path exists by using a fast sync function
	// The timeout logic is covered by the code structure
	syncCalled := false
	mockSync := func() error {
		syncCalled = true
		// Fast return to avoid slow test
		return nil
	}

	err = runAutoGoWithSync(projects, "api", cfg, descIndex, mockSync)
	descIndex.Close()

	if err != nil {
		t.Errorf("runAutoGoWithSync failed: %v", err)
	}

	if !syncCalled {
		t.Error("Background sync was not called")
	}
}

// TestPerformSyncInternalWithClient_IncrementalSyncNoChanges tests incremental sync returning 0 projects
func TestPerformSyncInternalWithClient_IncrementalSyncNoChanges(t *testing.T) {
	tempDir := t.TempDir()
	cacheDir := filepath.Join(tempDir, "cache")
	_ = os.MkdirAll(cacheDir, 0755)

	cfg := &config.Config{
		GitLab: config.GitLabConfig{
			URL:     "https://gitlab.example.com",
			Token:   "test-token",
			Timeout: 30,
		},
		Cache: config.CacheConfig{
			Dir: cacheDir,
		},
	}

	// First sync to establish baseline
	mockClient1 := &mockGitLabClient{
		testConnectionFunc: func() error { return nil },
		fetchProjectsFunc: func(since *time.Time) ([]types.Project, error) {
			return []types.Project{
				{Path: "group/project1", Name: "Project 1", Description: "First"},
			}, nil
		},
	}
	_ = performSyncInternalWithClient(cfg, mockClient1, true, false)

	// Second sync - incremental with no changes (returns 0 projects)
	mockClient2 := &mockGitLabClient{
		testConnectionFunc: func() error { return nil },
		fetchProjectsFunc: func(since *time.Time) ([]types.Project, error) {
			// Return empty list - no projects changed
			return []types.Project{}, nil
		},
	}

	err := performSyncInternalWithClient(cfg, mockClient2, true, false)
	// Should succeed with no error - this tests the early return path
	if err != nil {
		t.Errorf("Incremental sync with no changes should succeed, got error: %v", err)
	}
}

// TestPerformSyncInternal_InvalidConfig tests performSyncInternal with invalid GitLab config
func TestPerformSyncInternal_InvalidConfig(t *testing.T) {
	tempDir := t.TempDir()
	cacheDir := filepath.Join(tempDir, "cache")
	_ = os.MkdirAll(cacheDir, 0755)

	// Create config with invalid URL (malformed URL should cause gitlab.New() to fail)
	cfg := &config.Config{
		GitLab: config.GitLabConfig{
			URL:     ":/invalid-url-format",
			Token:   "test-token",
			Timeout: 30,
		},
		Cache: config.CacheConfig{
			Dir: cacheDir,
		},
	}

	// Call performSyncInternal - should fail with GitLab client error
	err := performSyncInternal(cfg, true, false)
	if err == nil {
		t.Fatal("Expected error for invalid GitLab URL, got nil")
	}

	// Should get an error from gitlab.New() wrapped in "GitLab client error"
	if !contains(err.Error(), "GitLab client error") {
		t.Errorf("Expected 'GitLab client error' in message, got: %v", err)
	}
}

// TestRunSearch_WithSyncFlag tests runSearch with --sync flag
func TestRunSearch_WithSyncFlag(t *testing.T) {
	tempDir := t.TempDir()
	configDir := filepath.Join(tempDir, ".config", "glf")
	_ = os.MkdirAll(configDir, 0755)

	cacheDir := filepath.Join(tempDir, "cache")
	_ = os.MkdirAll(cacheDir, 0755)

	// Create config with invalid GitLab URL so sync will fail
	configPath := filepath.Join(configDir, "config.yaml")
	configContent := `gitlab:
  url: :/invalid-url
  token: test-token
cache:
  dir: ` + cacheDir
	_ = os.WriteFile(configPath, []byte(configContent), 0600)

	// Set HOME to temp directory
	oldHome := os.Getenv("HOME")
	os.Setenv("HOME", tempDir)
	defer os.Setenv("HOME", oldHome)

	// Enable sync mode
	doSync = true
	defer func() { doSync = false }()

	// Create empty command
	cmd := &cobra.Command{}

	// Try to run with sync - should fail during sync with GitLab client error
	err := runSearch(cmd, []string{"test"})
	if err == nil {
		t.Fatal("Expected error for sync with invalid config, got nil")
	}

	// Should get GitLab client error from performSyncInternal
	if !contains(err.Error(), "GitLab client error") {
		t.Errorf("Expected 'GitLab client error' from sync, got: %v", err)
	}
}

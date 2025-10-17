package cache

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/igusev/glf/internal/model"
)

func TestSaveLoadLastSyncTime(t *testing.T) {
	// Create temp directory for testing
	tmpDir, err := os.MkdirTemp("", "glf-cache-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	cache := New(tmpDir)

	// Test save and load
	testTime := time.Now().UTC().Truncate(time.Second)
	if err := cache.SaveLastSyncTime(testTime); err != nil {
		t.Fatalf("SaveLastSyncTime failed: %v", err)
	}

	loaded, err := cache.LoadLastSyncTime()
	if err != nil {
		t.Fatalf("LoadLastSyncTime failed: %v", err)
	}

	if !loaded.Equal(testTime) {
		t.Errorf("Loaded time mismatch: got %v, want %v", loaded, testTime)
	}
}

func TestLoadLastSyncTime_FirstSync(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "glf-cache-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	cache := New(tmpDir)

	// Loading without any saved time should return zero time
	loaded, err := cache.LoadLastSyncTime()
	if err != nil {
		t.Fatalf("LoadLastSyncTime should not error on first sync: %v", err)
	}

	if !loaded.IsZero() {
		t.Errorf("First sync should return zero time, got: %v", loaded)
	}
}

func TestSaveLoadLastFullSyncTime(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "glf-cache-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	cache := New(tmpDir)

	testTime := time.Now().UTC().Truncate(time.Second)
	if err := cache.SaveLastFullSyncTime(testTime); err != nil {
		t.Fatalf("SaveLastFullSyncTime failed: %v", err)
	}

	loaded, err := cache.LoadLastFullSyncTime()
	if err != nil {
		t.Fatalf("LoadLastFullSyncTime failed: %v", err)
	}

	if !loaded.Equal(testTime) {
		t.Errorf("Loaded full sync time mismatch: got %v, want %v", loaded, testTime)
	}
}

func TestLoadLastFullSyncTime_NeverRan(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "glf-cache-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	cache := New(tmpDir)

	loaded, err := cache.LoadLastFullSyncTime()
	if err != nil {
		t.Fatalf("LoadLastFullSyncTime should not error when never ran: %v", err)
	}

	if !loaded.IsZero() {
		t.Errorf("Never ran full sync should return zero time, got: %v", loaded)
	}
}

func TestReadWriteProjects(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "glf-cache-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	cache := New(tmpDir)

	// Create test projects
	projects := []model.Project{
		{
			Path:        "group/project1",
			Name:        "Project 1",
			Description: "Test project 1",
		},
		{
			Path:        "group/project2",
			Name:        "Project 2",
			Description: "Test project 2 with pipe | character",
		},
		{
			Path:        "group/project3",
			Name:        "Project 3",
			Description: "", // Empty description
		},
	}

	// Write projects
	if err := cache.WriteProjects(projects); err != nil {
		t.Fatalf("WriteProjects failed: %v", err)
	}

	// Read projects
	loaded, err := cache.ReadProjects()
	if err != nil {
		t.Fatalf("ReadProjects failed: %v", err)
	}

	// Verify
	if len(loaded) != len(projects) {
		t.Fatalf("Project count mismatch: got %d, want %d", len(loaded), len(projects))
	}

	for i, proj := range loaded {
		if proj.Path != projects[i].Path {
			t.Errorf("Project %d path mismatch: got %q, want %q", i, proj.Path, projects[i].Path)
		}
		if proj.Name != projects[i].Name {
			t.Errorf("Project %d name mismatch: got %q, want %q", i, proj.Name, projects[i].Name)
		}
		if proj.Description != projects[i].Description {
			t.Errorf("Project %d description mismatch: got %q, want %q", i, proj.Description, projects[i].Description)
		}
	}
}

func TestReadProjects_NotFound(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "glf-cache-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	cache := New(tmpDir)

	_, err = cache.ReadProjects()
	if err == nil {
		t.Fatal("ReadProjects should error when file doesn't exist")
	}
}

func TestProjectsPath(t *testing.T) {
	tmpDir := "/tmp/test-cache"
	cache := New(tmpDir)

	expected := filepath.Join(tmpDir, "projects.txt")
	if cache.ProjectsPath() != expected {
		t.Errorf("ProjectsPath mismatch: got %q, want %q", cache.ProjectsPath(), expected)
	}
}

func TestEnsureDir(t *testing.T) {
	tmpDir := filepath.Join(os.TempDir(), "glf-cache-test-ensure-"+time.Now().Format("20060102150405"))
	defer os.RemoveAll(tmpDir)

	cache := New(tmpDir)

	if err := cache.EnsureDir(); err != nil {
		t.Fatalf("EnsureDir failed: %v", err)
	}

	// Verify directory exists
	if _, err := os.Stat(tmpDir); os.IsNotExist(err) {
		t.Error("Directory was not created")
	}

	// Second call should also work (idempotent)
	if err := cache.EnsureDir(); err != nil {
		t.Fatalf("Second EnsureDir failed: %v", err)
	}
}

func TestExists(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "glf-cache-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	cache := New(tmpDir)

	// Initially should not exist
	if cache.Exists() {
		t.Error("Cache should not exist initially")
	}

	// After writing, should exist
	projects := []model.Project{{Path: "test", Name: "Test"}}
	if err := cache.WriteProjects(projects); err != nil {
		t.Fatalf("WriteProjects failed: %v", err)
	}

	if !cache.Exists() {
		t.Error("Cache should exist after writing")
	}
}

func TestStats(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "glf-cache-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	cache := New(tmpDir)

	projects := []model.Project{
		{Path: "p1", Name: "Project 1"},
		{Path: "p2", Name: "Project 2"},
		{Path: "p3", Name: "Project 3"},
	}

	if err := cache.WriteProjects(projects); err != nil {
		t.Fatalf("WriteProjects failed: %v", err)
	}

	count, err := cache.Stats()
	if err != nil {
		t.Fatalf("Stats failed: %v", err)
	}

	if count != 3 {
		t.Errorf("Stats count mismatch: got %d, want 3", count)
	}
}

func TestStats_CacheNotExists(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "glf-cache-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	cache := New(tmpDir)

	// Stats should error when cache doesn't exist
	_, err = cache.Stats()
	if err == nil {
		t.Error("Stats should error when cache file doesn't exist")
	}
}

func TestReadProjects_MalformedLines(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "glf-cache-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	cache := New(tmpDir)

	// Write malformed cache file manually
	cacheFile := cache.ProjectsPath()
	if err := cache.EnsureDir(); err != nil {
		t.Fatalf("EnsureDir failed: %v", err)
	}

	// Create file with various malformed lines
	content := `group/project1|Project 1|Description 1
invalid-line-no-pipe

group/project2|Project 2|Description 2
|missing-path|Description
group/project3|Project 3|Description 3
`
	if err := os.WriteFile(cacheFile, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	// Read should skip malformed lines
	projects, err := cache.ReadProjects()
	if err != nil {
		t.Fatalf("ReadProjects failed: %v", err)
	}

	// Should have 4 projects: 3 valid ones + one with empty path ("|missing-path|Description")
	// The parser accepts empty path as valid (skipping only lines with < 2 fields or empty lines)
	if len(projects) != 4 {
		t.Errorf("Expected 4 projects, got %d", len(projects))
	}

	// Verify that "invalid-line-no-pipe" was skipped (has < 2 fields)
	for _, proj := range projects {
		if proj.Path == "invalid-line-no-pipe" || proj.Name == "invalid-line-no-pipe" {
			t.Error("Invalid line without pipe should have been skipped")
		}
	}
}

func TestReadProjects_OldFormat(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "glf-cache-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	cache := New(tmpDir)

	// Write old format (path|name without description)
	cacheFile := cache.ProjectsPath()
	if err := cache.EnsureDir(); err != nil {
		t.Fatalf("EnsureDir failed: %v", err)
	}

	content := `group/project1|Project 1
group/project2|Project 2
`
	if err := os.WriteFile(cacheFile, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	projects, err := cache.ReadProjects()
	if err != nil {
		t.Fatalf("ReadProjects failed: %v", err)
	}

	if len(projects) != 2 {
		t.Fatalf("Expected 2 projects, got %d", len(projects))
	}

	// Verify descriptions are empty (backward compatibility)
	for i, proj := range projects {
		if proj.Description != "" {
			t.Errorf("Project %d should have empty description, got %q", i, proj.Description)
		}
	}
}

func TestReadProjects_DescriptionWithNewlines(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "glf-cache-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	cache := New(tmpDir)

	// Create project with description containing newlines
	projects := []model.Project{
		{
			Path:        "group/project",
			Name:        "Project",
			Description: "Line 1\nLine 2\nLine 3",
		},
	}

	if err := cache.WriteProjects(projects); err != nil {
		t.Fatalf("WriteProjects failed: %v", err)
	}

	loaded, err := cache.ReadProjects()
	if err != nil {
		t.Fatalf("ReadProjects failed: %v", err)
	}

	// Newlines should be replaced with spaces
	expected := "Line 1 Line 2 Line 3"
	if loaded[0].Description != expected {
		t.Errorf("Description mismatch: got %q, want %q", loaded[0].Description, expected)
	}
}

func TestLoadLastSyncTime_CorruptedFile(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "glf-cache-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	cache := New(tmpDir)

	// Write invalid timestamp
	timestampPath := filepath.Join(tmpDir, ".last_sync_time")
	if err := cache.EnsureDir(); err != nil {
		t.Fatalf("EnsureDir failed: %v", err)
	}
	if err := os.WriteFile(timestampPath, []byte("invalid-timestamp"), 0644); err != nil {
		t.Fatalf("Failed to write corrupted file: %v", err)
	}

	_, err = cache.LoadLastSyncTime()
	if err == nil {
		t.Error("LoadLastSyncTime should error on corrupted timestamp")
	}
}

func TestLoadLastFullSyncTime_CorruptedFile(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "glf-cache-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	cache := New(tmpDir)

	// Write invalid timestamp
	timestampPath := filepath.Join(tmpDir, ".last_full_sync_time")
	if err := cache.EnsureDir(); err != nil {
		t.Fatalf("EnsureDir failed: %v", err)
	}
	if err := os.WriteFile(timestampPath, []byte("invalid-timestamp"), 0644); err != nil {
		t.Fatalf("Failed to write corrupted file: %v", err)
	}

	_, err = cache.LoadLastFullSyncTime()
	if err == nil {
		t.Error("LoadLastFullSyncTime should error on corrupted timestamp")
	}
}

func TestWriteProjects_ReadOnlyDir(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Skipping test on Windows: chmod doesn't work the same way")
	}
	if os.Getuid() == 0 {
		t.Skip("Skipping test when running as root")
	}

	tmpDir, err := os.MkdirTemp("", "glf-cache-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Make directory read-only
	if err := os.Chmod(tmpDir, 0444); err != nil {
		t.Fatalf("Failed to chmod: %v", err)
	}
	defer os.Chmod(tmpDir, 0755) // Restore for cleanup

	cache := New(filepath.Join(tmpDir, "subdir"))
	projects := []model.Project{{Path: "test", Name: "Test"}}

	err = cache.WriteProjects(projects)
	if err == nil {
		t.Error("WriteProjects should fail with read-only parent directory")
	}
}

func TestSaveLastSyncTime_ReadOnlyDir(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Skipping test on Windows: chmod doesn't work the same way")
	}
	if os.Getuid() == 0 {
		t.Skip("Skipping test when running as root")
	}

	tmpDir, err := os.MkdirTemp("", "glf-cache-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Make directory read-only
	if err := os.Chmod(tmpDir, 0444); err != nil {
		t.Fatalf("Failed to chmod: %v", err)
	}
	defer os.Chmod(tmpDir, 0755)

	cache := New(filepath.Join(tmpDir, "subdir"))

	err = cache.SaveLastSyncTime(time.Now())
	if err == nil {
		t.Error("SaveLastSyncTime should fail with read-only parent directory")
	}
}

func TestSaveLastFullSyncTime_ReadOnlyDir(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Skipping test on Windows: chmod doesn't work the same way")
	}
	if os.Getuid() == 0 {
		t.Skip("Skipping test when running as root")
	}

	tmpDir, err := os.MkdirTemp("", "glf-cache-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Make directory read-only
	if err := os.Chmod(tmpDir, 0444); err != nil {
		t.Fatalf("Failed to chmod: %v", err)
	}
	defer os.Chmod(tmpDir, 0755)

	cache := New(filepath.Join(tmpDir, "subdir"))

	err = cache.SaveLastFullSyncTime(time.Now())
	if err == nil {
		t.Error("SaveLastFullSyncTime should fail with read-only parent directory")
	}
}

func TestReadProjects_FileIsDirectory(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "glf-cache-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	cache := New(tmpDir)

	// Create directory with the same name as cache file
	if err := cache.EnsureDir(); err != nil {
		t.Fatalf("EnsureDir failed: %v", err)
	}
	if err := os.Mkdir(cache.ProjectsPath(), 0755); err != nil {
		t.Fatalf("Failed to create directory: %v", err)
	}

	// Trying to read should fail (file is actually a directory)
	_, err = cache.ReadProjects()
	if err == nil {
		t.Error("ReadProjects should fail when cache file is a directory")
	}
}

func TestLoadLastSyncTime_FileIsDirectory(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "glf-cache-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	cache := New(tmpDir)

	// Create directory with timestamp file name
	timestampPath := filepath.Join(tmpDir, ".last_sync_time")
	if err := cache.EnsureDir(); err != nil {
		t.Fatalf("EnsureDir failed: %v", err)
	}
	if err := os.Mkdir(timestampPath, 0755); err != nil {
		t.Fatalf("Failed to create directory: %v", err)
	}

	_, err = cache.LoadLastSyncTime()
	if err == nil {
		t.Error("LoadLastSyncTime should fail when timestamp file is a directory")
	}
}

func TestLoadLastFullSyncTime_FileIsDirectory(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "glf-cache-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	cache := New(tmpDir)

	// Create directory with timestamp file name
	timestampPath := filepath.Join(tmpDir, ".last_full_sync_time")
	if err := cache.EnsureDir(); err != nil {
		t.Fatalf("EnsureDir failed: %v", err)
	}
	if err := os.Mkdir(timestampPath, 0755); err != nil {
		t.Fatalf("Failed to create directory: %v", err)
	}

	_, err = cache.LoadLastFullSyncTime()
	if err == nil {
		t.Error("LoadLastFullSyncTime should fail when timestamp file is a directory")
	}
}

func TestSaveLastSyncTime_WriteError(t *testing.T) {
	if os.Getuid() == 0 {
		t.Skip("Skipping test when running as root")
	}

	tmpDir, err := os.MkdirTemp("", "glf-cache-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	cache := New(tmpDir)

	// Create directory first
	if err := cache.EnsureDir(); err != nil {
		t.Fatalf("EnsureDir failed: %v", err)
	}

	// Create read-only timestamp file to prevent overwriting
	timestampPath := filepath.Join(tmpDir, ".last_sync_time")
	if err := os.WriteFile(timestampPath, []byte("old"), 0444); err != nil {
		t.Fatalf("Failed to create read-only file: %v", err)
	}
	defer os.Chmod(timestampPath, 0644) // Cleanup

	err = cache.SaveLastSyncTime(time.Now())
	if err == nil {
		t.Error("SaveLastSyncTime should fail when file is read-only")
	}
	if err != nil && !contains(err.Error(), "failed to save sync timestamp") {
		t.Errorf("Expected 'failed to save sync timestamp' in error, got: %v", err)
	}
}

func TestSaveLastFullSyncTime_WriteError(t *testing.T) {
	if os.Getuid() == 0 {
		t.Skip("Skipping test when running as root")
	}

	tmpDir, err := os.MkdirTemp("", "glf-cache-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	cache := New(tmpDir)

	// Create directory first
	if err := cache.EnsureDir(); err != nil {
		t.Fatalf("EnsureDir failed: %v", err)
	}

	// Create read-only timestamp file to prevent overwriting
	timestampPath := filepath.Join(tmpDir, ".last_full_sync_time")
	if err := os.WriteFile(timestampPath, []byte("old"), 0444); err != nil {
		t.Fatalf("Failed to create read-only file: %v", err)
	}
	defer os.Chmod(timestampPath, 0644) // Cleanup

	err = cache.SaveLastFullSyncTime(time.Now())
	if err == nil {
		t.Error("SaveLastFullSyncTime should fail when file is read-only")
	}
	if err != nil && !contains(err.Error(), "failed to save full sync timestamp") {
		t.Errorf("Expected 'failed to save full sync timestamp' in error, got: %v", err)
	}
}

func TestWriteProjects_CreateExistingFile(t *testing.T) {
	if os.Getuid() == 0 {
		t.Skip("Skipping test when running as root")
	}

	tmpDir, err := os.MkdirTemp("", "glf-cache-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	cache := New(tmpDir)

	// Create directory first
	if err := cache.EnsureDir(); err != nil {
		t.Fatalf("EnsureDir failed: %v", err)
	}

	// Create read-only projects file
	projectsPath := cache.ProjectsPath()
	if err := os.WriteFile(projectsPath, []byte("old content"), 0444); err != nil {
		t.Fatalf("Failed to create read-only file: %v", err)
	}
	defer os.Chmod(projectsPath, 0644) // Cleanup

	projects := []model.Project{{Path: "test", Name: "Test"}}
	err = cache.WriteProjects(projects)
	if err == nil {
		t.Error("WriteProjects should fail when cannot create/overwrite file")
	}
	if err != nil && !contains(err.Error(), "failed to create cache file") {
		t.Errorf("Expected 'failed to create cache file' in error, got: %v", err)
	}
}

// Helper functions for string matching
func contains(s, substr string) bool {
	return len(s) >= len(substr) && findSubstring(s, substr)
}

func findSubstring(s, substr string) bool {
	if len(substr) == 0 {
		return true
	}
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// TestSaveLoadUsername tests username caching
func TestSaveLoadUsername(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "glf-cache-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	cache := New(tmpDir)

	// Test save and load
	testUsername := "test-user"
	if err := cache.SaveUsername(testUsername); err != nil {
		t.Fatalf("SaveUsername failed: %v", err)
	}

	loaded, err := cache.LoadUsername()
	if err != nil {
		t.Fatalf("LoadUsername failed: %v", err)
	}

	if loaded != testUsername {
		t.Errorf("Loaded username mismatch: got %q, want %q", loaded, testUsername)
	}
}

// TestLoadUsername_NotCached tests loading when username not cached
func TestLoadUsername_NotCached(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "glf-cache-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	cache := New(tmpDir)

	// Loading without cached username should return empty string
	loaded, err := cache.LoadUsername()
	if err != nil {
		t.Fatalf("LoadUsername should not error when not cached: %v", err)
	}

	if loaded != "" {
		t.Errorf("Not cached username should return empty string, got: %q", loaded)
	}
}

// TestSaveUsername_WriteError tests SaveUsername error handling
func TestSaveUsername_WriteError(t *testing.T) {
	if os.Getuid() == 0 {
		t.Skip("Skipping test when running as root")
	}

	tmpDir, err := os.MkdirTemp("", "glf-cache-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	cache := New(tmpDir)

	// Create directory first
	if err := cache.EnsureDir(); err != nil {
		t.Fatalf("EnsureDir failed: %v", err)
	}

	// Create read-only username file
	usernamePath := filepath.Join(tmpDir, ".username")
	if err := os.WriteFile(usernamePath, []byte("old"), 0444); err != nil {
		t.Fatalf("Failed to create read-only file: %v", err)
	}
	defer os.Chmod(usernamePath, 0644)

	err = cache.SaveUsername("new-user")
	if err == nil {
		t.Error("SaveUsername should fail when file is read-only")
	}
	if err != nil && !contains(err.Error(), "failed to save username") {
		t.Errorf("Expected 'failed to save username' in error, got: %v", err)
	}
}

// TestLoadUsername_WithWhitespace tests trimming of whitespace
func TestLoadUsername_WithWhitespace(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "glf-cache-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	cache := New(tmpDir)

	// Manually write username with whitespace
	usernamePath := filepath.Join(tmpDir, ".username")
	if err := cache.EnsureDir(); err != nil {
		t.Fatalf("EnsureDir failed: %v", err)
	}
	if err := os.WriteFile(usernamePath, []byte("  test-user  \n"), 0644); err != nil {
		t.Fatalf("Failed to write username file: %v", err)
	}

	loaded, err := cache.LoadUsername()
	if err != nil {
		t.Fatalf("LoadUsername failed: %v", err)
	}

	if loaded != "test-user" {
		t.Errorf("Username should be trimmed: got %q, want %q", loaded, "test-user")
	}
}

// TestSaveUsername_EnsureDirError tests SaveUsername when EnsureDir fails
func TestSaveUsername_EnsureDirError(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Skipping test on Windows: chmod doesn't work the same way")
	}
	if os.Getuid() == 0 {
		t.Skip("Skipping test when running as root")
	}

	tmpDir, err := os.MkdirTemp("", "glf-cache-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Make parent directory read-only to prevent EnsureDir from creating subdirectory
	if err := os.Chmod(tmpDir, 0444); err != nil {
		t.Fatalf("Failed to chmod: %v", err)
	}
	defer os.Chmod(tmpDir, 0755) // Restore for cleanup

	cache := New(filepath.Join(tmpDir, "subdir"))

	err = cache.SaveUsername("test-user")
	if err == nil {
		t.Error("SaveUsername should fail when EnsureDir fails")
	}
	if err != nil && !contains(err.Error(), "failed to create cache directory") {
		t.Errorf("Expected 'failed to create cache directory' in error, got: %v", err)
	}
}

// TestLoadUsername_FileIsDirectory tests LoadUsername when .username is a directory
func TestLoadUsername_FileIsDirectory(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "glf-cache-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	cache := New(tmpDir)

	// Create directory with username file name
	usernamePath := filepath.Join(tmpDir, ".username")
	if err := cache.EnsureDir(); err != nil {
		t.Fatalf("EnsureDir failed: %v", err)
	}
	if err := os.Mkdir(usernamePath, 0755); err != nil {
		t.Fatalf("Failed to create directory: %v", err)
	}

	_, err = cache.LoadUsername()
	if err == nil {
		t.Error("LoadUsername should fail when username file is a directory")
	}
	if err != nil && !contains(err.Error(), "failed to read username") {
		t.Errorf("Expected 'failed to read username' in error, got: %v", err)
	}
}

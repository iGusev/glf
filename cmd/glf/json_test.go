package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/igusev/glf/internal/config"
	"github.com/igusev/glf/internal/index"
	"github.com/igusev/glf/internal/types"
)

// TestOutputJSON tests JSON encoding function
func TestOutputJSON(t *testing.T) {
	// Test with simple structure
	data := map[string]string{
		"key": "value",
	}

	// Capture stdout
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := outputJSON(data)

	w.Close()
	os.Stdout = oldStdout

	if err != nil {
		t.Fatalf("outputJSON failed: %v", err)
	}

	// Read captured output
	buf := make([]byte, 1024)
	n, _ := r.Read(buf)
	output := string(buf[:n])

	// Verify JSON structure
	var result map[string]string
	if err := json.Unmarshal([]byte(output), &result); err != nil {
		t.Fatalf("Failed to parse JSON output: %v", err)
	}

	if result["key"] != "value" {
		t.Errorf("Expected 'value', got '%s'", result["key"])
	}
}

// TestRunJSONMode_EmptyProjects tests error handling for empty projects
func TestRunJSONMode_EmptyProjects(t *testing.T) {
	// This will exit the process, so we need to test it differently
	// For now, we'll skip this test as outputJSONError calls os.Exit(1)
	t.Skip("Cannot test outputJSONError directly as it calls os.Exit(1)")
}

// TestRunJSONMode_WithQuery tests JSON output with search query
func TestRunJSONMode_WithQuery(t *testing.T) {
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
		{Path: "frontend/app", Name: "Frontend App", Description: "React application"},
	}

	// Create and populate index
	indexPath := filepath.Join(cacheDir, "description.bleve")
	descIndex, err := index.NewDescriptionIndex(indexPath)
	if err != nil {
		t.Fatalf("Failed to create index: %v", err)
	}

	for _, proj := range projects {
		if err := descIndex.Add(proj.Path, proj.Name, proj.Description); err != nil {
			descIndex.Close()
			t.Fatalf("Failed to add document: %v", err)
		}
	}

	// Set limit for testing
	oldLimit := limitResults
	limitResults = 10
	defer func() { limitResults = oldLimit }()

	// Capture stdout
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	// Run JSON mode with query
	err = runJSONMode(projects, "api", cfg, descIndex)
	descIndex.Close()

	w.Close()
	os.Stdout = oldStdout

	if err != nil {
		t.Fatalf("runJSONMode failed: %v", err)
	}

	// Read captured output
	buf := make([]byte, 8192)
	n, _ := r.Read(buf)
	output := string(buf[:n])

	// Parse JSON output
	var result JSONSearchResult
	if err := json.Unmarshal([]byte(output), &result); err != nil {
		t.Fatalf("Failed to parse JSON output: %v", err)
	}

	// Verify structure
	if result.Query != "api" {
		t.Errorf("Expected query 'api', got '%s'", result.Query)
	}

	if len(result.Results) == 0 {
		t.Error("Expected at least one result, got none")
	}

	// Verify result contains path, name, description, url
	if result.Results[0].Path == "" {
		t.Error("Expected path to be set")
	}
	if result.Results[0].Name == "" {
		t.Error("Expected name to be set")
	}
	if result.Results[0].URL == "" {
		t.Error("Expected URL to be set")
	}

	// Verify limit is set
	if result.Limit != 10 {
		t.Errorf("Expected limit 10, got %d", result.Limit)
	}
}

// TestRunJSONMode_WithoutQuery tests JSON output without query (all projects)
func TestRunJSONMode_WithoutQuery(t *testing.T) {
	tempDir := t.TempDir()
	cacheDir := filepath.Join(tempDir, "cache")
	_ = os.MkdirAll(cacheDir, 0755)

	cfg := &config.Config{
		GitLab: config.GitLabConfig{URL: "https://gitlab.example.com"},
		Cache:  config.CacheConfig{Dir: cacheDir},
	}

	// Create test projects
	projects := []types.Project{
		{Path: "backend/api", Name: "API Server", Description: "REST API"},
		{Path: "frontend/app", Name: "Frontend App", Description: "React app"},
		{Path: "devops/tools", Name: "DevOps Tools", Description: "CI/CD tools"},
	}

	// Create empty index (not used for empty query, but needed for function signature)
	indexPath := filepath.Join(cacheDir, "description.bleve")
	descIndex, err := index.NewDescriptionIndex(indexPath)
	if err != nil {
		t.Fatalf("Failed to create index: %v", err)
	}
	defer descIndex.Close()

	// Set limit to 2 for testing
	oldLimit := limitResults
	limitResults = 2
	defer func() { limitResults = oldLimit }()

	// Capture stdout
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	// Run JSON mode without query (empty string)
	err = runJSONMode(projects, "", cfg, descIndex)

	w.Close()
	os.Stdout = oldStdout

	if err != nil {
		t.Fatalf("runJSONMode without query failed: %v", err)
	}

	// Read captured output
	buf := make([]byte, 8192)
	n, _ := r.Read(buf)
	output := string(buf[:n])

	// Parse JSON output
	var result JSONSearchResult
	if err := json.Unmarshal([]byte(output), &result); err != nil {
		t.Fatalf("Failed to parse JSON output: %v", err)
	}

	// Verify structure
	if result.Query != "" {
		t.Errorf("Expected empty query, got '%s'", result.Query)
	}

	// Should return limited results
	if len(result.Results) != 2 {
		t.Errorf("Expected 2 results (limit), got %d", len(result.Results))
	}

	// Verify total matches limit
	if result.Total != 2 {
		t.Errorf("Expected total 2, got %d", result.Total)
	}

	if result.Limit != 2 {
		t.Errorf("Expected limit 2, got %d", result.Limit)
	}
}

// TestRunJSONMode_WithScores tests JSON output includes scores when --scores flag is set
func TestRunJSONMode_WithScores(t *testing.T) {
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

	// Enable scores flag
	oldShowScores := showScores
	showScores = true
	defer func() { showScores = oldShowScores }()

	// Set limit
	oldLimit := limitResults
	limitResults = 10
	defer func() { limitResults = oldLimit }()

	// Capture stdout
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	// Run JSON mode with query
	err = runJSONMode(projects, "api", cfg, descIndex)
	descIndex.Close()

	w.Close()
	os.Stdout = oldStdout

	if err != nil {
		t.Fatalf("runJSONMode with scores failed: %v", err)
	}

	// Read captured output
	buf := make([]byte, 8192)
	n, _ := r.Read(buf)
	output := string(buf[:n])

	// Parse JSON output
	var result JSONSearchResult
	if err := json.Unmarshal([]byte(output), &result); err != nil {
		t.Fatalf("Failed to parse JSON output: %v", err)
	}

	// Verify scores are included
	if len(result.Results) == 0 {
		t.Fatal("Expected at least one result")
	}

	// Score should be non-zero (search score + any history score)
	if result.Results[0].Score == 0 {
		t.Error("Expected non-zero score when --scores flag is set")
	}
}

// TestRunJSONMode_URLConstruction tests that URLs are properly constructed
func TestRunJSONMode_URLConstruction(t *testing.T) {
	tempDir := t.TempDir()
	cacheDir := filepath.Join(tempDir, "cache")
	_ = os.MkdirAll(cacheDir, 0755)

	// Test different GitLab URL formats
	tests := []struct {
		name       string
		gitlabURL  string
		projectPath string
		expectedURL string
	}{
		{
			name:       "simple URL",
			gitlabURL:  "https://gitlab.com",
			projectPath: "user/project",
			expectedURL: "https://gitlab.com/user/project",
		},
		{
			name:       "URL with port",
			gitlabURL:  "https://gitlab.company.com:8443",
			projectPath: "group/project",
			expectedURL: "https://gitlab.company.com:8443/group/project",
		},
		{
			name:       "URL with trailing slash",
			gitlabURL:  "https://gitlab.com/",
			projectPath: "user/project",
			expectedURL: "https://gitlab.com/user/project",
		},
		{
			name:       "project path with leading slash",
			gitlabURL:  "https://gitlab.com",
			projectPath: "/user/project",
			expectedURL: "https://gitlab.com/user/project",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &config.Config{
				GitLab: config.GitLabConfig{URL: tt.gitlabURL},
				Cache:  config.CacheConfig{Dir: cacheDir},
			}

			projects := []types.Project{
				{Path: tt.projectPath, Name: "Test Project", Description: "Test"},
			}

			// Create empty index
			indexPath := filepath.Join(cacheDir, "description-"+tt.name+".bleve")
			descIndex, err := index.NewDescriptionIndex(indexPath)
			if err != nil {
				t.Fatalf("Failed to create index: %v", err)
			}

			// Set limit
			oldLimit := limitResults
			limitResults = 10
			defer func() { limitResults = oldLimit }()

			// Capture stdout
			oldStdout := os.Stdout
			r, w, _ := os.Pipe()
			os.Stdout = w

			// Run JSON mode without query to get all projects
			err = runJSONMode(projects, "", cfg, descIndex)
			descIndex.Close()

			w.Close()
			os.Stdout = oldStdout

			if err != nil {
				t.Fatalf("runJSONMode failed: %v", err)
			}

			// Read captured output
			buf := make([]byte, 8192)
			n, _ := r.Read(buf)
			output := string(buf[:n])

			// Parse JSON output
			var result JSONSearchResult
			if err := json.Unmarshal([]byte(output), &result); err != nil {
				t.Fatalf("Failed to parse JSON output: %v", err)
			}

			// Verify URL construction
			if len(result.Results) == 0 {
				t.Fatal("Expected at least one result")
			}

			if result.Results[0].URL != tt.expectedURL {
				t.Errorf("Expected URL '%s', got '%s'", tt.expectedURL, result.Results[0].URL)
			}
		})
	}
}

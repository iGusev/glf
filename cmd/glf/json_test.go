package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/igusev/glf/internal/config"
	"github.com/igusev/glf/internal/history"
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

// TestRunJSONMode_LimitEdgeCases tests various limit boundary conditions
func TestRunJSONMode_LimitEdgeCases(t *testing.T) {
	tests := []struct {
		name           string
		limit          int
		totalProjects  int
		expectedCount  int
		expectedTotal  int
	}{
		{
			name:          "limit zero returns all",
			limit:         0,
			totalProjects: 5,
			expectedCount: 5,
			expectedTotal: 5,
		},
		{
			name:          "limit negative returns all",
			limit:         -1,
			totalProjects: 5,
			expectedCount: 5,
			expectedTotal: 5,
		},
		{
			name:          "limit one returns single result",
			limit:         1,
			totalProjects: 5,
			expectedCount: 1,
			expectedTotal: 1,
		},
		{
			name:          "limit exceeds total returns all",
			limit:         100,
			totalProjects: 5,
			expectedCount: 5,
			expectedTotal: 5,
		},
		{
			name:          "limit equals total returns all",
			limit:         5,
			totalProjects: 5,
			expectedCount: 5,
			expectedTotal: 5,
		},
		{
			name:          "limit less than total returns limited",
			limit:         3,
			totalProjects: 10,
			expectedCount: 3,
			expectedTotal: 3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tempDir := t.TempDir()
			cacheDir := filepath.Join(tempDir, "cache")
			_ = os.MkdirAll(cacheDir, 0755)

			cfg := &config.Config{
				GitLab: config.GitLabConfig{URL: "https://gitlab.example.com"},
				Cache:  config.CacheConfig{Dir: cacheDir},
			}

			// Generate projects
			projects := make([]types.Project, tt.totalProjects)
			for i := 0; i < tt.totalProjects; i++ {
				projects[i] = types.Project{
					Path:        filepath.Join("group", "project"+string(rune('A'+i))),
					Name:        "Project " + string(rune('A'+i)),
					Description: "Test project",
				}
			}

			// Create empty index
			indexPath := filepath.Join(cacheDir, "description.bleve")
			descIndex, err := index.NewDescriptionIndex(indexPath)
			if err != nil {
				t.Fatalf("Failed to create index: %v", err)
			}
			defer descIndex.Close()

			// Set limit for test
			oldLimit := limitResults
			limitResults = tt.limit
			defer func() { limitResults = oldLimit }()

			// Capture stdout
			oldStdout := os.Stdout
			r, w, _ := os.Pipe()
			os.Stdout = w

			// Run without query to get all projects
			err = runJSONMode(projects, "", cfg, descIndex)

			w.Close()
			os.Stdout = oldStdout

			if err != nil {
				t.Fatalf("runJSONMode failed: %v", err)
			}

			// Read and parse output
			buf := make([]byte, 16384)
			n, _ := r.Read(buf)
			output := string(buf[:n])

			var result JSONSearchResult
			if err := json.Unmarshal([]byte(output), &result); err != nil {
				t.Fatalf("Failed to parse JSON: %v", err)
			}

			// Verify result count
			if len(result.Results) != tt.expectedCount {
				t.Errorf("Expected %d results, got %d", tt.expectedCount, len(result.Results))
			}

			// Verify total
			if result.Total != tt.expectedTotal {
				t.Errorf("Expected total %d, got %d", tt.expectedTotal, result.Total)
			}
		})
	}
}

// TestRunJSONMode_SpecialCharacters tests queries with special characters and Unicode
func TestRunJSONMode_SpecialCharacters(t *testing.T) {
	tests := []struct {
		name  string
		query string
	}{
		{"spaces", "api backend"},
		{"at symbol", "user@domain"},
		{"hash symbol", "issue#123"},
		{"dash", "project-name"},
		{"underscore", "snake_case"},
		{"unicode", "café-tëst"},
		{"emoji", "api🚀project"},
		{"mixed", "Project@2024-v1.0"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tempDir := t.TempDir()
			cacheDir := filepath.Join(tempDir, "cache")
			_ = os.MkdirAll(cacheDir, 0755)

			cfg := &config.Config{
				GitLab: config.GitLabConfig{URL: "https://gitlab.example.com"},
				Cache:  config.CacheConfig{Dir: cacheDir},
			}

			projects := []types.Project{
				{Path: "test/project", Name: "Test", Description: tt.query},
			}

			// Create and populate index
			indexPath := filepath.Join(cacheDir, "description.bleve")
			descIndex, err := index.NewDescriptionIndex(indexPath)
			if err != nil {
				t.Fatalf("Failed to create index: %v", err)
			}

			if err := descIndex.Add(projects[0].Path, projects[0].Name, projects[0].Description); err != nil {
				descIndex.Close()
				t.Fatalf("Failed to add to index: %v", err)
			}

			// Set limit
			oldLimit := limitResults
			limitResults = 10
			defer func() { limitResults = oldLimit }()

			// Capture stdout
			oldStdout := os.Stdout
			r, w, _ := os.Pipe()
			os.Stdout = w

			err = runJSONMode(projects, tt.query, cfg, descIndex)
			descIndex.Close()

			w.Close()
			os.Stdout = oldStdout

			if err != nil {
				t.Fatalf("runJSONMode failed: %v", err)
			}

			// Read and parse output
			buf := make([]byte, 8192)
			n, _ := r.Read(buf)
			output := string(buf[:n])

			var result JSONSearchResult
			if err := json.Unmarshal([]byte(output), &result); err != nil {
				t.Fatalf("Failed to parse JSON: %v", err)
			}

			// Verify query is preserved exactly
			if result.Query != tt.query {
				t.Errorf("Expected query '%s', got '%s'", tt.query, result.Query)
			}

			// Verify JSON is valid (we successfully unmarshaled)
			if result.Results == nil {
				t.Error("Results should not be nil")
			}
		})
	}
}

// TestRunJSONMode_EmptyResults tests query that returns no matches
func TestRunJSONMode_EmptyResults(t *testing.T) {
	tempDir := t.TempDir()
	cacheDir := filepath.Join(tempDir, "cache")
	_ = os.MkdirAll(cacheDir, 0755)

	cfg := &config.Config{
		GitLab: config.GitLabConfig{URL: "https://gitlab.example.com"},
		Cache:  config.CacheConfig{Dir: cacheDir},
	}

	// Create projects that won't match
	projects := []types.Project{
		{Path: "backend/api", Name: "API Server", Description: "REST API backend"},
		{Path: "frontend/app", Name: "Frontend", Description: "React application"},
		{Path: "devops/ci", Name: "DevOps", Description: "CI/CD pipeline"},
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
			t.Fatalf("Failed to add to index: %v", err)
		}
	}

	// Set limit
	oldLimit := limitResults
	limitResults = 10
	defer func() { limitResults = oldLimit }()

	// Capture stdout
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	// Query that matches nothing
	err = runJSONMode(projects, "zzznomatchxxx", cfg, descIndex)
	descIndex.Close()

	w.Close()
	os.Stdout = oldStdout

	if err != nil {
		t.Fatalf("runJSONMode failed: %v", err)
	}

	// Read and parse output
	buf := make([]byte, 8192)
	n, _ := r.Read(buf)
	output := string(buf[:n])

	var result JSONSearchResult
	if err := json.Unmarshal([]byte(output), &result); err != nil {
		t.Fatalf("Failed to parse JSON: %v", err)
	}

	// Verify empty results (not nil, but empty array)
	if result.Results == nil {
		t.Error("Results should not be nil, should be empty array")
	}

	if len(result.Results) != 0 {
		t.Errorf("Expected 0 results, got %d", len(result.Results))
	}

	if result.Total != 0 {
		t.Errorf("Expected total 0, got %d", result.Total)
	}
}

// TestRunJSONMode_LargeResultSet tests performance with many projects
func TestRunJSONMode_LargeResultSet(t *testing.T) {
	tempDir := t.TempDir()
	cacheDir := filepath.Join(tempDir, "cache")
	_ = os.MkdirAll(cacheDir, 0755)

	cfg := &config.Config{
		GitLab: config.GitLabConfig{URL: "https://gitlab.example.com"},
		Cache:  config.CacheConfig{Dir: cacheDir},
	}

	// Generate 100 projects
	const totalProjects = 100
	projects := make([]types.Project, totalProjects)
	for i := 0; i < totalProjects; i++ {
		projects[i] = types.Project{
			Path:        filepath.Join("group", "project", "subproject"+string(rune('0'+i%10)), "item"+string(rune('A'+i/10))),
			Name:        "Project " + string(rune('A'+i/26)) + string(rune('A'+i%26)),
			Description: "Test project number " + string(rune('0'+i%10)),
		}
	}

	// Create index (don't populate for speed - we're testing without query)
	indexPath := filepath.Join(cacheDir, "description.bleve")
	descIndex, err := index.NewDescriptionIndex(indexPath)
	if err != nil {
		t.Fatalf("Failed to create index: %v", err)
	}
	defer descIndex.Close()

	// Set limit to 20
	oldLimit := limitResults
	limitResults = 20
	defer func() { limitResults = oldLimit }()

	// Capture stdout
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	// Run without query to get all projects (limited to 20)
	err = runJSONMode(projects, "", cfg, descIndex)

	w.Close()
	os.Stdout = oldStdout

	if err != nil {
		t.Fatalf("runJSONMode failed: %v", err)
	}

	// Read and parse output
	buf := make([]byte, 32768)
	n, _ := r.Read(buf)
	output := string(buf[:n])

	var result JSONSearchResult
	if err := json.Unmarshal([]byte(output), &result); err != nil {
		t.Fatalf("Failed to parse JSON: %v", err)
	}

	// Verify exactly 20 results returned (limit applied)
	if len(result.Results) != 20 {
		t.Errorf("Expected 20 results (limit), got %d", len(result.Results))
	}

	// Verify total is 20 (after limiting)
	if result.Total != 20 {
		t.Errorf("Expected total 20, got %d", result.Total)
	}

	// Verify limit is set
	if result.Limit != 20 {
		t.Errorf("Expected limit 20, got %d", result.Limit)
	}
}

// TestRunJSONMode_HistoryLoadError tests graceful handling of corrupted history
func TestRunJSONMode_HistoryLoadError(t *testing.T) {
	tempDir := t.TempDir()
	cacheDir := filepath.Join(tempDir, "cache")
	_ = os.MkdirAll(cacheDir, 0755)

	cfg := &config.Config{
		GitLab: config.GitLabConfig{URL: "https://gitlab.example.com"},
		Cache:  config.CacheConfig{Dir: cacheDir},
	}

	projects := []types.Project{
		{Path: "test/project", Name: "Test", Description: "Test project"},
	}

	// Create corrupted history.gob file
	historyPath := filepath.Join(cacheDir, "history.gob")
	corruptedData := []byte{0x00, 0xFF, 0xDE, 0xAD, 0xBE, 0xEF, 0x00, 0xFF}
	if err := os.WriteFile(historyPath, corruptedData, 0644); err != nil {
		t.Fatalf("Failed to write corrupted history: %v", err)
	}

	// Create and populate index
	indexPath := filepath.Join(cacheDir, "description.bleve")
	descIndex, err := index.NewDescriptionIndex(indexPath)
	if err != nil {
		t.Fatalf("Failed to create index: %v", err)
	}

	if err := descIndex.Add(projects[0].Path, projects[0].Name, projects[0].Description); err != nil {
		descIndex.Close()
		t.Fatalf("Failed to add to index: %v", err)
	}

	// Set limit
	oldLimit := limitResults
	limitResults = 10
	defer func() { limitResults = oldLimit }()

	// Capture stdout
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	// Should still work despite corrupted history
	err = runJSONMode(projects, "test", cfg, descIndex)
	descIndex.Close()

	w.Close()
	os.Stdout = oldStdout

	if err != nil {
		t.Fatalf("runJSONMode should handle corrupted history gracefully: %v", err)
	}

	// Read and parse output
	buf := make([]byte, 8192)
	n, _ := r.Read(buf)
	output := string(buf[:n])

	var result JSONSearchResult
	if err := json.Unmarshal([]byte(output), &result); err != nil {
		t.Fatalf("Failed to parse JSON: %v", err)
	}

	// Should still get results
	if len(result.Results) == 0 {
		t.Error("Expected results even with corrupted history")
	}
}

// TestRunJSONMode_MultiTokenQuery tests multi-word query search
func TestRunJSONMode_MultiTokenQuery(t *testing.T) {
	tempDir := t.TempDir()
	cacheDir := filepath.Join(tempDir, "cache")
	_ = os.MkdirAll(cacheDir, 0755)

	cfg := &config.Config{
		GitLab: config.GitLabConfig{URL: "https://gitlab.example.com"},
		Cache:  config.CacheConfig{Dir: cacheDir},
	}

	// Create projects with varying matches
	projects := []types.Project{
		{Path: "backend/api-gateway", Name: "API Gateway", Description: "Gateway service for APIs"},
		{Path: "backend/service", Name: "Backend Service", Description: "Core backend logic"},
		{Path: "backend/api-core", Name: "API Backend Core", Description: "Core API backend service"},
		{Path: "frontend/app", Name: "Frontend", Description: "React application"},
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
			t.Fatalf("Failed to add to index: %v", err)
		}
	}

	// Set limit
	oldLimit := limitResults
	limitResults = 10
	defer func() { limitResults = oldLimit }()

	// Capture stdout
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	// Multi-token query
	err = runJSONMode(projects, "api backend", cfg, descIndex)
	descIndex.Close()

	w.Close()
	os.Stdout = oldStdout

	if err != nil {
		t.Fatalf("runJSONMode failed: %v", err)
	}

	// Read and parse output
	buf := make([]byte, 8192)
	n, _ := r.Read(buf)
	output := string(buf[:n])

	var result JSONSearchResult
	if err := json.Unmarshal([]byte(output), &result); err != nil {
		t.Fatalf("Failed to parse JSON: %v", err)
	}

	// Should have results (projects containing "api" and/or "backend")
	if len(result.Results) == 0 {
		t.Error("Expected results for multi-token query")
	}

	// Verify query is preserved
	if result.Query != "api backend" {
		t.Errorf("Expected query 'api backend', got '%s'", result.Query)
	}

	// "API Backend Core" should appear in results (contains both tokens)
	found := false
	for _, proj := range result.Results {
		if proj.Path == "backend/api-core" {
			found = true
			break
		}
	}

	if !found {
		t.Error("Expected 'backend/api-core' in results (contains both 'api' and 'backend')")
	}
}

// TestRunJSONMode_ProjectPathEdgeCases tests various project path formats
func TestRunJSONMode_ProjectPathEdgeCases(t *testing.T) {
	tests := []struct {
		name        string
		projectPath string
		description string
	}{
		{"dots in path", "group.subgroup/project.name", "Project with dots"},
		{"multiple slashes", "group/subgroup/deep/project", "Deep nested project"},
		{"numbers", "project123/sub456", "Project with numbers"},
		{"mixed case", "MyGroup/MyProject", "Mixed case project"},
		{"underscores", "group_name/project_name", "Project with underscores"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tempDir := t.TempDir()
			cacheDir := filepath.Join(tempDir, "cache")
			_ = os.MkdirAll(cacheDir, 0755)

			cfg := &config.Config{
				GitLab: config.GitLabConfig{URL: "https://gitlab.example.com"},
				Cache:  config.CacheConfig{Dir: cacheDir},
			}

			projects := []types.Project{
				{Path: tt.projectPath, Name: "Test Project", Description: tt.description},
			}

			// Create index
			indexPath := filepath.Join(cacheDir, "description.bleve")
			descIndex, err := index.NewDescriptionIndex(indexPath)
			if err != nil {
				t.Fatalf("Failed to create index: %v", err)
			}
			defer descIndex.Close()

			// Set limit
			oldLimit := limitResults
			limitResults = 10
			defer func() { limitResults = oldLimit }()

			// Capture stdout
			oldStdout := os.Stdout
			r, w, _ := os.Pipe()
			os.Stdout = w

			err = runJSONMode(projects, "", cfg, descIndex)

			w.Close()
			os.Stdout = oldStdout

			if err != nil {
				t.Fatalf("runJSONMode failed: %v", err)
			}

			// Read and parse output
			buf := make([]byte, 8192)
			n, _ := r.Read(buf)
			output := string(buf[:n])

			var result JSONSearchResult
			if err := json.Unmarshal([]byte(output), &result); err != nil {
				t.Fatalf("Failed to parse JSON: %v", err)
			}

			if len(result.Results) == 0 {
				t.Fatal("Expected at least one result")
			}

			// Verify path is preserved
			if result.Results[0].Path != tt.projectPath {
				t.Errorf("Expected path '%s', got '%s'", tt.projectPath, result.Results[0].Path)
			}

			// Verify URL construction works
			expectedURL := "https://gitlab.example.com/" + tt.projectPath
			if result.Results[0].URL != expectedURL {
				t.Errorf("Expected URL '%s', got '%s'", expectedURL, result.Results[0].URL)
			}
		})
	}
}

// TestRunJSONMode_HistoryScoreIntegration tests history score boosting with --scores
func TestRunJSONMode_HistoryScoreIntegration(t *testing.T) {
	tempDir := t.TempDir()
	cacheDir := filepath.Join(tempDir, "cache")
	_ = os.MkdirAll(cacheDir, 0755)

	cfg := &config.Config{
		GitLab: config.GitLabConfig{URL: "https://gitlab.example.com"},
		Cache:  config.CacheConfig{Dir: cacheDir},
	}

	projects := []types.Project{
		{Path: "backend/api-server", Name: "API Server", Description: "REST API backend"},
		{Path: "backend/worker", Name: "Worker", Description: "Background jobs"},
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
			t.Fatalf("Failed to add to index: %v", err)
		}
	}

	// Create history and record selections
	historyPath := filepath.Join(cacheDir, "history.gob")
	hist := history.New(historyPath)

	// Record multiple selections with query context
	for i := 0; i < 10; i++ {
		hist.RecordSelectionWithQuery("api", "backend/api-server")
	}

	// Save history
	if err := hist.Save(); err != nil {
		t.Fatalf("Failed to save history: %v", err)
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

	err = runJSONMode(projects, "api", cfg, descIndex)
	descIndex.Close()

	w.Close()
	os.Stdout = oldStdout

	if err != nil {
		t.Fatalf("runJSONMode failed: %v", err)
	}

	// Read and parse output
	buf := make([]byte, 8192)
	n, _ := r.Read(buf)
	output := string(buf[:n])

	var result JSONSearchResult
	if err := json.Unmarshal([]byte(output), &result); err != nil {
		t.Fatalf("Failed to parse JSON: %v", err)
	}

	if len(result.Results) == 0 {
		t.Fatal("Expected at least one result")
	}

	// Find the API server in results
	var apiServerScore float64
	found := false
	for _, proj := range result.Results {
		if proj.Path == "backend/api-server" {
			apiServerScore = proj.Score
			found = true
			break
		}
	}

	if !found {
		t.Error("Expected 'backend/api-server' in results")
	}

	// Score should be significantly boosted by history (>100 from 10 selections)
	if apiServerScore < 100 {
		t.Errorf("Expected history-boosted score >100, got %.2f", apiServerScore)
	}
}

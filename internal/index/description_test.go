package index

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/blevesearch/bleve/v2/search"
	"github.com/igusev/glf/internal/model"
)

func TestNewDescriptionIndex(t *testing.T) {
	tests := []struct {
		name      string
		wantError bool
	}{
		{
			name:      "create new index",
			wantError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Use temp directory for test index
			tempDir := t.TempDir()
			indexPath := filepath.Join(tempDir, "test.bleve")

			di, err := NewDescriptionIndex(indexPath)
			if (err != nil) != tt.wantError {
				t.Errorf("NewDescriptionIndex() error = %v, wantError %v", err, tt.wantError)
				return
			}

			if !tt.wantError {
				if di == nil {
					t.Error("Expected non-nil DescriptionIndex")
				}
				defer di.Close()

				// Verify index was created
				if !Exists(indexPath) {
					t.Error("Index file should exist")
				}
			}
		})
	}
}

func TestDescriptionIndex_OpenExisting(t *testing.T) {
	tempDir := t.TempDir()
	indexPath := filepath.Join(tempDir, "test.bleve")

	// Create index
	di1, err := NewDescriptionIndex(indexPath)
	if err != nil {
		t.Fatalf("Failed to create index: %v", err)
	}

	// Add a document
	err = di1.Add("group/project", "Test Project", "A test description", false, false)
	if err != nil {
		t.Fatalf("Failed to add document: %v", err)
	}

	// Close it
	if err := di1.Close(); err != nil {
		t.Fatalf("Failed to close index: %v", err)
	}

	// Reopen the same index
	di2, err := NewDescriptionIndex(indexPath)
	if err != nil {
		t.Fatalf("Failed to open existing index: %v", err)
	}
	defer di2.Close()

	// Verify document still exists
	// Note: Count includes the version document, so expect 2 (1 project + 1 version)
	count, err := di2.Count()
	if err != nil {
		t.Fatalf("Failed to count documents: %v", err)
	}

	if count != 2 {
		t.Errorf("Expected 2 documents (1 project + 1 version), got %d", count)
	}
}

func TestDescriptionIndex_Add(t *testing.T) {
	tempDir := t.TempDir()
	indexPath := filepath.Join(tempDir, "test.bleve")

	di, err := NewDescriptionIndex(indexPath)
	if err != nil {
		t.Fatalf("Failed to create index: %v", err)
	}
	defer di.Close()

	tests := []struct {
		projectPath string
		projectName string
		description string
	}{
		{
			projectPath: "group/backend",
			projectName: "Backend API",
			description: "RESTful API for the application",
		},
		{
			projectPath: "group/frontend",
			projectName: "Frontend App",
			description: "React-based user interface",
		},
	}

	for _, tt := range tests {
		t.Run(tt.projectPath, func(t *testing.T) {
			err := di.Add(tt.projectPath, tt.projectName, tt.description, false, false)
			if err != nil {
				t.Errorf("Add() error = %v", err)
			}
		})
	}

	// Verify count (includes version document)
	count, err := di.Count()
	if err != nil {
		t.Fatalf("Failed to count: %v", err)
	}

	expected := uint64(len(tests) + 1) // +1 for version document
	if count != expected {
		t.Errorf("Expected %d documents (%d projects + 1 version), got %d", expected, len(tests), count)
	}
}

func TestDescriptionIndex_AddBatch(t *testing.T) {
	tempDir := t.TempDir()
	indexPath := filepath.Join(tempDir, "test.bleve")

	di, err := NewDescriptionIndex(indexPath)
	if err != nil {
		t.Fatalf("Failed to create index: %v", err)
	}
	defer di.Close()

	docs := []DescriptionDocument{
		{
			ProjectPath: "org/project1",
			ProjectName: "Project One",
			Description: "First test project",
		},
		{
			ProjectPath: "org/project2",
			ProjectName: "Project Two",
			Description: "Second test project",
		},
		{
			ProjectPath: "org/project3",
			ProjectName: "Project Three",
			Description: "Third test project",
		},
	}

	err = di.AddBatch(docs)
	if err != nil {
		t.Fatalf("AddBatch() error = %v", err)
	}

	// Verify count (includes version document)
	count, err := di.Count()
	if err != nil {
		t.Fatalf("Failed to count: %v", err)
	}

	expected := uint64(4) // 3 projects + 1 version document
	if count != expected {
		t.Errorf("Expected %d documents (3 projects + 1 version), got %d", expected, count)
	}
}

func TestDescriptionIndex_Search_SingleToken(t *testing.T) {
	tempDir := t.TempDir()
	indexPath := filepath.Join(tempDir, "test.bleve")

	di, err := NewDescriptionIndex(indexPath)
	if err != nil {
		t.Fatalf("Failed to create index: %v", err)
	}
	defer di.Close()

	// Add test documents
	testDocs := []DescriptionDocument{
		{
			ProjectPath: "backend/auth",
			ProjectName: "Authentication Service",
			Description: "Handles user authentication and authorization with JWT tokens",
		},
		{
			ProjectPath: "backend/api",
			ProjectName: "API Gateway",
			Description: "Central API gateway for microservices",
		},
		{
			ProjectPath: "frontend/dashboard",
			ProjectName: "Admin Dashboard",
			Description: "Administrative interface for system management",
		},
	}

	err = di.AddBatch(testDocs)
	if err != nil {
		t.Fatalf("Failed to add batch: %v", err)
	}

	tests := []struct {
		query         string
		minResults    int
		expectedFirst string // Expected first result's project path
	}{
		{
			query:         "auth",
			minResults:    1,
			expectedFirst: "backend/auth",
		},
		{
			query:         "api",
			minResults:    1,
			expectedFirst: "backend/api",
		},
		{
			query:         "dashboard",
			minResults:    1,
			expectedFirst: "frontend/dashboard",
		},
		{
			query:         "authentication",
			minResults:    1,
			expectedFirst: "backend/auth",
		},
	}

	for _, tt := range tests {
		t.Run(tt.query, func(t *testing.T) {
			matches, err := di.Search(tt.query, 10)
			if err != nil {
				t.Fatalf("Search() error = %v", err)
			}

			if len(matches) < tt.minResults {
				t.Errorf("Expected at least %d results, got %d", tt.minResults, len(matches))
				return
			}

			if matches[0].Project.Path != tt.expectedFirst {
				t.Errorf("Expected first result '%s', got '%s'", tt.expectedFirst, matches[0].Project.Path)
			}

			// Verify all results have valid scores
			for i, match := range matches {
				if match.Score <= 0 {
					t.Errorf("Result %d has invalid score: %f", i, match.Score)
				}
			}
		})
	}
}

func TestDescriptionIndex_Search_MultipleTokens(t *testing.T) {
	tempDir := t.TempDir()
	indexPath := filepath.Join(tempDir, "test.bleve")

	di, err := NewDescriptionIndex(indexPath)
	if err != nil {
		t.Fatalf("Failed to create index: %v", err)
	}
	defer di.Close()

	// Add test documents
	testDocs := []DescriptionDocument{
		{
			ProjectPath: "backend/user-service",
			ProjectName: "User Management Service",
			Description: "Service for managing user accounts and profiles",
		},
		{
			ProjectPath: "backend/auth-service",
			ProjectName: "Authentication Service",
			Description: "Handles authentication and authorization",
		},
		{
			ProjectPath: "frontend/user-dashboard",
			ProjectName: "User Dashboard",
			Description: "Dashboard for user profile management",
		},
	}

	err = di.AddBatch(testDocs)
	if err != nil {
		t.Fatalf("Failed to add batch: %v", err)
	}

	tests := []struct {
		query         string
		expectResults bool
		checkFirst    bool
		expectedFirst string
	}{
		{
			query:         "user service",
			expectResults: true,
			checkFirst:    true,
			expectedFirst: "backend/user-service", // Should match both tokens
		},
		{
			query:         "user management",
			expectResults: true,
			checkFirst:    false, // Multiple valid matches, don't check order
		},
		{
			query:         "auth service",
			expectResults: true,
			checkFirst:    true,
			expectedFirst: "backend/auth-service",
		},
	}

	for _, tt := range tests {
		t.Run(tt.query, func(t *testing.T) {
			matches, err := di.Search(tt.query, 10)
			if err != nil {
				t.Fatalf("Search() error = %v", err)
			}

			if tt.expectResults && len(matches) == 0 {
				t.Error("Expected results but got none")
			}

			if tt.checkFirst && len(matches) > 0 && matches[0].Project.Path != tt.expectedFirst {
				t.Errorf("Expected first result '%s', got '%s'", tt.expectedFirst, matches[0].Project.Path)
			}
		})
	}
}

func TestDescriptionIndex_Search_EmptyQuery(t *testing.T) {
	tempDir := t.TempDir()
	indexPath := filepath.Join(tempDir, "test.bleve")

	di, err := NewDescriptionIndex(indexPath)
	if err != nil {
		t.Fatalf("Failed to create index: %v", err)
	}
	defer di.Close()

	matches, err := di.Search("", 10)
	if err != nil {
		t.Fatalf("Search() error = %v", err)
	}

	if len(matches) != 0 {
		t.Errorf("Expected 0 results for empty query, got %d", len(matches))
	}
}

func TestDescriptionIndex_Search_FuzzyMatching(t *testing.T) {
	tempDir := t.TempDir()
	indexPath := filepath.Join(tempDir, "test.bleve")

	di, err := NewDescriptionIndex(indexPath)
	if err != nil {
		t.Fatalf("Failed to create index: %v", err)
	}
	defer di.Close()

	// Add document with "template"
	err = di.Add("tools/template-engine", "Template Engine", "Advanced templating system", false, false)
	if err != nil {
		t.Fatalf("Failed to add document: %v", err)
	}

	// Test fuzzy matching (typo tolerance)
	matches, err := di.Search("tmeplate", 10) // Typo: missing 'e', extra 'm'
	if err != nil {
		t.Fatalf("Search() error = %v", err)
	}

	if len(matches) == 0 {
		t.Error("Expected fuzzy match for 'tmeplate' → 'template', got no results")
	}
}

func TestDescriptionIndex_Search_PrefixMatching(t *testing.T) {
	tempDir := t.TempDir()
	indexPath := filepath.Join(tempDir, "test.bleve")

	di, err := NewDescriptionIndex(indexPath)
	if err != nil {
		t.Fatalf("Failed to create index: %v", err)
	}
	defer di.Close()

	// Add document
	err = di.Add("backend/kubernetes", "Kubernetes Deployment", "K8s deployment configuration", false, false)
	if err != nil {
		t.Fatalf("Failed to add document: %v", err)
	}

	// Test prefix matching
	tests := []string{"kub", "kuber", "kubernet"}
	for _, query := range tests {
		t.Run(query, func(t *testing.T) {
			matches, err := di.Search(query, 10)
			if err != nil {
				t.Fatalf("Search() error = %v", err)
			}

			if len(matches) == 0 {
				t.Errorf("Expected prefix match for '%s' → 'kubernetes', got no results", query)
			}
		})
	}
}

func TestDescriptionIndex_Delete(t *testing.T) {
	tempDir := t.TempDir()
	indexPath := filepath.Join(tempDir, "test.bleve")

	di, err := NewDescriptionIndex(indexPath)
	if err != nil {
		t.Fatalf("Failed to create index: %v", err)
	}
	defer di.Close()

	// Add documents
	docs := []DescriptionDocument{
		{ProjectPath: "project1", ProjectName: "P1", Description: "Description 1"},
		{ProjectPath: "project2", ProjectName: "P2", Description: "Description 2"},
	}

	err = di.AddBatch(docs)
	if err != nil {
		t.Fatalf("Failed to add batch: %v", err)
	}

	// Verify initial count (includes version document)
	count, _ := di.Count()
	expected := uint64(3) // 2 projects + 1 version document
	if count != expected {
		t.Fatalf("Expected %d documents (2 projects + 1 version), got %d", expected, count)
	}

	// Delete one document
	err = di.Delete("project1")
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}

	// Verify count decreased (still includes version document)
	count, _ = di.Count()
	expected = uint64(2) // 1 project + 1 version document
	if count != expected {
		t.Errorf("Expected %d documents (1 project + 1 version) after delete, got %d", expected, count)
	}

	// Verify correct document was deleted by searching for specific path
	matches, _ := di.Search("P2", 10) // Search for non-deleted project name
	if len(matches) == 0 {
		t.Error("Non-deleted document should still be searchable")
	}

	// Verify only one project remains
	found := false
	for _, match := range matches {
		if match.Project.Path == "project1" {
			t.Error("Deleted document should not be found in search")
		}
		if match.Project.Path == "project2" {
			found = true
		}
	}

	if !found {
		t.Error("Non-deleted project2 should be found")
	}
}

func TestDescriptionIndex_Count(t *testing.T) {
	tempDir := t.TempDir()
	indexPath := filepath.Join(tempDir, "test.bleve")

	di, err := NewDescriptionIndex(indexPath)
	if err != nil {
		t.Fatalf("Failed to create index: %v", err)
	}
	defer di.Close()

	// Initially contains only version document
	count, err := di.Count()
	if err != nil {
		t.Fatalf("Count() error = %v", err)
	}
	if count != 1 {
		t.Errorf("Expected count 1 (version document only), got %d", count)
	}

	// Add documents
	for i := 1; i <= 5; i++ {
		err = di.Add("project"+string(rune('0'+i)), "P", "D", false, false)
		if err != nil {
			t.Fatalf("Failed to add document: %v", err)
		}
	}

	count, err = di.Count()
	if err != nil {
		t.Fatalf("Count() error = %v", err)
	}
	expected := uint64(6) // 5 projects + 1 version document
	if count != expected {
		t.Errorf("Expected count %d (5 projects + 1 version), got %d", expected, count)
	}
}

func TestExists(t *testing.T) {
	tempDir := t.TempDir()
	existingPath := filepath.Join(tempDir, "existing.bleve")
	nonExistentPath := filepath.Join(tempDir, "nonexistent.bleve")

	// Create an index
	di, err := NewDescriptionIndex(existingPath)
	if err != nil {
		t.Fatalf("Failed to create index: %v", err)
	}
	di.Close()

	// Test existing index
	if !Exists(existingPath) {
		t.Error("Exists() should return true for existing index")
	}

	// Test non-existent index
	if Exists(nonExistentPath) {
		t.Error("Exists() should return false for non-existent index")
	}
}

func TestDescriptionIndex_GetAllProjects(t *testing.T) {
	tempDir := t.TempDir()
	indexPath := filepath.Join(tempDir, "test.bleve")

	di, err := NewDescriptionIndex(indexPath)
	if err != nil {
		t.Fatalf("Failed to create index: %v", err)
	}
	defer di.Close()

	// Test empty index (version document should be filtered out)
	projects, err := di.GetAllProjects()
	if err != nil {
		t.Fatalf("GetAllProjects() error = %v", err)
	}
	if len(projects) != 0 {
		t.Errorf("Expected 0 projects in empty index (version doc filtered), got %d", len(projects))
	}

	// Add documents
	testDocs := []DescriptionDocument{
		{ProjectPath: "org/p1", ProjectName: "Project 1", Description: "Desc 1"},
		{ProjectPath: "org/p2", ProjectName: "Project 2", Description: "Desc 2"},
		{ProjectPath: "org/p3", ProjectName: "Project 3", Description: "Desc 3"},
	}

	err = di.AddBatch(testDocs)
	if err != nil {
		t.Fatalf("Failed to add batch: %v", err)
	}

	// Get all projects (version document should be filtered out)
	projects, err = di.GetAllProjects()
	if err != nil {
		t.Fatalf("GetAllProjects() error = %v", err)
	}

	if len(projects) != 3 {
		t.Errorf("Expected 3 projects (version doc filtered), got %d", len(projects))
	}

	// Verify projects have correct data
	pathMap := make(map[string]model.Project)
	for _, p := range projects {
		pathMap[p.Path] = p
	}

	for _, expected := range testDocs {
		project, exists := pathMap[expected.ProjectPath]
		if !exists {
			t.Errorf("Expected project '%s' not found", expected.ProjectPath)
			continue
		}

		if project.Name != expected.ProjectName {
			t.Errorf("Project %s: expected name '%s', got '%s'", expected.ProjectPath, expected.ProjectName, project.Name)
		}

		if project.Description != expected.Description {
			t.Errorf("Project %s: expected description '%s', got '%s'", expected.ProjectPath, expected.Description, project.Description)
		}
	}
}

func TestDescriptionIndex_Search_FieldBoosting(t *testing.T) {
	tempDir := t.TempDir()
	indexPath := filepath.Join(tempDir, "test.bleve")

	di, err := NewDescriptionIndex(indexPath)
	if err != nil {
		t.Fatalf("Failed to create index: %v", err)
	}
	defer di.Close()

	// Add documents where "api" appears in different fields
	testDocs := []DescriptionDocument{
		{
			ProjectPath: "backend/api-gateway", // "api" in path (5x boost)
			ProjectName: "Gateway Service",
			Description: "Microservices gateway",
		},
		{
			ProjectPath: "backend/service",
			ProjectName: "API Service", // "api" in name (10x boost) - should rank highest
			Description: "REST API implementation",
		},
		{
			ProjectPath: "backend/handler",
			ProjectName: "Request Handler",
			Description: "Handles API requests", // "api" only in description (1x boost)
		},
	}

	err = di.AddBatch(testDocs)
	if err != nil {
		t.Fatalf("Failed to add batch: %v", err)
	}

	matches, err := di.Search("api", 10)
	if err != nil {
		t.Fatalf("Search() error = %v", err)
	}

	if len(matches) < 3 {
		t.Fatalf("Expected at least 3 results, got %d", len(matches))
	}

	// Result with "api" in ProjectName should rank highest (10x boost)
	if matches[0].Project.Path != "backend/service" {
		t.Errorf("Expected 'backend/service' (name match) to rank highest, got '%s'", matches[0].Project.Path)
	}

	// Verify scores are descending
	for i := 1; i < len(matches); i++ {
		if matches[i].Score > matches[i-1].Score {
			t.Errorf("Scores should be descending: result %d (%.2f) > result %d (%.2f)",
				i, matches[i].Score, i-1, matches[i-1].Score)
		}
	}
}

func TestStripHTMLTags(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{
			input:    "plain text",
			expected: "plain text",
		},
		{
			input:    "text with <mark>highlighted</mark> word",
			expected: "text with highlighted word",
		},
		{
			input:    "<b>bold</b> and <i>italic</i>",
			expected: "bold and italic",
		},
		{
			input:    "nested <div>tags <span>here</span></div>",
			expected: "nested tags here",
		},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := stripHTMLTags(tt.input)
			if result != tt.expected {
				t.Errorf("stripHTMLTags(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestExtractSnippet_WithFragments(t *testing.T) {
	tests := []struct {
		name        string
		fragments   map[string][]string
		expected    string
		description string
	}{
		{
			name: "single fragment",
			fragments: map[string][]string{
				"Description": {"This is a <mark>test</mark> fragment"},
			},
			expected:    "This is a test fragment",
			description: "Should use fragment and strip HTML tags",
		},
		{
			name: "two fragments",
			fragments: map[string][]string{
				"Description": {"First <mark>fragment</mark>", "Second fragment"},
			},
			expected:    "First fragment ... Second fragment",
			description: "Should join both fragments with separator",
		},
		{
			name: "more than two fragments",
			fragments: map[string][]string{
				"Description": {"First", "Second", "Third", "Fourth"},
			},
			expected:    "First ... Second",
			description: "Should limit to first 2 fragments",
		},
		{
			name: "fragments with HTML tags",
			fragments: map[string][]string{
				"Description": {"<mark>Highlighted</mark> text", "More <b>bold</b> text"},
			},
			expected:    "Highlighted text ... More bold text",
			description: "Should strip all HTML tags",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create mock DocumentMatch with fragments
			hit := &search.DocumentMatch{
				Fragments: tt.fragments,
				Fields:    map[string]interface{}{},
			}

			result := extractSnippet(hit)
			if result != tt.expected {
				t.Errorf("extractSnippet() = %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestExtractSnippet_WithoutFragments(t *testing.T) {
	tests := []struct {
		name        string
		fields      map[string]interface{}
		expected    string
		description string
	}{
		{
			name: "short description",
			fields: map[string]interface{}{
				"Description": "Short description text",
			},
			expected:    "Short description text",
			description: "Should return description as-is when <= 150 chars",
		},
		{
			name: "long description",
			fields: map[string]interface{}{
				"Description": "This is a very long description that exceeds the 150 character limit and should be truncated with ellipsis at the end to indicate that there is more content available in the full description text",
			},
			expected:    "This is a very long description that exceeds the 150 character limit and should be truncated with ellipsis at the end to indicate that there is more c...",
			description: "Should truncate to 150 chars and add ellipsis",
		},
		{
			name: "exactly 150 characters",
			fields: map[string]interface{}{
				"Description": "This description is exactly one hundred and fifty characters long and should not be truncated because it fits within the maximum allowed length test",
			},
			expected:    "This description is exactly one hundred and fifty characters long and should not be truncated because it fits within the maximum allowed length test",
			description: "Should return exactly 150 chars without truncation",
		},
		{
			name: "empty description",
			fields: map[string]interface{}{
				"Description": "",
			},
			expected:    "",
			description: "Should return empty string for empty description",
		},
		{
			name: "no description field",
			fields: map[string]interface{}{
				"ProjectPath": "some/path",
			},
			expected:    "",
			description: "Should return empty string when Description field missing",
		},
		{
			name: "description wrong type",
			fields: map[string]interface{}{
				"Description": 123, // Not a string
			},
			expected:    "",
			description: "Should return empty string for non-string Description",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create mock DocumentMatch without fragments
			hit := &search.DocumentMatch{
				Fragments: map[string][]string{},
				Fields:    tt.fields,
			}

			result := extractSnippet(hit)
			if result != tt.expected {
				t.Errorf("extractSnippet() = %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestExtractSnippet_EmptyFragments(t *testing.T) {
	// Edge case: Fragments map exists but Description key is empty array
	hit := &search.DocumentMatch{
		Fragments: map[string][]string{
			"Description": {},
		},
		Fields: map[string]interface{}{
			"Description": "Fallback description text",
		},
	}

	result := extractSnippet(hit)
	expected := "Fallback description text"
	if result != expected {
		t.Errorf("extractSnippet() with empty fragments = %q, want %q", result, expected)
	}
}

func TestNewDescriptionIndex_OpenError(t *testing.T) {
	tempDir := t.TempDir()
	indexPath := filepath.Join(tempDir, "corrupted.bleve")

	// Create a regular file instead of a bleve index directory
	// This will cause bleve.Open to fail
	if err := os.WriteFile(indexPath, []byte("not a bleve index"), 0644); err != nil {
		t.Fatalf("Failed to create corrupted file: %v", err)
	}

	// Attempting to open should fail
	_, err := NewDescriptionIndex(indexPath)
	if err == nil {
		t.Error("Expected error when opening corrupted index, got nil")
	}
	if !contains(err.Error(), "failed to open index") {
		t.Errorf("Expected 'failed to open index' error, got: %v", err)
	}
}

func TestDescriptionIndex_AddBatch_Error(t *testing.T) {
	tempDir := t.TempDir()
	indexPath := filepath.Join(tempDir, "test.bleve")

	di, err := NewDescriptionIndex(indexPath)
	if err != nil {
		t.Fatalf("Failed to create index: %v", err)
	}

	// Close the index to cause AddBatch to fail
	if err := di.Close(); err != nil {
		t.Fatalf("Failed to close index: %v", err)
	}

	// Now AddBatch should fail because index is closed
	docs := []DescriptionDocument{
		{ProjectPath: "test/path", ProjectName: "Test", Description: "Test"},
	}

	err = di.AddBatch(docs)
	if err == nil {
		t.Error("Expected AddBatch to fail with closed index")
	}
}

func TestDescriptionIndex_GetAllProjects_CountError(t *testing.T) {
	tempDir := t.TempDir()
	indexPath := filepath.Join(tempDir, "test.bleve")

	di, err := NewDescriptionIndex(indexPath)
	if err != nil {
		t.Fatalf("Failed to create index: %v", err)
	}

	// Close the index to cause Count() to fail
	if err := di.Close(); err != nil {
		t.Fatalf("Failed to close index: %v", err)
	}

	// GetAllProjects should fail when Count() fails
	_, err = di.GetAllProjects()
	if err == nil {
		t.Error("Expected GetAllProjects to fail when index is closed")
	}
	if !contains(err.Error(), "failed to get document count") {
		t.Errorf("Expected 'failed to get document count' error, got: %v", err)
	}
}

// Helper function for substring matching in error messages
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || findSubstring(s, substr))
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

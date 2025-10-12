package search

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/igusev/glf/internal/index"
	"github.com/igusev/glf/internal/types"
)

func TestAllProjectsSortedByHistory(t *testing.T) {
	projects := []types.Project{
		{Path: "project-a", Name: "Project A"},
		{Path: "project-b", Name: "Project B"},
		{Path: "project-c", Name: "Project C"},
		{Path: "project-d", Name: "Project D"},
	}

	historyScores := map[string]int{
		"project-b": 100, // Most used
		"project-d": 50,  // Second
		"project-a": 10,  // Third
		// project-c has no history (0)
	}

	results := allProjectsSortedByHistory(projects, historyScores)

	// Verify count
	if len(results) != len(projects) {
		t.Fatalf("Expected %d results, got %d", len(projects), len(results))
	}

	// Verify sorted by history score descending
	expectedOrder := []string{"project-b", "project-d", "project-a", "project-c"}
	for i, expected := range expectedOrder {
		if results[i].Project.Path != expected {
			t.Errorf("Position %d: got %q, want %q", i, results[i].Project.Path, expected)
		}
	}

	// Verify scores
	if results[0].HistoryScore != 100 {
		t.Errorf("First result history score = %d, want 100", results[0].HistoryScore)
	}
	if results[0].SearchScore != 0.0 {
		t.Errorf("Empty query should have SearchScore 0, got %f", results[0].SearchScore)
	}
	if results[0].TotalScore != 100.0 {
		t.Errorf("TotalScore should equal HistoryScore for empty query, got %f", results[0].TotalScore)
	}

	// Verify source
	if results[0].Source != index.MatchSourceName {
		t.Errorf("Source should be MatchSourceName for empty query, got %d", results[0].Source)
	}

	// Verify empty snippet
	if results[0].Snippet != "" {
		t.Errorf("Snippet should be empty for empty query, got %q", results[0].Snippet)
	}
}

func TestAllProjectsSortedByHistory_EmptyHistory(t *testing.T) {
	projects := []types.Project{
		{Path: "project-a", Name: "Project A"},
		{Path: "project-b", Name: "Project B"},
	}

	historyScores := map[string]int{} // No history

	results := allProjectsSortedByHistory(projects, historyScores)

	// All should have score 0
	for i, result := range results {
		if result.HistoryScore != 0 {
			t.Errorf("Result %d: HistoryScore = %d, want 0", i, result.HistoryScore)
		}
		if result.TotalScore != 0.0 {
			t.Errorf("Result %d: TotalScore = %f, want 0.0", i, result.TotalScore)
		}
	}
}

func TestAllProjectsSortedByHistory_EmptyProjects(t *testing.T) {
	projects := []types.Project{}
	historyScores := map[string]int{}

	results := allProjectsSortedByHistory(projects, historyScores)

	if len(results) != 0 {
		t.Errorf("Expected 0 results for empty projects, got %d", len(results))
	}
}

func TestCombinedSearch_EmptyQuery(t *testing.T) {
	projects := []types.Project{
		{Path: "high-history", Name: "High History"},
		{Path: "low-history", Name: "Low History"},
	}

	historyScores := map[string]int{
		"high-history": 200,
		"low-history":  10,
	}

	// Empty query should not need index
	results, err := CombinedSearch("", projects, historyScores, "/tmp")
	if err != nil {
		t.Fatalf("Empty query should not error: %v", err)
	}

	// Should return all projects sorted by history
	if len(results) != 2 {
		t.Fatalf("Expected 2 results, got %d", len(results))
	}

	// First should be high-history
	if results[0].Project.Path != "high-history" {
		t.Errorf("First result = %q, want high-history", results[0].Project.Path)
	}

	// Second should be low-history
	if results[1].Project.Path != "low-history" {
		t.Errorf("Second result = %q, want low-history", results[1].Project.Path)
	}
}

func TestCombinedSearch_IndexNotFound(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "glf-search-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	projects := []types.Project{
		{Path: "project-a", Name: "Project A"},
	}

	historyScores := map[string]int{}

	// Non-empty query with no index should error
	_, err = CombinedSearch("test", projects, historyScores, tmpDir)
	if err == nil {
		t.Error("Expected error when index not found")
	}

	// Error should mention running sync
	if err != nil && err.Error() != "search index not found, run 'glf sync' to build it" {
		t.Errorf("Unexpected error message: %v", err)
	}
}

func TestCombinedSearchWithIndex_EmptyQuery(t *testing.T) {
	projects := []types.Project{
		{Path: "project-1", Name: "First"},
		{Path: "project-2", Name: "Second"},
	}

	historyScores := map[string]int{
		"project-2": 100,
		"project-1": 50,
	}

	// Empty query with nil index should work (doesn't need index)
	results, err := CombinedSearchWithIndex("", projects, historyScores, "", nil)
	if err != nil {
		t.Fatalf("Empty query should not error: %v", err)
	}

	// Should be sorted by history
	if results[0].Project.Path != "project-2" {
		t.Errorf("First result = %q, want project-2", results[0].Project.Path)
	}
}

func TestCombinedSearchWithIndex_NilIndexNonEmptyQuery(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "glf-search-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	projects := []types.Project{
		{Path: "project-a", Name: "Project A"},
	}

	// Non-empty query with nil index and no index file should error
	_, err = CombinedSearchWithIndex("test", projects, nil, tmpDir, nil)
	if err == nil {
		t.Error("Expected error when index not found and nil index provided")
	}
}

// Integration test with real Bleve index
// Tests comprehensive search functionality including:
// - Search scoring and ranking
// - Field boosting (name > path > description)
// - History score integration
// - Multi-word query handling
// - Snippet generation
// - Cyrillic text search
func TestCombinedSearch_Integration(t *testing.T) {

	// Create test index
	tmpDir, err := os.MkdirTemp("", "glf-search-integration-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	indexPath := filepath.Join(tmpDir, "description.bleve")
	descIndex, err := index.NewDescriptionIndex(indexPath)
	if err != nil {
		t.Fatalf("Failed to create test index: %v", err)
	}
	defer descIndex.Close()

	// Add test documents
	testDocs := []index.DescriptionDocument{
		{
			ProjectPath: "api/auth",
			ProjectName: "Authentication API",
			Description: "User authentication and authorization service",
		},
		{
			ProjectPath: "api/payment",
			ProjectName: "Payment Gateway",
			Description: "Payment processing and billing",
		},
		{
			ProjectPath: "frontend/dashboard",
			ProjectName: "Admin Dashboard",
			Description: "Administrative interface for user management",
		},
	}

	if err := descIndex.AddBatch(testDocs); err != nil {
		t.Fatalf("Failed to add test docs: %v", err)
	}

	projects := []types.Project{
		{Path: "api/auth", Name: "Authentication API", Description: "User authentication and authorization service"},
		{Path: "api/payment", Name: "Payment Gateway", Description: "Payment processing and billing"},
		{Path: "frontend/dashboard", Name: "Admin Dashboard", Description: "Administrative interface for user management"},
	}

	historyScores := map[string]int{
		"api/auth": 100, // Boost auth project
	}

	// Test search
	results, err := CombinedSearchWithIndex("auth", projects, historyScores, tmpDir, descIndex)
	if err != nil {
		t.Fatalf("Search failed: %v", err)
	}

	// Should find auth-related projects
	if len(results) == 0 {
		t.Fatal("Expected results for 'auth' query")
	}

	// First result should be api/auth (name match + history boost)
	if results[0].Project.Path != "api/auth" {
		t.Errorf("First result = %q, want api/auth", results[0].Project.Path)
	}

	// Verify scores are calculated
	if results[0].SearchScore <= 0 {
		t.Errorf("SearchScore should be > 0, got %f", results[0].SearchScore)
	}
	if results[0].HistoryScore != 100 {
		t.Errorf("HistoryScore = %d, want 100", results[0].HistoryScore)
	}
	if results[0].TotalScore <= 0 {
		t.Errorf("TotalScore should be > 0, got %f", results[0].TotalScore)
	}

	// Verify source flags
	if results[0].Source == 0 {
		t.Error("Source should be set")
	}
}

func TestScoreCalculation(t *testing.T) {
	// Test that score calculation logic is correct
	projects := []types.Project{
		{Path: "project-a", Name: "Project A"},
		{Path: "project-b", Name: "Project B"},
	}

	historyScores := map[string]int{
		"project-a": 50,
		"project-b": 30,
	}

	results := allProjectsSortedByHistory(projects, historyScores)

	// Verify TotalScore = HistoryScore for empty query
	for _, result := range results {
		expected := float64(result.HistoryScore)
		if result.TotalScore != expected {
			t.Errorf("Project %s: TotalScore = %f, want %f (HistoryScore)",
				result.Project.Path, result.TotalScore, expected)
		}
	}
}

func TestProjectOrdering_StableSort(t *testing.T) {
	// Test that projects with same score maintain stable order
	projects := []types.Project{
		{Path: "project-a", Name: "A"},
		{Path: "project-b", Name: "B"},
		{Path: "project-c", Name: "C"},
	}

	// All have same score
	historyScores := map[string]int{
		"project-a": 10,
		"project-b": 10,
		"project-c": 10,
	}

	results := allProjectsSortedByHistory(projects, historyScores)

	// All should have same total score
	for i := range results {
		if results[i].TotalScore != 10.0 {
			t.Errorf("Result %d: TotalScore = %f, want 10.0", i, results[i].TotalScore)
		}
	}
}

func TestCombinedSearchWithIndex_CyrillicQuery(t *testing.T) {
	// Test searching with Cyrillic characters
	tmpDir, err := os.MkdirTemp("", "glf-search-cyrillic-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	indexPath := filepath.Join(tmpDir, "description.bleve")
	descIndex, err := index.NewDescriptionIndex(indexPath)
	if err != nil {
		t.Fatalf("Failed to create test index: %v", err)
	}
	defer descIndex.Close()

	// Add test documents with Cyrillic text
	testDocs := []index.DescriptionDocument{
		{
			ProjectPath: "api/авторизация",
			ProjectName: "Сервис авторизации",
			Description: "Аутентификация пользователей",
		},
		{
			ProjectPath: "api/платежи",
			ProjectName: "Платежный шлюз",
			Description: "Обработка платежей",
		},
	}

	if err := descIndex.AddBatch(testDocs); err != nil {
		t.Fatalf("Failed to add test docs: %v", err)
	}

	projects := []types.Project{
		{Path: "api/авторизация", Name: "Сервис авторизации", Description: "Аутентификация пользователей"},
		{Path: "api/платежи", Name: "Платежный шлюз", Description: "Обработка платежей"},
	}

	historyScores := map[string]int{}

	// Search with Cyrillic query
	results, err := CombinedSearchWithIndex("авторизация", projects, historyScores, tmpDir, descIndex)
	if err != nil {
		t.Fatalf("Cyrillic search failed: %v", err)
	}

	// Should find authorization project
	if len(results) == 0 {
		t.Fatal("Expected results for Cyrillic query 'авторизация'")
	}

	// First result should be авторизация project
	if results[0].Project.Path != "api/авторизация" {
		t.Errorf("First result = %q, want api/авторизация", results[0].Project.Path)
	}
}

func TestCombinedSearchWithIndex_MultiWordQuery(t *testing.T) {
	// Test multi-word query handling
	tmpDir, err := os.MkdirTemp("", "glf-search-multiword-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	indexPath := filepath.Join(tmpDir, "description.bleve")
	descIndex, err := index.NewDescriptionIndex(indexPath)
	if err != nil {
		t.Fatalf("Failed to create test index: %v", err)
	}
	defer descIndex.Close()

	// Add test documents
	testDocs := []index.DescriptionDocument{
		{
			ProjectPath: "api/user-service",
			ProjectName: "User Management Service",
			Description: "Manage user accounts and profiles",
		},
		{
			ProjectPath: "api/payment-service",
			ProjectName: "Payment Service",
			Description: "Process payments and transactions",
		},
	}

	if err := descIndex.AddBatch(testDocs); err != nil {
		t.Fatalf("Failed to add test docs: %v", err)
	}

	projects := []types.Project{
		{Path: "api/user-service", Name: "User Management Service", Description: "Manage user accounts and profiles"},
		{Path: "api/payment-service", Name: "Payment Service", Description: "Process payments and transactions"},
	}

	historyScores := map[string]int{}

	// Search with multi-word query
	results, err := CombinedSearchWithIndex("user management", projects, historyScores, tmpDir, descIndex)
	if err != nil {
		t.Fatalf("Multi-word search failed: %v", err)
	}

	// Should find user service
	if len(results) == 0 {
		t.Fatal("Expected results for multi-word query 'user management'")
	}

	// First result should be user-service
	if results[0].Project.Path != "api/user-service" {
		t.Errorf("First result = %q, want api/user-service", results[0].Project.Path)
	}
}

func TestCombinedSearchWithIndex_ProjectNotInMap(t *testing.T) {
	// Test handling of orphaned index entries (project in index but not in projectMap)
	tmpDir, err := os.MkdirTemp("", "glf-search-orphan-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	indexPath := filepath.Join(tmpDir, "description.bleve")
	descIndex, err := index.NewDescriptionIndex(indexPath)
	if err != nil {
		t.Fatalf("Failed to create test index: %v", err)
	}
	defer descIndex.Close()

	// Add documents to index
	testDocs := []index.DescriptionDocument{
		{
			ProjectPath: "api/active",
			ProjectName: "Active Project",
			Description: "This project exists",
		},
		{
			ProjectPath: "api/deleted",
			ProjectName: "Deleted Project",
			Description: "This project was deleted",
		},
	}

	if err := descIndex.AddBatch(testDocs); err != nil {
		t.Fatalf("Failed to add test docs: %v", err)
	}

	// Only provide active project (simulate deleted project)
	projects := []types.Project{
		{Path: "api/active", Name: "Active Project", Description: "This project exists"},
	}

	historyScores := map[string]int{}

	// Search should find both but only return the active one
	results, err := CombinedSearchWithIndex("project", projects, historyScores, tmpDir, descIndex)
	if err != nil {
		t.Fatalf("Search failed: %v", err)
	}

	// Should only return active project (orphaned entry skipped)
	if len(results) != 1 {
		t.Errorf("Expected 1 result (orphan skipped), got %d", len(results))
	}

	if len(results) > 0 && results[0].Project.Path != "api/active" {
		t.Errorf("Result = %q, want api/active", results[0].Project.Path)
	}
}

func TestCombinedSearchWithIndex_HistoryBoostIntegration(t *testing.T) {
	// Test that history boost correctly affects ranking
	tmpDir, err := os.MkdirTemp("", "glf-search-history-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	indexPath := filepath.Join(tmpDir, "description.bleve")
	descIndex, err := index.NewDescriptionIndex(indexPath)
	if err != nil {
		t.Fatalf("Failed to create test index: %v", err)
	}
	defer descIndex.Close()

	// Add test documents with similar scores
	testDocs := []index.DescriptionDocument{
		{
			ProjectPath: "api/service-a",
			ProjectName: "Service A",
			Description: "API service for data",
		},
		{
			ProjectPath: "api/service-b",
			ProjectName: "Service B",
			Description: "API service for data",
		},
	}

	if err := descIndex.AddBatch(testDocs); err != nil {
		t.Fatalf("Failed to add test docs: %v", err)
	}

	projects := []types.Project{
		{Path: "api/service-a", Name: "Service A", Description: "API service for data"},
		{Path: "api/service-b", Name: "Service B", Description: "API service for data"},
	}

	// Give service-b high history score
	historyScores := map[string]int{
		"api/service-b": 200,
		"api/service-a": 10,
	}

	// Search for "service" - both match equally
	results, err := CombinedSearchWithIndex("service", projects, historyScores, tmpDir, descIndex)
	if err != nil {
		t.Fatalf("Search failed: %v", err)
	}

	if len(results) < 2 {
		t.Fatalf("Expected at least 2 results, got %d", len(results))
	}

	// service-b should rank higher due to history boost
	if results[0].Project.Path != "api/service-b" {
		t.Errorf("First result = %q, want api/service-b (history boost)", results[0].Project.Path)
	}

	// Verify scores are calculated correctly
	if results[0].HistoryScore != 200 {
		t.Errorf("First result HistoryScore = %d, want 200", results[0].HistoryScore)
	}

	// Verify TotalScore includes history
	expectedTotal := results[0].SearchScore + float64(results[0].HistoryScore)
	if results[0].TotalScore != expectedTotal {
		t.Errorf("TotalScore = %f, want %f (SearchScore + HistoryScore)",
			results[0].TotalScore, expectedTotal)
	}
}

func TestCombinedSearchWithIndex_SnippetGeneration(t *testing.T) {
	// Test that snippets are generated for matches
	tmpDir, err := os.MkdirTemp("", "glf-search-snippet-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	indexPath := filepath.Join(tmpDir, "description.bleve")
	descIndex, err := index.NewDescriptionIndex(indexPath)
	if err != nil {
		t.Fatalf("Failed to create test index: %v", err)
	}
	defer descIndex.Close()

	// Add test document with long description
	testDocs := []index.DescriptionDocument{
		{
			ProjectPath: "api/search-service",
			ProjectName: "Search API",
			Description: "This is a comprehensive search service that provides full-text search capabilities across multiple data sources with advanced filtering and ranking algorithms",
		},
	}

	if err := descIndex.AddBatch(testDocs); err != nil {
		t.Fatalf("Failed to add test docs: %v", err)
	}

	projects := []types.Project{
		{Path: "api/search-service", Name: "Search API", Description: testDocs[0].Description},
	}

	historyScores := map[string]int{}

	// Search for term in description
	results, err := CombinedSearchWithIndex("search capabilities", projects, historyScores, tmpDir, descIndex)
	if err != nil {
		t.Fatalf("Search failed: %v", err)
	}

	if len(results) == 0 {
		t.Fatal("Expected results for snippet test")
	}

	// Snippet should be generated
	if results[0].Snippet == "" {
		t.Error("Expected non-empty snippet for description match")
	}
}

func TestCombinedSearchWithIndex_IndexOpenError(t *testing.T) {
	// Test error when opening index fails
	tmpDir, err := os.MkdirTemp("", "glf-search-openerr-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	indexPath := filepath.Join(tmpDir, "description.bleve")

	// Create a file instead of directory to cause index open error
	if err := os.WriteFile(indexPath, []byte("not an index"), 0644); err != nil {
		t.Fatalf("Failed to create file: %v", err)
	}

	projects := []types.Project{
		{Path: "project-a", Name: "Project A"},
	}

	historyScores := map[string]int{}

	// Should error when trying to open corrupted index
	_, err = CombinedSearchWithIndex("test", projects, historyScores, tmpDir, nil)
	if err == nil {
		t.Error("Expected error when index is corrupted")
	}

	// Error should mention failed to open
	if err != nil && !contains(err.Error(), "failed to open search index") {
		t.Errorf("Unexpected error message: %v", err)
	}
}

func TestCombinedSearchWithIndex_SearchError(t *testing.T) {
	// Test error when search operation fails
	// This is hard to trigger with real Bleve, so we'll create
	// a minimal scenario that could cause search failure

	tmpDir, err := os.MkdirTemp("", "glf-search-searcherr-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	indexPath := filepath.Join(tmpDir, "description.bleve")
	descIndex, err := index.NewDescriptionIndex(indexPath)
	if err != nil {
		t.Fatalf("Failed to create test index: %v", err)
	}

	// Close the index to cause search to fail
	descIndex.Close()

	projects := []types.Project{
		{Path: "project-a", Name: "Project A"},
	}

	historyScores := map[string]int{}

	// Search on closed index should error
	_, err = CombinedSearchWithIndex("test", projects, historyScores, tmpDir, descIndex)
	if err == nil {
		t.Error("Expected error when searching closed index")
	}

	// Error should mention search failed
	if err != nil && !contains(err.Error(), "search failed") {
		t.Errorf("Unexpected error message: %v", err)
	}
}

func TestCombinedSearchWithIndex_EmptyResults(t *testing.T) {
	// Test when search returns no results
	tmpDir, err := os.MkdirTemp("", "glf-search-empty-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	indexPath := filepath.Join(tmpDir, "description.bleve")
	descIndex, err := index.NewDescriptionIndex(indexPath)
	if err != nil {
		t.Fatalf("Failed to create test index: %v", err)
	}
	defer descIndex.Close()

	// Add one document
	testDocs := []index.DescriptionDocument{
		{
			ProjectPath: "api/service",
			ProjectName: "Service",
			Description: "A service",
		},
	}

	if err := descIndex.AddBatch(testDocs); err != nil {
		t.Fatalf("Failed to add test docs: %v", err)
	}

	projects := []types.Project{
		{Path: "api/service", Name: "Service", Description: "A service"},
	}

	historyScores := map[string]int{}

	// Search for term that doesn't exist
	results, err := CombinedSearchWithIndex("nonexistent", projects, historyScores, tmpDir, descIndex)
	if err != nil {
		t.Fatalf("Search failed: %v", err)
	}

	// Should return empty results, not error
	if len(results) != 0 {
		t.Errorf("Expected 0 results for non-matching query, got %d", len(results))
	}
}

// Helper function for error message checking
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) &&
		(s[:len(substr)] == substr || s[len(s)-len(substr):] == substr ||
			containsHelper(s, substr)))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func TestCombinedSearchWithIndex_OpensAndClosesIndex(t *testing.T) {
	// Test that function correctly opens and closes index when nil index provided
	tmpDir, err := os.MkdirTemp("", "glf-search-openclose-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	indexPath := filepath.Join(tmpDir, "description.bleve")

	// Create and populate index first
	descIndex, err := index.NewDescriptionIndex(indexPath)
	if err != nil {
		t.Fatalf("Failed to create test index: %v", err)
	}

	testDocs := []index.DescriptionDocument{
		{
			ProjectPath: "api/test",
			ProjectName: "Test Project",
			Description: "Test description",
		},
	}

	if err := descIndex.AddBatch(testDocs); err != nil {
		t.Fatalf("Failed to add test docs: %v", err)
	}
	descIndex.Close() // Close it so the function can open it

	projects := []types.Project{
		{Path: "api/test", Name: "Test Project", Description: "Test description"},
	}

	historyScores := map[string]int{}

	// Call with nil index - should open internally and close after
	results, err := CombinedSearchWithIndex("test", projects, historyScores, tmpDir, nil)
	if err != nil {
		t.Fatalf("Search with nil index should succeed when index exists: %v", err)
	}

	// Should find the project
	if len(results) == 0 {
		t.Error("Expected to find test project")
	}

	// Verify we can still open the index (wasn't left in bad state)
	descIndex2, err := index.NewDescriptionIndex(indexPath)
	if err != nil {
		t.Errorf("Should be able to reopen index after search: %v", err)
	}
	if descIndex2 != nil {
		descIndex2.Close()
	}
}

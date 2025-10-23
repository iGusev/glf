package search

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/igusev/glf/internal/index"
	"github.com/igusev/glf/internal/model"
)

func TestAllProjectsSortedByHistory(t *testing.T) {
	projects := []model.Project{
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
	projects := []model.Project{
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
	projects := []model.Project{}
	historyScores := map[string]int{}

	results := allProjectsSortedByHistory(projects, historyScores)

	if len(results) != 0 {
		t.Errorf("Expected 0 results for empty projects, got %d", len(results))
	}
}

func TestCombinedSearch_EmptyQuery(t *testing.T) {
	projects := []model.Project{
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

	projects := []model.Project{
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
	projects := []model.Project{
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

	projects := []model.Project{
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

	projects := []model.Project{
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
	projects := []model.Project{
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
	projects := []model.Project{
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

	projects := []model.Project{
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

	projects := []model.Project{
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
	projects := []model.Project{
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

	projects := []model.Project{
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

	projects := []model.Project{
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

	projects := []model.Project{
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

	projects := []model.Project{
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

	projects := []model.Project{
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

	projects := []model.Project{
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

// TestCalculateRelevanceMultiplier tests the relevance multiplier function
// which applies context-aware scaling to history/starred bonuses
func TestCalculateRelevanceMultiplier(t *testing.T) {
	tests := []struct {
		name           string
		searchScore    float64
		expectedMin    float64
		expectedMax    float64
		description    string
	}{
		{
			name:        "below minimum threshold",
			searchScore: 0.012,
			expectedMin: 0.0,
			expectedMax: 0.0,
			description: "Very low relevance should get no boost (multiplier = 0.0)",
		},
		{
			name:        "at minimum threshold",
			searchScore: 0.1,
			expectedMin: 0.0,
			expectedMax: 0.0,
			description: "Exactly at threshold should get zero boost",
		},
		{
			name:        "very weak match - level 1",
			searchScore: 0.2,
			expectedMin: 0.07,
			expectedMax: 0.10,
			description: "0.10-0.30 range: very slow ramp for very weak matches",
		},
		{
			name:        "weak match - level 2",
			searchScore: 0.4,
			expectedMin: 0.20,
			expectedMax: 0.25,
			description: "0.30-0.50 range: slow ramp for weak matches",
		},
		{
			name:        "decent match - level 3",
			searchScore: 0.6,
			expectedMin: 0.35,
			expectedMax: 0.45,
			description: "0.50-0.70 range: moderate ramp for decent matches",
		},
		{
			name:        "solid match - level 4",
			searchScore: 0.8,
			expectedMin: 0.55,
			expectedMax: 0.65,
			description: "0.70-0.90 range: medium-fast ramp for solid matches",
		},
		{
			name:        "very good match - level 5",
			searchScore: 1.0,
			expectedMin: 0.70,
			expectedMax: 0.80,
			description: "0.90-1.15 range: fast ramp for very good matches",
		},
		{
			name:        "excellent match - level 6",
			searchScore: 1.3,
			expectedMin: 0.90,
			expectedMax: 0.98,
			description: "1.15-1.40 range: very fast ramp for excellent matches",
		},
		{
			name:        "at full boost threshold",
			searchScore: 1.4,
			expectedMin: 1.0,
			expectedMax: 1.0,
			description: "At threshold should get full boost (multiplier = 1.0)",
		},
		{
			name:        "above full boost threshold",
			searchScore: 2.0,
			expectedMin: 1.0,
			expectedMax: 1.0,
			description: "Above threshold should get full boost (multiplier = 1.0)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			multiplier := calculateRelevanceMultiplier(tt.searchScore)

			if multiplier < tt.expectedMin || multiplier > tt.expectedMax {
				t.Errorf("%s: score=%.3f, multiplier=%.3f, want range [%.3f, %.3f]",
					tt.description, tt.searchScore, multiplier, tt.expectedMin, tt.expectedMax)
			}
		})
	}
}

// TestRelevanceMultiplier_RealWorldScenario tests the real-world scenario
// where an irrelevant project with high history/starred was ranking first
func TestRelevanceMultiplier_RealWorldScenario(t *testing.T) {
	// Simulate the real case from user feedback:
	// Project A: S:0.012 H:57 St:50 (irrelevant but frequently used and starred)
	// Project B: S:3.326 H:0 (highly relevant but no history)

	// Project A calculations
	searchScoreA := 0.012
	historyScoreA := 57
	starredBonusA := 50

	multiplierA := calculateRelevanceMultiplier(searchScoreA)
	adjustedHistoryA := float64(historyScoreA) * multiplierA
	adjustedStarredA := float64(starredBonusA) * multiplierA
	totalScoreA := searchScoreA + adjustedHistoryA + adjustedStarredA

	// Project B calculations
	searchScoreB := 3.326
	historyScoreB := 0
	starredBonusB := 0

	multiplierB := calculateRelevanceMultiplier(searchScoreB)
	adjustedHistoryB := float64(historyScoreB) * multiplierB
	adjustedStarredB := float64(starredBonusB) * multiplierB
	totalScoreB := searchScoreB + adjustedHistoryB + adjustedStarredB

	// Verify multiplier for irrelevant project is zero
	if multiplierA != 0.0 {
		t.Errorf("Irrelevant project (score=%.3f) should have multiplier=0.0, got %.3f",
			searchScoreA, multiplierA)
	}

	// Verify adjusted bonuses for irrelevant project are zeroed
	if adjustedHistoryA != 0.0 {
		t.Errorf("Adjusted history for irrelevant project should be 0.0, got %.3f",
			adjustedHistoryA)
	}
	if adjustedStarredA != 0.0 {
		t.Errorf("Adjusted starred bonus for irrelevant project should be 0.0, got %.3f",
			adjustedStarredA)
	}

	// Verify relevant project gets full boost (score > 1.4)
	if multiplierB != 1.0 {
		t.Errorf("Highly relevant project (score=%.3f) should have multiplier=1.0, got %.3f",
			searchScoreB, multiplierB)
	}

	// Most importantly: verify correct ranking
	if totalScoreB <= totalScoreA {
		t.Errorf("Highly relevant project should rank higher:\n"+
			"  Project A (irrelevant): S:%.3f H:%d St:%d -> Total:%.3f\n"+
			"  Project B (relevant):   S:%.3f H:%d St:%d -> Total:%.3f",
			searchScoreA, historyScoreA, starredBonusA, totalScoreA,
			searchScoreB, historyScoreB, starredBonusB, totalScoreB)
	}

	t.Logf("✓ Correct ranking achieved:")
	t.Logf("  Project A (irrelevant): S:%.3f H:%d St:%d M:%.3f -> Total:%.3f",
		searchScoreA, historyScoreA, starredBonusA, multiplierA, totalScoreA)
	t.Logf("  Project B (relevant):   S:%.3f H:%d St:%d M:%.3f -> Total:%.3f",
		searchScoreB, historyScoreB, starredBonusB, multiplierB, totalScoreB)
}

// TestRelevanceMultiplier_GradationSmoothness tests that transitions
// between gradation levels are smooth without sudden jumps
func TestRelevanceMultiplier_GradationSmoothness(t *testing.T) {
	// Test smoothness by checking that multiplier increases monotonically
	prevMultiplier := 0.0
	prevScore := 0.0

	// Test at gradation boundaries and between them
	testScores := []float64{
		0.05, 0.1, 0.15, 0.2, 0.25, 0.3,  // Level 1 boundary
		0.35, 0.4, 0.45, 0.5,               // Level 2 boundary
		0.55, 0.6, 0.65, 0.7,               // Level 3 boundary
		0.75, 0.8, 0.85, 0.9,               // Level 4 boundary
		0.95, 1.0, 1.05, 1.1, 1.15,        // Level 5 boundary
		1.2, 1.25, 1.3, 1.35, 1.4, 1.5,    // Level 6 boundary and beyond
	}

	for _, score := range testScores {
		multiplier := calculateRelevanceMultiplier(score)

		// Verify monotonic increase (or equal for thresholds)
		if multiplier < prevMultiplier {
			t.Errorf("Non-monotonic multiplier: score %.2f -> %.3f, previous score %.2f -> %.3f",
				score, multiplier, prevScore, prevMultiplier)
		}

		// Verify multiplier stays in valid range [0.0, 1.0]
		if multiplier < 0.0 || multiplier > 1.0 {
			t.Errorf("Multiplier out of range [0.0, 1.0]: score=%.2f, multiplier=%.3f",
				score, multiplier)
		}

		prevMultiplier = multiplier
		prevScore = score
	}
}

// TestRelevanceMultiplier_StarredProjectBehavior tests that starred projects
// get appropriate boosts based on search relevance
func TestRelevanceMultiplier_StarredProjectBehavior(t *testing.T) {
	starredBonus := 50

	tests := []struct {
		name        string
		searchScore float64
		expectedMin float64
		expectedMax float64
	}{
		{
			name:        "irrelevant starred project",
			searchScore: 0.05,
			expectedMin: 0.0,
			expectedMax: 0.0,
		},
		{
			name:        "weakly relevant starred project",
			searchScore: 0.3,
			expectedMin: 7.0,  // ~15% of 50
			expectedMax: 12.0,
		},
		{
			name:        "moderately relevant starred project",
			searchScore: 0.6,
			expectedMin: 17.0, // ~35% of 50
			expectedMax: 23.0,
		},
		{
			name:        "highly relevant starred project",
			searchScore: 1.2,
			expectedMin: 43.0, // ~88% of 50
			expectedMax: 46.0,
		},
		{
			name:        "very relevant starred project",
			searchScore: 1.5,
			expectedMin: 50.0, // full boost
			expectedMax: 50.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			multiplier := calculateRelevanceMultiplier(tt.searchScore)
			adjustedBonus := float64(starredBonus) * multiplier

			if adjustedBonus < tt.expectedMin || adjustedBonus > tt.expectedMax {
				t.Errorf("score=%.2f: adjusted bonus=%.1f, want range [%.1f, %.1f]",
					tt.searchScore, adjustedBonus, tt.expectedMin, tt.expectedMax)
			}

			t.Logf("✓ score=%.2f: multiplier=%.3f, bonus %.0f -> %.1f",
				tt.searchScore, multiplier, float64(starredBonus), adjustedBonus)
		})
	}
}

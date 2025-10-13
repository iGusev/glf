package history

import (
	"encoding/gob"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"
)

func TestHistory_RecordAndGetScore(t *testing.T) {
	h := New("/tmp/test_history.gob")

	// Initially, score should be 0
	if score := h.GetScore("project-a"); score != 0 {
		t.Errorf("Expected score 0 for new item, got %d", score)
	}

	// Record selection
	h.RecordSelection("project-a")

	// Score should now be ~10 (1 selection * 10) with minimal decay (truncated to int)
	score := h.GetScore("project-a")
	if score < 9 {
		t.Errorf("Expected score >= 9, got %d", score)
	}

	// Multiple selections increase score
	h.RecordSelection("project-a")
	h.RecordSelection("project-a")

	newScore := h.GetScore("project-a")
	if newScore <= score {
		t.Errorf("Expected score to increase after more selections")
	}
}

func TestHistory_SaveAndLoad(t *testing.T) {
	tempDir := t.TempDir()
	historyPath := filepath.Join(tempDir, "history.gob")

	// Create history and record some selections
	h1 := New(historyPath)
	h1.RecordSelection("project-a")
	h1.RecordSelection("project-b")
	h1.RecordSelection("project-a")

	// Save
	if err := h1.Save(); err != nil {
		t.Fatalf("Failed to save history: %v", err)
	}

	// Create new history instance and load
	h2 := New(historyPath)
	errCh := h2.LoadAsync()
	if err := <-errCh; err != nil {
		t.Fatalf("Failed to load history: %v", err)
	}

	// Verify data (scores reduced by 10x with decay: 2 selections ~19, 1 selection ~9)
	if score := h2.GetScore("project-a"); score < 19 {
		t.Errorf("Expected score >= 19 for project-a, got %d", score)
	}

	if score := h2.GetScore("project-b"); score < 9 {
		t.Errorf("Expected score >= 9 for project-b, got %d", score)
	}
}

func TestHistory_LoadAsync_NonExistent(t *testing.T) {
	h := New("/tmp/nonexistent_history.gob")

	errCh := h.LoadAsync()
	if err := <-errCh; err != nil {
		t.Errorf("Loading non-existent file should not return error, got: %v", err)
	}
}

func TestHistory_GetAllScores(t *testing.T) {
	h := New("/tmp/test_history.gob")

	h.RecordSelection("project-a")
	h.RecordSelection("project-b")
	h.RecordSelection("project-a")

	scores := h.GetAllScores()

	if len(scores) != 2 {
		t.Errorf("Expected 2 scores, got %d", len(scores))
	}

	if scores["project-a"] <= scores["project-b"] {
		t.Error("project-a should have higher score than project-b")
	}
}

func TestHistory_Stats(t *testing.T) {
	h := New("/tmp/test_history.gob")

	h.RecordSelection("project-a")
	h.RecordSelection("project-b")
	h.RecordSelection("project-a")

	total, unique := h.Stats()

	if total != 3 {
		t.Errorf("Expected 3 total selections, got %d", total)
	}

	if unique != 2 {
		t.Errorf("Expected 2 unique items, got %d", unique)
	}
}

func TestHistory_RecencyBoost(t *testing.T) {
	tempDir := t.TempDir()
	historyPath := filepath.Join(tempDir, "history.gob")

	h := New(historyPath)

	// Record old selection
	h.mu.Lock()
	h.selections["old-project"] = SelectionInfo{
		Count:    1,
		LastUsed: time.Now().Add(-30 * 24 * time.Hour), // 30 days ago
	}
	h.dirty = true
	h.mu.Unlock()

	// Record recent selection
	h.RecordSelection("new-project")

	oldScore := h.GetScore("old-project")
	newScore := h.GetScore("new-project")

	// New project should have higher score due to recency boost
	// Both have 1 selection, but new one gets recency bonus
	if newScore <= oldScore {
		t.Errorf("Recent item should have higher score. Old: %d, New: %d", oldScore, newScore)
	}
}

func TestHistory_Clear(t *testing.T) {
	h := New("/tmp/test_history.gob")

	h.RecordSelection("project-a")
	h.RecordSelection("project-b")

	h.Clear()

	if score := h.GetScore("project-a"); score != 0 {
		t.Errorf("Expected score 0 after clear, got %d", score)
	}

	total, unique := h.Stats()
	if total != 0 || unique != 0 {
		t.Errorf("Expected empty stats after clear, got total=%d, unique=%d", total, unique)
	}
}

func TestHistory_ConcurrentAccess(t *testing.T) {
	h := New("/tmp/test_history.gob")

	// Simulate concurrent access
	done := make(chan bool)

	for i := 0; i < 10; i++ {
		go func() {
			for j := 0; j < 100; j++ {
				h.RecordSelection("project-a")
				_ = h.GetScore("project-a")
				_ = h.GetAllScores()
			}
			done <- true
		}()
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}

	// Should have 1000 selections (10 goroutines * 100 selections)
	total, _ := h.Stats()
	if total != 1000 {
		t.Errorf("Expected 1000 selections, got %d", total)
	}
}

func TestHistory_DirtyFlag(t *testing.T) {
	tempDir := t.TempDir()
	historyPath := filepath.Join(tempDir, "history.gob")

	h := New(historyPath)

	// Initially not dirty
	if h.dirty {
		t.Error("New history should not be dirty")
	}

	// Recording selection makes it dirty
	h.RecordSelection("project-a")
	if !h.dirty {
		t.Error("History should be dirty after recording selection")
	}

	// Saving clears dirty flag
	if err := h.Save(); err != nil {
		t.Fatalf("Failed to save: %v", err)
	}

	if h.dirty {
		t.Error("History should not be dirty after save")
	}
}

func TestNormalizeQuery(t *testing.T) {
	tests := []struct {
		name     string
		query1   string
		query2   string
		samehash bool
	}{
		{
			name:     "same query lowercase",
			query1:   "backend",
			query2:   "Backend",
			samehash: true,
		},
		{
			name:     "whitespace trimming",
			query1:   "  backend  ",
			query2:   "backend",
			samehash: true,
		},
		{
			name:     "multiple spaces collapsed",
			query1:   "backend    api",
			query2:   "backend api",
			samehash: true,
		},
		{
			name:     "different queries",
			query1:   "backend",
			query2:   "frontend",
			samehash: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hash1 := normalizeQuery(tt.query1)
			hash2 := normalizeQuery(tt.query2)

			if tt.samehash && hash1 != hash2 {
				t.Errorf("Expected same hash for %q and %q", tt.query1, tt.query2)
			}
			if !tt.samehash && hash1 == hash2 {
				t.Errorf("Expected different hash for %q and %q", tt.query1, tt.query2)
			}
		})
	}
}

func TestHistory_RecordSelectionWithQuery(t *testing.T) {
	h := New("/tmp/test_history_query.gob")

	// Record with query
	h.RecordSelectionWithQuery("backend", "project-api")
	h.RecordSelectionWithQuery("backend", "project-api")
	h.RecordSelectionWithQuery("frontend", "project-web")

	// Verify global history
	if score := h.GetScore("project-api"); score < 19 {
		t.Errorf("Expected global score >= 19 for project-api, got %d", score)
	}

	// Verify query-specific history exists
	h.mu.RLock()
	queryHash := normalizeQuery("backend")
	if h.querySelections[queryHash] == nil {
		t.Error("Expected query-specific history for 'backend'")
	}
	if info, exists := h.querySelections[queryHash]["project-api"]; !exists {
		t.Error("Expected query-specific entry for project-api")
	} else if info.Count != 2 {
		t.Errorf("Expected query-specific count 2, got %d", info.Count)
	}
	h.mu.RUnlock()
}

func TestHistory_GetScoreForQuery(t *testing.T) {
	h := New("/tmp/test_history_query.gob")

	// Record global selection
	h.RecordSelection("project-a")

	// Record query-specific selection
	h.RecordSelectionWithQuery("backend", "project-a")
	h.RecordSelectionWithQuery("backend", "project-a")

	// Global score only
	globalScore := h.GetScore("project-a")

	// Score with query boost
	queryScore := h.GetScoreForQuery("backend", "project-a")

	// Query score should be significantly higher (3x boost)
	if queryScore <= globalScore {
		t.Errorf("Query score (%d) should be higher than global score (%d)", queryScore, globalScore)
	}

	// Different query should give lower score
	otherQueryScore := h.GetScoreForQuery("frontend", "project-a")
	if otherQueryScore >= queryScore {
		t.Errorf("Other query score should be lower than matching query score")
	}
}

func TestHistory_GetAllScoresForQuery(t *testing.T) {
	h := New("/tmp/test_history_query.gob")

	// Record some selections
	h.RecordSelection("project-a")
	h.RecordSelectionWithQuery("backend", "project-b")
	h.RecordSelectionWithQuery("backend", "project-b")
	h.RecordSelection("project-c")

	// Get all scores for "backend" query
	scores := h.GetAllScoresForQuery("backend")

	// Should have all 3 projects
	if len(scores) != 3 {
		t.Errorf("Expected 3 projects, got %d", len(scores))
	}

	// project-b should have highest score (query-specific boost)
	if scores["project-b"] <= scores["project-a"] {
		t.Error("project-b should have higher score due to query boost")
	}
	if scores["project-b"] <= scores["project-c"] {
		t.Error("project-b should have higher score due to query boost")
	}
}

func TestHistory_QueryBoostWithEmptyQuery(t *testing.T) {
	h := New("/tmp/test_history_query.gob")

	h.RecordSelection("project-a")

	// Empty query should work without errors
	score := h.GetScoreForQuery("", "project-a")
	if score < 9 {
		t.Errorf("Expected score >= 9 even with empty query, got %d", score)
	}

	scores := h.GetAllScoresForQuery("")
	if len(scores) != 1 {
		t.Errorf("Expected 1 project with empty query, got %d", len(scores))
	}
}

func TestHistory_SaveAndLoadWithQuery(t *testing.T) {
	tempDir := t.TempDir()
	historyPath := filepath.Join(tempDir, "history.gob")

	// Create and save history with query selections
	h1 := New(historyPath)
	h1.RecordSelectionWithQuery("backend", "project-api")
	h1.RecordSelectionWithQuery("backend", "project-api")
	h1.RecordSelection("project-web")

	if err := h1.Save(); err != nil {
		t.Fatalf("Failed to save: %v", err)
	}

	// Load and verify
	h2 := New(historyPath)
	errCh := h2.LoadAsync()
	if err := <-errCh; err != nil {
		t.Fatalf("Failed to load: %v", err)
	}

	// Verify query-specific data persisted
	queryScore := h2.GetScoreForQuery("backend", "project-api")
	if queryScore == 0 {
		t.Error("Query-specific data not loaded correctly")
	}
}

func TestHistory_CleanupOldEntries_WithQueries(t *testing.T) {
	h := New("/tmp/test_history_cleanup.gob")

	// Add old query-specific entry
	h.mu.Lock()
	queryHash := normalizeQuery("backend")
	h.querySelections[queryHash] = make(map[string]SelectionInfo)
	h.querySelections[queryHash]["old-project"] = SelectionInfo{
		Count:    1,
		LastUsed: time.Now().Add(-200 * 24 * time.Hour), // 200 days ago
	}
	h.mu.Unlock()

	// Add recent entry
	h.RecordSelectionWithQuery("backend", "new-project")

	// Cleanup
	removed := h.CleanupOldEntries()

	if removed == 0 {
		t.Error("Expected cleanup to remove old entries")
	}

	// Old project should be gone
	score := h.GetScoreForQuery("backend", "old-project")
	if score != 0 {
		t.Errorf("Old project should have score 0 after cleanup, got %d", score)
	}

	// New project should remain
	score = h.GetScoreForQuery("backend", "new-project")
	if score == 0 {
		t.Error("New project should still have score after cleanup")
	}
}

// Cleanup after tests
func TestMain(m *testing.M) {
	code := m.Run()

	// Cleanup test files
	os.Remove("/tmp/test_history.gob")
	os.Remove("/tmp/test_history.gob.tmp")
	os.Remove("/tmp/nonexistent_history.gob")
	os.Remove("/tmp/test_history_query.gob")
	os.Remove("/tmp/test_history_cleanup.gob")

	os.Exit(code)
}

func TestCalculateDecayMultiplier(t *testing.T) {
	tests := []struct {
		name     string
		days     float64
		wantZero bool
		wantHalf bool
	}{
		{
			name:     "0 days - no decay",
			days:     0,
			wantZero: false,
			wantHalf: false,
		},
		{
			name:     "30 days - half life",
			days:     30,
			wantZero: false,
			wantHalf: true,
		},
		{
			name:     "100 days - at boundary",
			days:     100,
			wantZero: false,
			wantHalf: false,
		},
		{
			name:     "101 days - beyond max age",
			days:     101,
			wantZero: true,
			wantHalf: false,
		},
		{
			name:     "200 days - way beyond",
			days:     200,
			wantZero: true,
			wantHalf: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := calculateDecayMultiplier(tt.days)

			if tt.wantZero && result != 0 {
				t.Errorf("Expected 0 for %f days, got %f", tt.days, result)
			}

			if !tt.wantZero && result == 0 {
				t.Errorf("Expected non-zero for %f days, got 0", tt.days)
			}

			if tt.wantHalf {
				// At half-life, should be ~0.5
				if result < 0.49 || result > 0.51 {
					t.Errorf("Expected ~0.5 for half-life, got %f", result)
				}
			}

			if tt.days == 0 {
				// At 0 days, should be ~1.0
				if result < 0.99 || result > 1.01 {
					t.Errorf("Expected ~1.0 for 0 days, got %f", result)
				}
			}
		})
	}
}

func TestHistory_CleanupOldEntries_GlobalSelections(t *testing.T) {
	h := New("/tmp/test_cleanup_global.gob")

	// Add old global entry
	h.mu.Lock()
	h.selections["old-project"] = SelectionInfo{
		Count:    5,
		LastUsed: time.Now().Add(-150 * 24 * time.Hour), // 150 days ago
	}
	h.mu.Unlock()

	// Add recent entry
	h.RecordSelection("new-project")

	// Cleanup should remove old entry
	removed := h.CleanupOldEntries()

	if removed != 1 {
		t.Errorf("Expected 1 entry removed, got %d", removed)
	}

	// Old project should have 0 score
	if score := h.GetScore("old-project"); score != 0 {
		t.Errorf("Expected 0 score for old project, got %d", score)
	}

	// New project should still exist
	if score := h.GetScore("new-project"); score == 0 {
		t.Error("New project should still have score")
	}

	// Verify dirty flag set
	if !h.dirty {
		t.Error("History should be dirty after cleanup")
	}
}

func TestHistory_CleanupOldEntries_EmptyQueryHash(t *testing.T) {
	h := New("/tmp/test_cleanup_empty.gob")

	// Add old query-specific entry that will be cleaned
	h.mu.Lock()
	queryHash := normalizeQuery("backend")
	h.querySelections[queryHash] = make(map[string]SelectionInfo)
	h.querySelections[queryHash]["old-project"] = SelectionInfo{
		Count:    1,
		LastUsed: time.Now().Add(-150 * 24 * time.Hour),
	}
	h.mu.Unlock()

	// Cleanup should remove entry and empty query hash
	removed := h.CleanupOldEntries()

	if removed != 1 {
		t.Errorf("Expected 1 entry removed, got %d", removed)
	}

	// Query hash should be completely removed
	h.mu.RLock()
	_, exists := h.querySelections[queryHash]
	h.mu.RUnlock()

	if exists {
		t.Error("Empty query hash should be removed")
	}
}

func TestHistory_CleanupOldEntries_NoOldEntries(t *testing.T) {
	h := New("/tmp/test_cleanup_none.gob")

	// Add only recent entries
	h.RecordSelection("project-a")
	h.RecordSelection("project-b")

	// Mark as not dirty to verify it stays clean if nothing removed
	h.mu.Lock()
	h.dirty = false
	h.mu.Unlock()

	// Cleanup should find nothing to remove
	removed := h.CleanupOldEntries()

	if removed != 0 {
		t.Errorf("Expected 0 entries removed, got %d", removed)
	}

	// Should not set dirty flag if nothing removed
	if h.dirty {
		t.Error("History should not be dirty if nothing was cleaned")
	}
}

func TestHistory_Save_NotDirty(t *testing.T) {
	tempDir := t.TempDir()
	historyPath := filepath.Join(tempDir, "history.gob")

	h := New(historyPath)
	h.RecordSelection("project-a")

	// Save once
	if err := h.Save(); err != nil {
		t.Fatalf("First save failed: %v", err)
	}

	// Modify file timestamp
	time.Sleep(10 * time.Millisecond)
	stat1, _ := os.Stat(historyPath)

	// Save again (not dirty, should skip)
	if err := h.Save(); err != nil {
		t.Fatalf("Second save failed: %v", err)
	}

	stat2, _ := os.Stat(historyPath)

	// File should not be modified (timestamps should match)
	if !stat1.ModTime().Equal(stat2.ModTime()) {
		t.Error("File should not be modified when saving non-dirty history")
	}
}

func TestHistory_Save_DirectoryCreation(t *testing.T) {
	tempDir := t.TempDir()
	// Use nested path that doesn't exist
	historyPath := filepath.Join(tempDir, "nested", "deep", "history.gob")

	h := New(historyPath)
	h.RecordSelection("project-a")

	// Should create nested directories
	if err := h.Save(); err != nil {
		t.Fatalf("Save with directory creation failed: %v", err)
	}

	// Verify file exists
	if _, err := os.Stat(historyPath); os.IsNotExist(err) {
		t.Error("History file should exist after save")
	}
}

func TestHistory_LoadAsync_OldFormat(t *testing.T) {
	tempDir := t.TempDir()
	historyPath := filepath.Join(tempDir, "old_format.gob")

	// Create old format file (just map[string]SelectionInfo)
	oldData := map[string]SelectionInfo{
		"project-a": {Count: 5, LastUsed: time.Now()},
		"project-b": {Count: 3, LastUsed: time.Now()},
	}

	file, err := os.Create(historyPath)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	encoder := gob.NewEncoder(file)
	if err := encoder.Encode(oldData); err != nil {
		t.Fatalf("Failed to encode old format: %v", err)
	}
	file.Close()

	// Load with new History instance
	h := New(historyPath)
	errCh := h.LoadAsync()
	if err := <-errCh; err != nil {
		t.Fatalf("Failed to load old format: %v", err)
	}

	// Should have migrated data
	if score := h.GetScore("project-a"); score == 0 {
		t.Error("Old format data should be loaded")
	}

	// Query selections should be empty (new field)
	h.mu.RLock()
	qsLen := len(h.querySelections)
	h.mu.RUnlock()

	if qsLen != 0 {
		t.Errorf("Query selections should be empty after migration, got %d", qsLen)
	}
}

func TestHistory_LoadAsync_CorruptedFile(t *testing.T) {
	tempDir := t.TempDir()
	historyPath := filepath.Join(tempDir, "corrupted.gob")

	// Create corrupted file
	if err := os.WriteFile(historyPath, []byte("not a valid gob file"), 0600); err != nil {
		t.Fatalf("Failed to create corrupted file: %v", err)
	}

	h := New(historyPath)
	errCh := h.LoadAsync()
	err := <-errCh

	// Should return error for corrupted file
	if err == nil {
		t.Error("Expected error loading corrupted file")
	}
}

func TestHistory_LoadAsync_PermissionDenied(t *testing.T) {
	// Skip on systems where we can't test permissions
	if os.Getuid() == 0 {
		t.Skip("Skipping permission test when running as root")
	}

	tempDir := t.TempDir()
	historyPath := filepath.Join(tempDir, "noperm.gob")

	// Create file with no read permissions
	if err := os.WriteFile(historyPath, []byte("test"), 0000); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}
	defer os.Chmod(historyPath, 0600) // Cleanup

	h := New(historyPath)
	errCh := h.LoadAsync()
	err := <-errCh

	// Should return error for permission denied
	if err == nil {
		t.Error("Expected error loading file without permissions")
	}
}

func TestHistory_Save_EncodingError(t *testing.T) {
	// This is hard to trigger naturally since gob.Encode rarely fails
	// Testing directory creation and other error paths is more practical
	t.Skip("Gob encoding errors are difficult to trigger in tests")
}

func TestHistory_GetScore_VeryOldEntry(t *testing.T) {
	h := New("/tmp/test_old_entry.gob")

	// Add entry beyond max age
	h.mu.Lock()
	h.selections["ancient-project"] = SelectionInfo{
		Count:    100,                                   // High count
		LastUsed: time.Now().Add(-200 * 24 * time.Hour), // 200 days ago
	}
	h.mu.Unlock()

	// Score should be 0 due to age cutoff
	score := h.GetScore("ancient-project")
	if score != 0 {
		t.Errorf("Expected 0 score for very old entry, got %d", score)
	}
}

func TestHistory_GetAllScores_SkipsOldEntries(t *testing.T) {
	h := New("/tmp/test_getall_old.gob")

	// Add mix of old and new entries
	h.mu.Lock()
	h.selections["old-project"] = SelectionInfo{
		Count:    10,
		LastUsed: time.Now().Add(-200 * 24 * time.Hour),
	}
	h.mu.Unlock()

	h.RecordSelection("new-project")

	// GetAllScores should skip old entries
	scores := h.GetAllScores()

	if len(scores) != 1 {
		t.Errorf("Expected 1 score (old entry skipped), got %d", len(scores))
	}

	if _, exists := scores["old-project"]; exists {
		t.Error("Old project should not be in scores")
	}

	if _, exists := scores["new-project"]; !exists {
		t.Error("New project should be in scores")
	}
}

func TestHistory_Save_MkdirError(t *testing.T) {
	// Skip on systems where we can't test permissions
	if runtime.GOOS == "windows" {
		t.Skip("Skipping test on Windows: chmod doesn't work the same way")
	}
	if os.Getuid() == 0 {
		t.Skip("Skipping permission test when running as root")
	}

	tempDir := t.TempDir()

	// Create read-only directory
	readOnlyDir := filepath.Join(tempDir, "readonly")
	if err := os.Mkdir(readOnlyDir, 0555); err != nil {
		t.Fatalf("Failed to create readonly directory: %v", err)
	}
	defer os.Chmod(readOnlyDir, 0755) // Cleanup

	// Try to create history in subdirectory of read-only dir
	historyPath := filepath.Join(readOnlyDir, "subdir", "history.gob")

	h := New(historyPath)
	h.RecordSelection("project-a")

	// Should fail to create directory
	err := h.Save()
	if err == nil {
		t.Error("Expected MkdirAll error, got nil")
	}
	if err != nil && !contains(err.Error(), "failed to create history directory") {
		t.Errorf("Expected 'failed to create history directory' in error, got: %v", err)
	}
}

func TestHistory_Save_CreateFileError(t *testing.T) {
	// Skip on systems where we can't test permissions
	if runtime.GOOS == "windows" {
		t.Skip("Skipping test on Windows: chmod doesn't work the same way")
	}
	if os.Getuid() == 0 {
		t.Skip("Skipping permission test when running as root")
	}

	tempDir := t.TempDir()

	// Create directory structure
	historyDir := filepath.Join(tempDir, "history")
	if err := os.Mkdir(historyDir, 0755); err != nil {
		t.Fatalf("Failed to create history directory: %v", err)
	}

	// Make directory read-only to prevent file creation
	if err := os.Chmod(historyDir, 0555); err != nil {
		t.Fatalf("Failed to chmod directory: %v", err)
	}
	defer os.Chmod(historyDir, 0755) // Cleanup

	historyPath := filepath.Join(historyDir, "history.gob")

	h := New(historyPath)
	h.RecordSelection("project-a")

	// Should fail to create temp file
	err := h.Save()
	if err == nil {
		t.Error("Expected Create error, got nil")
	}
	if err != nil && !contains(err.Error(), "failed to create temp file") {
		t.Errorf("Expected 'failed to create temp file' in error, got: %v", err)
	}
}

func TestHistory_Save_RenameError(t *testing.T) {
	tempDir := t.TempDir()
	historyPath := filepath.Join(tempDir, "history.gob")

	// Create a directory where the target file should be (prevents rename)
	if err := os.Mkdir(historyPath, 0755); err != nil {
		t.Fatalf("Failed to create directory: %v", err)
	}

	h := New(historyPath)
	h.RecordSelection("project-a")

	// Should fail to rename (can't replace directory with file)
	err := h.Save()
	if err == nil {
		t.Error("Expected Rename error, got nil")
	}
	if err != nil && !contains(err.Error(), "failed to rename temp file") {
		t.Errorf("Expected 'failed to rename temp file' in error, got: %v", err)
	}

	// Verify temp file was cleaned up
	tempPath := historyPath + ".tmp"
	if _, err := os.Stat(tempPath); !os.IsNotExist(err) {
		t.Error("Temp file should be cleaned up after rename error")
	}
}

func TestHistory_LoadAsync_WithCleanupBackgroundSave(t *testing.T) {
	// Test that LoadAsync triggers cleanup and background save for old entries
	tempDir := t.TempDir()
	historyPath := filepath.Join(tempDir, "history_cleanup.gob")

	// Create history with mix of old and new entries
	h1 := New(historyPath)

	// Add old entries (beyond maxAgeDays)
	h1.mu.Lock()
	h1.selections["very-old-project"] = SelectionInfo{
		Count:    5,
		LastUsed: time.Now().Add(-150 * 24 * time.Hour), // 150 days old
	}
	h1.selections["old-project"] = SelectionInfo{
		Count:    3,
		LastUsed: time.Now().Add(-120 * 24 * time.Hour), // 120 days old
	}
	h1.mu.Unlock()

	// Add recent entry
	h1.RecordSelection("new-project")

	// Save to disk
	if err := h1.Save(); err != nil {
		t.Fatalf("Failed to save initial history: %v", err)
	}

	// Load with new History instance - should trigger cleanup
	h2 := New(historyPath)
	errCh := h2.LoadAsync()
	err := <-errCh
	if err != nil {
		t.Fatalf("Failed to load history: %v", err)
	}

	// Give background save goroutine time to complete
	time.Sleep(100 * time.Millisecond)

	// Verify old entries were cleaned
	if score := h2.GetScore("very-old-project"); score != 0 {
		t.Errorf("Very old project should have score 0, got %d", score)
	}
	if score := h2.GetScore("old-project"); score != 0 {
		t.Errorf("Old project should have score 0, got %d", score)
	}

	// New project should remain
	if score := h2.GetScore("new-project"); score == 0 {
		t.Error("New project should have non-zero score")
	}

	// Verify cleanup was saved to disk by loading again
	h3 := New(historyPath)
	errCh = h3.LoadAsync()
	if err := <-errCh; err != nil {
		t.Fatalf("Failed to load after cleanup: %v", err)
	}

	// Old entries should still be gone
	_, unique := h3.Stats()
	if unique != 1 {
		t.Errorf("Expected 1 unique item after cleanup, got %d", unique)
	}
}

func TestHistory_LoadAsync_QuerySelectionsNil(t *testing.T) {
	// Test loading when QuerySelections is nil in saved data
	tempDir := t.TempDir()
	historyPath := filepath.Join(tempDir, "history_nil_qs.gob")

	// Create history data with nil QuerySelections
	h1 := New(historyPath)
	h1.RecordSelection("project-a")

	// Manually set QuerySelections to nil before saving
	h1.mu.Lock()
	h1.querySelections = nil
	h1.dirty = true
	h1.mu.Unlock()

	// Save
	if err := h1.Save(); err != nil {
		t.Fatalf("Failed to save: %v", err)
	}

	// Load should initialize empty map
	h2 := New(historyPath)
	errCh := h2.LoadAsync()
	if err := <-errCh; err != nil {
		t.Fatalf("Failed to load: %v", err)
	}

	// QuerySelections should be initialized to empty map, not nil
	h2.mu.RLock()
	if h2.querySelections == nil {
		t.Error("QuerySelections should be initialized, not nil")
	}
	h2.mu.RUnlock()

	// Should work without panicking
	h2.RecordSelectionWithQuery("backend", "project-b")
	if score := h2.GetScoreForQuery("backend", "project-b"); score == 0 {
		t.Error("Should be able to record and get query-specific scores")
	}
}

// Helper function for string matching
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

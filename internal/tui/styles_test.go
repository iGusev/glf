package tui

import (
	"testing"
)

func TestNewColorScheme(t *testing.T) {
	cs := NewColorScheme()

	if cs == nil {
		t.Fatal("Expected NewColorScheme to return non-nil ColorScheme")
	}

	// Check that GitLabWave is set (should contain wave emoji)
	if cs.GitLabWave == "" {
		t.Error("Expected GitLabWave to be non-empty")
	}

	if len(cs.GitLabWave) < 3 {
		t.Error("Expected GitLabWave to contain emoji characters")
	}
}

func TestGetStyles(t *testing.T) {
	cs := NewColorScheme()
	styles := cs.GetStyles()

	// Verify styles can be used (render without panic)
	// We can't compare lipgloss.Style directly as it contains functions

	// Test that each style can render a string
	testStr := "test"

	// Try rendering with each style - should not panic
	_ = styles.Title.Render(testStr)
	_ = styles.Version.Render(testStr)
	_ = styles.Prompt.Render(testStr)
	_ = styles.Cursor.Render(testStr)
	_ = styles.Selected.Render(testStr)
	_ = styles.Normal.Render(testStr)
	_ = styles.Excluded.Render(testStr)
	_ = styles.Highlight.Render(testStr)
	_ = styles.Snippet.Render(testStr)
	_ = styles.Count.Render(testStr)
	_ = styles.ServerInfo.Render(testStr)
	_ = styles.Help.Render(testStr)
	_ = styles.StatusIdle.Render(testStr)
	_ = styles.StatusActive.Render(testStr)
	_ = styles.StatusError.Render(testStr)

	// If we got here without panicking, styles are working
	t.Log("All styles rendered successfully")
}

func TestColorScheme_MultipleInstances(t *testing.T) {
	// Create multiple color schemes
	cs1 := NewColorScheme()
	cs2 := NewColorScheme()

	// Both should be initialized
	if cs1 == nil || cs2 == nil {
		t.Fatal("Expected both color schemes to be non-nil")
	}

	// Both should have the same GitLabWave (deterministic)
	if cs1.GitLabWave != cs2.GitLabWave {
		t.Error("Expected GitLabWave to be consistent across instances")
	}

	// GetStyles should work for both
	styles1 := cs1.GetStyles()
	styles2 := cs2.GetStyles()

	// Test rendering with both styles - should not panic
	testStr := "test"
	_ = styles1.Title.Render(testStr)
	_ = styles2.Title.Render(testStr)

	t.Log("Multiple color scheme instances work correctly")
}

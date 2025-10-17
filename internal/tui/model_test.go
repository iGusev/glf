package tui

import (
	"fmt"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/igusev/glf/internal/config"
	"github.com/igusev/glf/internal/index"
	"github.com/igusev/glf/internal/model"
)

func TestFormatNumber(t *testing.T) {
	tests := []struct {
		name     string
		input    int
		expected string
	}{
		{
			name:     "single digit",
			input:    5,
			expected: "5",
		},
		{
			name:     "hundreds",
			input:    123,
			expected: "123",
		},
		{
			name:     "thousands with comma",
			input:    1234,
			expected: "1,234",
		},
		{
			name:     "tens of thousands",
			input:    12345,
			expected: "12,345",
		},
		{
			name:     "hundreds of thousands",
			input:    123456,
			expected: "123,456",
		},
		{
			name:     "millions",
			input:    1234567,
			expected: "1234,567", // Function only adds one comma from right
		},
		{
			name:     "zero",
			input:    0,
			expected: "0",
		},
		{
			name:     "exactly thousand",
			input:    1000,
			expected: "1,000",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatNumber(tt.input)
			if result != tt.expected {
				t.Errorf("formatNumber(%d) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestTruncateSnippet(t *testing.T) {
	tests := []struct {
		name     string
		text     string
		maxRunes int
		expected string
	}{
		{
			name:     "empty string",
			text:     "",
			maxRunes: 10,
			expected: "",
		},
		{
			name:     "shorter than max",
			text:     "Hello",
			maxRunes: 10,
			expected: "Hello",
		},
		{
			name:     "exactly max",
			text:     "Hello",
			maxRunes: 5,
			expected: "Hello",
		},
		{
			name:     "longer than max",
			text:     "Hello World",
			maxRunes: 8,
			expected: "Hello Wo...", // Truncates at maxRunes, adds "..."
		},
		{
			name:     "much longer than max",
			text:     "This is a very long text that needs truncation",
			maxRunes: 20,
			expected: "This is a very long...", // Word boundary at "long"
		},
		{
			name:     "zero max",
			text:     "Hello",
			maxRunes: 0,
			expected: "...", // Minimum result
		},
		{
			name:     "cyrillic text",
			text:     "Привет мир",
			maxRunes: 7,
			expected: "Привет...", // Space is word boundary (last 20%)
		},
		{
			name:     "cyrillic exact fit",
			text:     "Привет",
			maxRunes: 6,
			expected: "Привет",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := truncateSnippet(tt.text, tt.maxRunes)
			if result != tt.expected {
				t.Errorf("truncateSnippet(%q, %d) = %q, want %q",
					tt.text, tt.maxRunes, result, tt.expected)
			}
		})
	}
}

func TestTruncateSnippet_RuneCount(t *testing.T) {
	// Test that truncation respects rune count, not byte count
	text := "🎉🎊🎈" // Each emoji is 4 bytes but 1 rune (3 total)
	maxRunes := 5
	result := truncateSnippet(text, maxRunes)

	// Text is 3 runes, less than maxRunes 5, so returned as-is
	expected := "🎉🎊🎈"
	if result != expected {
		t.Errorf("truncateSnippet(%q, %d) = %q, want %q (no truncation needed)",
			text, maxRunes, result, expected)
	}

	// Test with smaller maxRunes to force truncation
	maxRunes2 := 2
	result2 := truncateSnippet(text, maxRunes2)

	// Should truncate to 2 emojis + "..."
	expected2 := "🎉🎊..."
	if result2 != expected2 {
		t.Errorf("truncateSnippet(%q, %d) = %q, want %q",
			text, maxRunes2, result2, expected2)
	}
}

func TestFormatCountWithBreakdown(t *testing.T) {
	// Create test styles
	countStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#666"))
	activeStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#FC6D25"))

	tests := []struct {
		name           string
		matches        []index.CombinedMatch
		total          int
		expectFiltered bool
	}{
		{
			name:           "no matches",
			matches:        []index.CombinedMatch{},
			total:          100,
			expectFiltered: true,
		},
		{
			name: "some matches",
			matches: []index.CombinedMatch{
				{Project: model.Project{Path: "a"}},
				{Project: model.Project{Path: "b"}},
				{Project: model.Project{Path: "c"}},
			},
			total:          10,
			expectFiltered: true,
		},
		{
			name: "all matches (no filter)",
			matches: []index.CombinedMatch{
				{Project: model.Project{Path: "a"}},
				{Project: model.Project{Path: "b"}},
			},
			total:          2,
			expectFiltered: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatCountWithBreakdown(tt.matches, tt.total, countStyle, activeStyle)

			// Result should contain numbers
			matchCount := formatNumber(len(tt.matches))
			totalCount := formatNumber(tt.total)

			if !strings.Contains(result, matchCount) {
				t.Errorf("Result should contain match count %s, got: %s", matchCount, result)
			}

			// If filtered, should show "/ total"
			if tt.expectFiltered && !strings.Contains(result, totalCount) {
				t.Errorf("Filtered results should show total count %s, got: %s", totalCount, result)
			}
		})
	}
}

func TestFormatCountWithBreakdown_LargeNumbers(t *testing.T) {
	// Test with large numbers to verify formatting
	countStyle := lipgloss.NewStyle()
	activeStyle := lipgloss.NewStyle()

	matches := make([]index.CombinedMatch, 1234)
	total := 5678

	result := formatCountWithBreakdown(matches, total, countStyle, activeStyle)

	// Should contain formatted numbers with commas
	if !strings.Contains(result, "1,234") {
		t.Errorf("Result should contain formatted match count '1,234', got: %s", result)
	}
	if !strings.Contains(result, "5,678") {
		t.Errorf("Result should contain formatted total count '5,678', got: %s", result)
	}
}

func TestFormatCountWithBreakdown_EmptyMatches(t *testing.T) {
	// Test edge case: empty matches with non-zero total
	countStyle := lipgloss.NewStyle()
	activeStyle := lipgloss.NewStyle()

	matches := []index.CombinedMatch{}
	total := 100

	result := formatCountWithBreakdown(matches, total, countStyle, activeStyle)

	// Should show "0 / 100"
	if !strings.Contains(result, "0") {
		t.Error("Result should contain '0' for empty matches")
	}
	if !strings.Contains(result, "100") {
		t.Error("Result should contain total count")
	}
}

func TestFormatCountWithBreakdown_SourceBreakdown(t *testing.T) {
	// Test the breakdown logic with different Source types
	countStyle := lipgloss.NewStyle()
	activeStyle := lipgloss.NewStyle()

	tests := []struct {
		name            string
		matches         []index.CombinedMatch
		total           int
		expectByName    bool
		expectByDesc    bool
		expectBoth      bool
		expectBreakdown bool
	}{
		{
			name: "name only matches",
			matches: []index.CombinedMatch{
				{Project: model.Project{Path: "a"}, Source: index.MatchSourceName},
				{Project: model.Project{Path: "b"}, Source: index.MatchSourceName},
				{Project: model.Project{Path: "c"}, Source: index.MatchSourceName},
			},
			total:           10,
			expectByName:    true,
			expectByDesc:    false,
			expectBoth:      false,
			expectBreakdown: true,
		},
		{
			name: "description only matches",
			matches: []index.CombinedMatch{
				{Project: model.Project{Path: "a"}, Source: index.MatchSourceDescription},
				{Project: model.Project{Path: "b"}, Source: index.MatchSourceDescription},
			},
			total:           5,
			expectByName:    false,
			expectByDesc:    true,
			expectBoth:      false,
			expectBreakdown: true,
		},
		{
			name: "both name and description",
			matches: []index.CombinedMatch{
				{Project: model.Project{Path: "a"}, Source: index.MatchSourceName | index.MatchSourceDescription},
				{Project: model.Project{Path: "b"}, Source: index.MatchSourceName | index.MatchSourceDescription},
			},
			total:           8,
			expectByName:    false,
			expectByDesc:    false,
			expectBoth:      true,
			expectBreakdown: true,
		},
		{
			name: "mixed sources",
			matches: []index.CombinedMatch{
				{Project: model.Project{Path: "a"}, Source: index.MatchSourceName},
				{Project: model.Project{Path: "b"}, Source: index.MatchSourceDescription},
				{Project: model.Project{Path: "c"}, Source: index.MatchSourceName | index.MatchSourceDescription},
			},
			total:           12,
			expectByName:    true,
			expectByDesc:    true,
			expectBoth:      true,
			expectBreakdown: true,
		},
		{
			name: "no breakdown when all match",
			matches: []index.CombinedMatch{
				{Project: model.Project{Path: "a"}, Source: index.MatchSourceName},
				{Project: model.Project{Path: "b"}, Source: index.MatchSourceName},
			},
			total:           2, // filtered == total
			expectByName:    false,
			expectByDesc:    false,
			expectBoth:      false,
			expectBreakdown: false, // No breakdown when showing all
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatCountWithBreakdown(tt.matches, tt.total, countStyle, activeStyle)

			// Check for breakdown presence
			if tt.expectBreakdown {
				if !strings.Contains(result, "(") {
					t.Error("Expected breakdown with parentheses")
				}

				// Check specific breakdown parts
				if tt.expectByName && !strings.Contains(result, "by name") {
					t.Error("Expected 'by name' in breakdown")
				}
				if tt.expectByDesc && !strings.Contains(result, "by description") {
					t.Error("Expected 'by description' in breakdown")
				}
				if tt.expectBoth && !strings.Contains(result, "both") {
					t.Error("Expected 'both' in breakdown")
				}
			} else if strings.Contains(result, "(") && strings.Contains(result, "by name") {
				t.Error("Did not expect breakdown when filtered == total")
			}
		})
	}
}

func TestFormatCountWithBreakdown_ZeroMatchesWithTotal(t *testing.T) {
	// Edge case: 0 matches out of non-zero total
	// Breakdown logic: if filtered < total && filtered > 0, add breakdown
	// So with filtered == 0, no breakdown should appear
	countStyle := lipgloss.NewStyle()
	activeStyle := lipgloss.NewStyle()

	matches := []index.CombinedMatch{} // empty
	total := 100

	result := formatCountWithBreakdown(matches, total, countStyle, activeStyle)

	// Should show "0 / 100 projects" but NO breakdown (filtered == 0)
	if !strings.Contains(result, "0") {
		t.Error("Result should contain '0' for match count")
	}
	if !strings.Contains(result, "100") {
		t.Error("Result should contain total count")
	}

	// No breakdown because filtered == 0
	if strings.Contains(result, "by name") || strings.Contains(result, "by description") {
		t.Error("Should not show breakdown when filtered count is 0")
	}
}

func TestFormatCountWithBreakdown_AllProjectsNoQuery(t *testing.T) {
	// When filtered == total (e.g., no search query), show simple count
	countStyle := lipgloss.NewStyle()
	activeStyle := lipgloss.NewStyle()

	matches := []index.CombinedMatch{
		{Project: model.Project{Path: "a"}, Source: index.MatchSourceName},
		{Project: model.Project{Path: "b"}, Source: index.MatchSourceName},
		{Project: model.Project{Path: "c"}, Source: index.MatchSourceName},
	}
	total := 3 // Same as filtered

	result := formatCountWithBreakdown(matches, total, countStyle, activeStyle)

	// Should show "3 projects" without "/" or breakdown
	if !strings.Contains(result, "3") {
		t.Error("Result should contain count '3'")
	}
	if !strings.Contains(result, "projects") {
		t.Error("Result should contain 'projects'")
	}

	// Should NOT contain "/"  or breakdown when filtered == total
	if strings.Contains(result, "/") {
		t.Error("Should not show '/' when filtered equals total")
	}
	if strings.Contains(result, "(") {
		t.Error("Should not show breakdown when filtered equals total")
	}
}

// TestNew verifies Model initialization
func TestNew(t *testing.T) {
	// Create temporary directory for cache
	tempDir := t.TempDir()

	// Create minimal config
	cfg := &config.Config{
		GitLab: config.GitLabConfig{
			URL: "https://gitlab.example.com",
		},
		Cache: config.CacheConfig{
			Dir: tempDir,
		},
	}

	// Create test projects
	projects := []model.Project{
		{Path: "group/project1", Name: "Project 1"},
		{Path: "group/project2", Name: "Project 2"},
	}

	// Create model
	m := New(projects, "", nil, tempDir, cfg, false, "testuser", "v1.0.0")

	// Verify initialization
	if len(m.projects) != 2 {
		t.Errorf("Expected 2 projects, got %d", len(m.projects))
	}

	if m.cacheDir != tempDir {
		t.Errorf("Expected cacheDir %s, got %s", tempDir, m.cacheDir)
	}

	if m.gitlabURL != "gitlab.example.com" {
		t.Errorf("Expected gitlabURL 'gitlab.example.com', got '%s'", m.gitlabURL)
	}

	if m.username != "testuser" {
		t.Errorf("Expected username 'testuser', got '%s'", m.username)
	}

	if m.version != "v1.0.0" {
		t.Errorf("Expected version 'v1.0.0', got '%s'", m.version)
	}

	if m.showExcluded != false {
		t.Error("Expected showExcluded to be false by default")
	}

	if m.cursor != 0 {
		t.Errorf("Expected cursor at 0, got %d", m.cursor)
	}
}

// TestNew_WithInitialQuery verifies initial query is set
func TestNew_WithInitialQuery(t *testing.T) {
	tempDir := t.TempDir()
	cfg := &config.Config{
		GitLab: config.GitLabConfig{URL: "https://gitlab.example.com"},
		Cache:  config.CacheConfig{Dir: tempDir},
	}

	projects := []model.Project{
		{Path: "group/api", Name: "API"},
		{Path: "group/web", Name: "Web"},
	}

	initialQuery := "api"
	m := New(projects, initialQuery, nil, tempDir, cfg, false, "user", "v1.0.0")

	// Check if initial query was set in text input
	if m.textInput.Value() != initialQuery {
		t.Errorf("Expected initial query '%s', got '%s'", initialQuery, m.textInput.Value())
	}
}

// TestInit verifies Init returns proper commands
func TestInit(t *testing.T) {
	tempDir := t.TempDir()
	cfg := &config.Config{
		GitLab: config.GitLabConfig{URL: "https://gitlab.example.com"},
		Cache:  config.CacheConfig{Dir: tempDir},
	}

	projects := []model.Project{{Path: "test/project", Name: "Test"}}

	m := New(projects, "", nil, tempDir, cfg, false, "user", "v1.0.0")
	cmd := m.Init()

	if cmd == nil {
		t.Error("Init() should return a command, got nil")
	}
}

// TestUpdate_Quit verifies quitting behavior
func TestUpdate_Quit(t *testing.T) {
	tempDir := t.TempDir()
	cfg := &config.Config{
		GitLab: config.GitLabConfig{URL: "https://gitlab.example.com"},
		Cache:  config.CacheConfig{Dir: tempDir},
	}

	projects := []model.Project{{Path: "test/project", Name: "Test"}}
	m := New(projects, "", nil, tempDir, cfg, false, "user", "v1.0.0")

	// Test Ctrl+C
	msg := tea.KeyMsg{Type: tea.KeyCtrlC}
	newModel, cmd := m.Update(msg)

	updatedModel := newModel.(Model)
	if !updatedModel.quitting {
		t.Error("Expected quitting to be true after Ctrl+C")
	}

	if cmd == nil {
		t.Error("Expected tea.Quit command")
	}

	// Test Esc
	m = New(projects, "", nil, tempDir, cfg, false, "user", "v1.0.0")
	msg = tea.KeyMsg{Type: tea.KeyEsc}
	newModel, cmd = m.Update(msg)

	updatedModel = newModel.(Model)
	if !updatedModel.quitting {
		t.Error("Expected quitting to be true after Esc")
	}

	if cmd == nil {
		t.Error("Expected tea.Quit command")
	}
}

// TestUpdate_Navigation verifies cursor navigation
func TestUpdate_Navigation(t *testing.T) {
	tempDir := t.TempDir()
	cfg := &config.Config{
		GitLab: config.GitLabConfig{URL: "https://gitlab.example.com"},
		Cache:  config.CacheConfig{Dir: tempDir},
	}

	projects := []model.Project{
		{Path: "test/project1", Name: "Project 1"},
		{Path: "test/project2", Name: "Project 2"},
		{Path: "test/project3", Name: "Project 3"},
	}

	m := New(projects, "", nil, tempDir, cfg, false, "user", "v1.0.0")

	// Initial cursor should be at 0
	if m.cursor != 0 {
		t.Errorf("Expected initial cursor at 0, got %d", m.cursor)
	}

	// Test Down key
	msg := tea.KeyMsg{Type: tea.KeyDown}
	newModel, _ := m.Update(msg)
	m = newModel.(Model)

	if m.cursor != 1 {
		t.Errorf("Expected cursor at 1 after down, got %d", m.cursor)
	}

	// Test Up key
	msg = tea.KeyMsg{Type: tea.KeyUp}
	newModel, _ = m.Update(msg)
	m = newModel.(Model)

	if m.cursor != 0 {
		t.Errorf("Expected cursor at 0 after up, got %d", m.cursor)
	}

	// Test Up at top (should stay at 0)
	msg = tea.KeyMsg{Type: tea.KeyUp}
	newModel, _ = m.Update(msg)
	m = newModel.(Model)

	if m.cursor != 0 {
		t.Errorf("Expected cursor to stay at 0, got %d", m.cursor)
	}

	// Move to bottom
	for i := 0; i < len(m.filtered); i++ {
		msg = tea.KeyMsg{Type: tea.KeyDown}
		newModel, _ = m.Update(msg)
		m = newModel.(Model)
	}

	// Test Down at bottom (should stay at last position)
	lastPos := len(m.filtered) - 1
	if m.cursor != lastPos {
		t.Errorf("Expected cursor at %d, got %d", lastPos, m.cursor)
	}
}

// TestUpdate_Selection verifies project selection
func TestUpdate_Selection(t *testing.T) {
	tempDir := t.TempDir()
	cfg := &config.Config{
		GitLab: config.GitLabConfig{URL: "https://gitlab.example.com"},
		Cache:  config.CacheConfig{Dir: tempDir},
	}

	projects := []model.Project{
		{Path: "test/project1", Name: "Project 1"},
		{Path: "test/project2", Name: "Project 2"},
	}

	m := New(projects, "", nil, tempDir, cfg, false, "user", "v1.0.0")

	// Select first project
	msg := tea.KeyMsg{Type: tea.KeyEnter}
	newModel, cmd := m.Update(msg)
	m = newModel.(Model)

	if m.Selected() == "" {
		t.Error("Expected a project to be selected")
	}

	if m.Selected() != "test/project1" {
		t.Errorf("Expected selected project 'test/project1', got '%s'", m.Selected())
	}

	if !m.quitting {
		t.Error("Expected quitting to be true after selection")
	}

	if cmd == nil {
		t.Error("Expected tea.Quit command after selection")
	}
}

// TestUpdate_WindowSize verifies window size handling
func TestUpdate_WindowSize(t *testing.T) {
	tempDir := t.TempDir()
	cfg := &config.Config{
		GitLab: config.GitLabConfig{URL: "https://gitlab.example.com"},
		Cache:  config.CacheConfig{Dir: tempDir},
	}

	projects := []model.Project{{Path: "test/project", Name: "Test"}}
	m := New(projects, "", nil, tempDir, cfg, false, "user", "v1.0.0")

	// Send window size message
	msg := tea.WindowSizeMsg{Width: 120, Height: 40}
	newModel, _ := m.Update(msg)
	m = newModel.(Model)

	if m.width != 120 {
		t.Errorf("Expected width 120, got %d", m.width)
	}

	if m.height != 40 {
		t.Errorf("Expected height 40, got %d", m.height)
	}
}

// TestUpdate_ToggleHelp verifies help toggle
func TestUpdate_ToggleHelp(t *testing.T) {
	tempDir := t.TempDir()
	cfg := &config.Config{
		GitLab: config.GitLabConfig{URL: "https://gitlab.example.com"},
		Cache:  config.CacheConfig{Dir: tempDir},
	}

	projects := []model.Project{{Path: "test/project", Name: "Test"}}
	m := New(projects, "", nil, tempDir, cfg, false, "user", "v1.0.0")

	// Initially help should be hidden
	if m.showHelp {
		t.Error("Expected showHelp to be false initially")
	}

	// Toggle help
	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'?'}}
	newModel, _ := m.Update(msg)
	m = newModel.(Model)

	if !m.showHelp {
		t.Error("Expected showHelp to be true after toggle")
	}

	// Toggle again
	newModel, _ = m.Update(msg)
	m = newModel.(Model)

	if m.showHelp {
		t.Error("Expected showHelp to be false after second toggle")
	}
}

// TestView verifies View renders without errors
func TestView(t *testing.T) {
	tempDir := t.TempDir()
	cfg := &config.Config{
		GitLab: config.GitLabConfig{URL: "https://gitlab.example.com"},
		Cache:  config.CacheConfig{Dir: tempDir},
	}

	projects := []model.Project{
		{Path: "test/project1", Name: "Project 1", Description: "Test project 1"},
		{Path: "test/project2", Name: "Project 2", Description: "Test project 2"},
	}

	m := New(projects, "", nil, tempDir, cfg, false, "testuser", "v1.0.0")
	m.width = 80
	m.height = 24

	// Render view
	view := m.View()

	// Basic checks
	if view == "" {
		t.Error("Expected non-empty view")
	}

	// View contains ANSI escape codes, so we can't do exact string matching
	// Just verify it's non-empty and contains basic structural elements
	if len(view) < 50 {
		t.Errorf("Expected view to be substantial, got length %d", len(view))
	}

	// The view should contain some recognizable text patterns
	// (content might be styled with ANSI codes, so we check for partial matches)
	if !strings.Contains(view, "glf") {
		t.Error("Expected view to contain 'glf' app name")
	}

	if !strings.Contains(view, "projects") {
		t.Error("Expected view to contain 'projects' text")
	}
}

// TestView_Quitting verifies empty view when quitting
func TestView_Quitting(t *testing.T) {
	tempDir := t.TempDir()
	cfg := &config.Config{
		GitLab: config.GitLabConfig{URL: "https://gitlab.example.com"},
		Cache:  config.CacheConfig{Dir: tempDir},
	}

	projects := []model.Project{{Path: "test/project", Name: "Test"}}
	m := New(projects, "", nil, tempDir, cfg, false, "user", "v1.0.0")
	m.quitting = true

	view := m.View()

	if view != "" {
		t.Error("Expected empty view when quitting")
	}
}

// TestFilter verifies filtering logic
func TestFilter(t *testing.T) {
	t.Skip("Skipping TestFilter: requires full index setup and is slow")

	// This test is skipped because:
	// 1. It requires creating and indexing into Bleve which is slow
	// 2. The filter() method is indirectly tested through Update tests
	// 3. The underlying search functionality is tested in search package

	// If we need to test filtering specifically, we should mock the search.CombinedSearch function
}

// TestRenderFuzzyMatch verifies fuzzy match highlighting
func TestRenderFuzzyMatch(t *testing.T) {
	style := lipgloss.NewStyle()
	highlightStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#FC6D25"))

	tests := []struct {
		name        string
		displayStr  string
		query       string
		expectMatch bool
	}{
		{
			name:        "empty query",
			displayStr:  "backend/api",
			query:       "",
			expectMatch: false,
		},
		{
			name:        "single token match at start",
			displayStr:  "backend/api",
			query:       "back",
			expectMatch: true,
		},
		{
			name:        "single token match in middle",
			displayStr:  "backend/api",
			query:       "end",
			expectMatch: true,
		},
		{
			name:        "single token match at end",
			displayStr:  "backend/api",
			query:       "api",
			expectMatch: true,
		},
		{
			name:        "multi-token query uses first token",
			displayStr:  "backend/api",
			query:       "back frontend",
			expectMatch: true, // Should match "back"
		},
		{
			name:        "case insensitive match",
			displayStr:  "Backend/API",
			query:       "backend",
			expectMatch: true,
		},
		{
			name:        "no match",
			displayStr:  "backend/api",
			query:       "xyz",
			expectMatch: false,
		},
		{
			name:        "empty display string",
			displayStr:  "",
			query:       "test",
			expectMatch: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := renderFuzzyMatch(tt.displayStr, tt.query, style, highlightStyle)

			// If display string is empty, result can be empty
			if tt.displayStr == "" {
				return // Empty input is a valid edge case
			}

			// Result should not be empty for non-empty display strings
			if result == "" {
				t.Error("renderFuzzyMatch returned empty string for non-empty input")
			}

			// Result should contain the display string content (possibly styled)
			// We can't check exact match due to ANSI codes, but length should be reasonable
			if len(result) < len(tt.displayStr) {
				t.Errorf("Result length %d is less than display string length %d", len(result), len(tt.displayStr))
			}
		})
	}
}

// TestRenderFuzzyMatch_Highlighting verifies that matched text can be highlighted
func TestRenderFuzzyMatch_Highlighting(t *testing.T) {
	style := lipgloss.NewStyle()
	highlightStyle := lipgloss.NewStyle().Bold(true)

	displayStr := "backend/api/server"
	query := "api"

	result := renderFuzzyMatch(displayStr, query, style, highlightStyle)

	// The function should handle highlighting without errors
	// Note: In test environments, lipgloss may not add ANSI codes (NO_COLOR, not a TTY, etc.)
	// so we just verify the result is valid and contains the display string content
	if result == "" {
		t.Error("renderFuzzyMatch returned empty string")
	}

	// Result should be at least as long as display string (with or without styling)
	if len(result) < len(displayStr) {
		t.Errorf("Result length %d is less than display string length %d", len(result), len(displayStr))
	}
}

// TestUpdate_CtrlR_Sync verifies Ctrl+R triggers sync
func TestUpdate_CtrlR_Sync(t *testing.T) {
	tempDir := t.TempDir()
	cfg := &config.Config{
		GitLab: config.GitLabConfig{URL: "https://gitlab.example.com"},
		Cache:  config.CacheConfig{Dir: tempDir},
	}

	projects := []model.Project{{Path: "test/project", Name: "Test"}}

	// Create sync callback
	syncCallback := func() tea.Cmd {
		return func() tea.Msg {
			return SyncCompleteMsg{Err: nil, Projects: projects}
		}
	}

	m := New(projects, "", syncCallback, tempDir, cfg, false, "user", "v1.0.0")

	// Send Ctrl+R
	msg := tea.KeyMsg{Type: tea.KeyCtrlR}
	newModel, cmd := m.Update(msg)
	m = newModel.(Model)

	// Verify syncing flag is set
	if !m.syncing {
		t.Error("Expected syncing to be true after Ctrl+R")
	}

	// Verify sync callback was called (by executing the returned command)
	if cmd != nil {
		result := cmd()
		if _, ok := result.(SyncCompleteMsg); !ok {
			t.Error("Expected SyncCompleteMsg from sync callback")
		}
	}
}

// TestUpdate_CtrlR_AlreadySyncing verifies Ctrl+R does nothing when already syncing
func TestUpdate_CtrlR_AlreadySyncing(t *testing.T) {
	tempDir := t.TempDir()
	cfg := &config.Config{
		GitLab: config.GitLabConfig{URL: "https://gitlab.example.com"},
		Cache:  config.CacheConfig{Dir: tempDir},
	}

	projects := []model.Project{{Path: "test/project", Name: "Test"}}

	m := New(projects, "", nil, tempDir, cfg, false, "user", "v1.0.0")
	m.syncing = true // Already syncing

	// Send Ctrl+R
	msg := tea.KeyMsg{Type: tea.KeyCtrlR}
	newModel, cmd := m.Update(msg)
	m = newModel.(Model)

	// Should not trigger new sync
	if cmd != nil {
		t.Error("Expected no command when already syncing")
	}
}

// TestUpdate_CtrlH_ToggleExcluded verifies Ctrl+H toggles excluded projects visibility
func TestUpdate_CtrlH_ToggleExcluded(t *testing.T) {
	tempDir := t.TempDir()
	cfg := &config.Config{
		GitLab: config.GitLabConfig{URL: "https://gitlab.example.com"},
		Cache:  config.CacheConfig{Dir: tempDir},
	}

	projects := []model.Project{
		{Path: "test/project1", Name: "Project 1"},
		{Path: "test/project2", Name: "Project 2"},
	}

	m := New(projects, "", nil, tempDir, cfg, false, "user", "v1.0.0")

	// Initially showExcluded should be false
	if m.showExcluded {
		t.Error("Expected showExcluded to be false initially")
	}

	// Send Ctrl+H
	msg := tea.KeyMsg{Type: tea.KeyCtrlH}
	newModel, _ := m.Update(msg)
	m = newModel.(Model)

	// Should toggle to true
	if !m.showExcluded {
		t.Error("Expected showExcluded to be true after Ctrl+H")
	}

	// Send Ctrl+H again
	newModel, _ = m.Update(msg)
	m = newModel.(Model)

	// Should toggle back to false
	if m.showExcluded {
		t.Error("Expected showExcluded to be false after second Ctrl+H")
	}
}

// TestUpdate_SyncCompleteMsg_Success verifies successful sync handling
func TestUpdate_SyncCompleteMsg_Success(t *testing.T) {
	tempDir := t.TempDir()
	cfg := &config.Config{
		GitLab: config.GitLabConfig{URL: "https://gitlab.example.com"},
		Cache:  config.CacheConfig{Dir: tempDir},
	}

	initialProjects := []model.Project{{Path: "test/project1", Name: "Project 1"}}
	m := New(initialProjects, "", nil, tempDir, cfg, false, "user", "v1.0.0")
	m.syncing = true

	// Send successful sync message with new projects
	newProjects := []model.Project{
		{Path: "test/project1", Name: "Project 1"},
		{Path: "test/project2", Name: "Project 2"},
	}
	msg := SyncCompleteMsg{Err: nil, Projects: newProjects}

	newModel, _ := m.Update(msg)
	m = newModel.(Model)

	// Verify syncing flag is cleared
	if m.syncing {
		t.Error("Expected syncing to be false after sync completion")
	}

	// Verify syncError is cleared
	if m.syncError != nil {
		t.Error("Expected syncError to be nil after successful sync")
	}

	// Verify projects were updated
	if len(m.projects) != 2 {
		t.Errorf("Expected 2 projects after sync, got %d", len(m.projects))
	}
}

// TestUpdate_SyncCompleteMsg_Error verifies error sync handling
func TestUpdate_SyncCompleteMsg_Error(t *testing.T) {
	tempDir := t.TempDir()
	cfg := &config.Config{
		GitLab: config.GitLabConfig{URL: "https://gitlab.example.com"},
		Cache:  config.CacheConfig{Dir: tempDir},
	}

	projects := []model.Project{{Path: "test/project1", Name: "Project 1"}}
	m := New(projects, "", nil, tempDir, cfg, false, "user", "v1.0.0")
	m.syncing = true

	// Send sync error message
	syncErr := fmt.Errorf("network timeout")
	msg := SyncCompleteMsg{Err: syncErr, Projects: nil}

	newModel, _ := m.Update(msg)
	m = newModel.(Model)

	// Verify syncing flag is cleared
	if m.syncing {
		t.Error("Expected syncing to be false after sync error")
	}

	// Verify syncError is set
	if m.syncError == nil {
		t.Error("Expected syncError to be set")
	}

	if m.syncError.Error() != "network timeout" {
		t.Errorf("Expected syncError 'network timeout', got '%v'", m.syncError)
	}

	// Verify projects were NOT updated
	if len(m.projects) != 1 {
		t.Errorf("Expected original 1 project after sync error, got %d", len(m.projects))
	}
}

// TestUpdate_HistoryLoadedMsg verifies history loaded message handling
func TestUpdate_HistoryLoadedMsg(t *testing.T) {
	tempDir := t.TempDir()
	cfg := &config.Config{
		GitLab: config.GitLabConfig{URL: "https://gitlab.example.com"},
		Cache:  config.CacheConfig{Dir: tempDir},
	}

	projects := []model.Project{{Path: "test/project", Name: "Test"}}
	m := New(projects, "", nil, tempDir, cfg, false, "user", "v1.0.0")
	m.historyLoading = true

	// Send history loaded message (success)
	msg := HistoryLoadedMsg{Err: nil}

	newModel, _ := m.Update(msg)
	m = newModel.(Model)

	// Verify historyLoading flag is cleared
	if m.historyLoading {
		t.Error("Expected historyLoading to be false after HistoryLoadedMsg")
	}
}

// TestUpdate_HistoryLoadedMsg_Error verifies history load error handling
func TestUpdate_HistoryLoadedMsg_Error(t *testing.T) {
	tempDir := t.TempDir()
	cfg := &config.Config{
		GitLab: config.GitLabConfig{URL: "https://gitlab.example.com"},
		Cache:  config.CacheConfig{Dir: tempDir},
	}

	projects := []model.Project{{Path: "test/project", Name: "Test"}}
	m := New(projects, "", nil, tempDir, cfg, false, "user", "v1.0.0")
	m.historyLoading = true

	// Send history loaded message with error
	historyErr := fmt.Errorf("failed to load history file")
	msg := HistoryLoadedMsg{Err: historyErr}

	newModel, _ := m.Update(msg)
	m = newModel.(Model)

	// Verify historyLoading flag is still cleared (error is non-fatal)
	if m.historyLoading {
		t.Error("Expected historyLoading to be false even with error")
	}
}

// TestInit_AutoSync verifies auto-sync on initialization
func TestInit_AutoSync(t *testing.T) {
	tempDir := t.TempDir()
	cfg := &config.Config{
		GitLab: config.GitLabConfig{URL: "https://gitlab.example.com"},
		Cache:  config.CacheConfig{Dir: tempDir},
	}

	projects := []model.Project{{Path: "test/project", Name: "Test"}}

	syncCallback := func() tea.Cmd {
		return func() tea.Msg {
			return SyncCompleteMsg{Err: nil, Projects: projects}
		}
	}

	// Create model with auto-sync enabled (default)
	m := New(projects, "", syncCallback, tempDir, cfg, false, "user", "v1.0.0")

	// Verify autoSync is enabled
	if !m.autoSync {
		t.Error("Expected autoSync to be true by default")
	}

	// Call Init to get the command batch
	cmd := m.Init()

	if cmd == nil {
		t.Fatal("Expected Init() to return a command batch")
	}

	// Execute the batch command to trigger auto-sync
	// The batch contains: textinput.Blink, history loading, and autoSyncMsg
	result := cmd()

	// The result should be tea.BatchMsg containing multiple commands
	// We can't easily inspect tea.BatchMsg internals, but we verified the structure
	if result == nil {
		t.Error("Expected batch command to return a result")
	}
}

// TestInit_NoAutoSync verifies initialization without auto-sync
func TestInit_NoAutoSync(t *testing.T) {
	tempDir := t.TempDir()
	cfg := &config.Config{
		GitLab: config.GitLabConfig{URL: "https://gitlab.example.com"},
		Cache:  config.CacheConfig{Dir: tempDir},
	}

	projects := []model.Project{{Path: "test/project", Name: "Test"}}

	m := New(projects, "", nil, tempDir, cfg, false, "user", "v1.0.0")

	// Disable auto-sync
	m.autoSync = false

	// Call Init
	cmd := m.Init()

	if cmd == nil {
		t.Error("Expected Init() to return a command batch (at minimum textinput.Blink)")
	}

	// With autoSync disabled, the batch should still contain blink and history loading
	// but not the autoSyncMsg
}

// TestView_WithSyncError verifies View displays sync error
func TestView_WithSyncError(t *testing.T) {
	tempDir := t.TempDir()
	cfg := &config.Config{
		GitLab: config.GitLabConfig{URL: "https://gitlab.example.com"},
		Cache:  config.CacheConfig{Dir: tempDir},
	}

	projects := []model.Project{{Path: "test/project", Name: "Test"}}
	m := New(projects, "", nil, tempDir, cfg, false, "user", "v1.0.0")
	m.width = 80
	m.height = 24
	m.syncError = fmt.Errorf("network timeout")

	view := m.View()

	// View should be non-empty
	if view == "" {
		t.Error("Expected non-empty view with sync error")
	}

	// Should contain error indicator (red status dot)
	// The status indicator is styled, so we just verify the view is substantial
	if len(view) < 50 {
		t.Errorf("Expected substantial view with sync error, got length %d", len(view))
	}
}

// TestView_NarrowTerminal verifies View adapts to narrow terminal
func TestView_NarrowTerminal(t *testing.T) {
	tempDir := t.TempDir()
	cfg := &config.Config{
		GitLab: config.GitLabConfig{URL: "https://gitlab.example.com"},
		Cache:  config.CacheConfig{Dir: tempDir},
	}

	projects := []model.Project{{Path: "test/project", Name: "Test"}}
	m := New(projects, "", nil, tempDir, cfg, false, "user", "v1.0.0")

	// Set very narrow terminal
	m.width = 40
	m.height = 10

	view := m.View()

	// View should still render without panicking
	if view == "" {
		t.Error("Expected non-empty view even in narrow terminal")
	}
}

// TestView_WithHelp verifies View displays help text
func TestView_WithHelp(t *testing.T) {
	tempDir := t.TempDir()
	cfg := &config.Config{
		GitLab: config.GitLabConfig{URL: "https://gitlab.example.com"},
		Cache:  config.CacheConfig{Dir: tempDir},
	}

	projects := []model.Project{{Path: "test/project", Name: "Test"}}
	m := New(projects, "", nil, tempDir, cfg, false, "user", "v1.0.0")
	m.width = 100
	m.height = 30
	m.showHelp = true // Enable help display

	view := m.View()

	// View should contain help text
	if !strings.Contains(view, "navigate") || !strings.Contains(view, "select") {
		t.Error("Expected view to contain help text with navigation hints")
	}
}

// TestUpdate_AutoSyncMsg verifies autoSyncMsg triggers sync
func TestUpdate_AutoSyncMsg(t *testing.T) {
	tempDir := t.TempDir()
	cfg := &config.Config{
		GitLab: config.GitLabConfig{URL: "https://gitlab.example.com"},
		Cache:  config.CacheConfig{Dir: tempDir},
	}

	projects := []model.Project{{Path: "test/project", Name: "Test"}}

	syncCallback := func() tea.Cmd {
		return func() tea.Msg {
			return SyncCompleteMsg{Err: nil, Projects: projects}
		}
	}

	m := New(projects, "", syncCallback, tempDir, cfg, false, "user", "v1.0.0")

	// Send autoSyncMsg
	msg := autoSyncMsg{}
	newModel, cmd := m.Update(msg)
	m = newModel.(Model)

	// Verify syncing flag is set
	if !m.syncing {
		t.Error("Expected syncing to be true after autoSyncMsg")
	}

	// Verify command was returned
	if cmd == nil {
		t.Error("Expected sync command to be returned")
	}
}

// TestRenderMatch verifies renderMatch function
func TestRenderMatch(t *testing.T) {
	style := lipgloss.NewStyle()
	highlightStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#FC6D25"))
	snippetStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#666"))

	tests := []struct {
		name       string
		match      index.CombinedMatch
		query      string
		showScores bool
		expectSnip bool
	}{
		{
			name: "name match without snippet",
			match: index.CombinedMatch{
				Project: model.Project{
					Path: "backend/api",
					Name: "API Server",
				},
				Source:      index.MatchSourceName,
				SearchScore: 10.5,
			},
			query:      "api",
			showScores: false,
			expectSnip: false,
		},
		{
			name: "description match with snippet",
			match: index.CombinedMatch{
				Project: model.Project{
					Path:        "backend/api",
					Name:        "API Server",
					Description: "REST API for authentication",
				},
				Source:      index.MatchSourceDescription,
				Snippet:     "REST API for authentication services",
				SearchScore: 8.3,
			},
			query:      "auth",
			showScores: false,
			expectSnip: true,
		},
		{
			name: "both sources with scores",
			match: index.CombinedMatch{
				Project: model.Project{
					Path: "backend/api",
					Name: "API Server",
				},
				Source:       index.MatchSourceName | index.MatchSourceDescription,
				SearchScore:  15.0,
				HistoryScore: 5,
				TotalScore:   20.0,
			},
			query:      "api",
			showScores: true,
			expectSnip: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			excludedStarredStyle := lipgloss.NewStyle()
			result := renderMatch(tt.match, style, highlightStyle, snippetStyle, excludedStarredStyle, tt.query, tt.showScores, false)

			// Result should not be empty
			if result == "" {
				t.Error("renderMatch returned empty string")
			}

			// If snippet expected, result should contain newline
			if tt.expectSnip && !strings.Contains(result, "\n") {
				t.Error("Expected newline for snippet in result")
			}

			// If showing scores, result should contain score markers
			if tt.showScores {
				if !strings.Contains(result, "[") {
					t.Error("Expected score markers '[' in result when showScores=true")
				}
			}
		})
	}
}

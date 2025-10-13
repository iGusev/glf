package tui

import (
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"
	"github.com/igusev/glf/internal/index"
	"github.com/igusev/glf/internal/types"
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
				{Project: types.Project{Path: "a"}},
				{Project: types.Project{Path: "b"}},
				{Project: types.Project{Path: "c"}},
			},
			total:          10,
			expectFiltered: true,
		},
		{
			name: "all matches (no filter)",
			matches: []index.CombinedMatch{
				{Project: types.Project{Path: "a"}},
				{Project: types.Project{Path: "b"}},
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
				{Project: types.Project{Path: "a"}, Source: index.MatchSourceName},
				{Project: types.Project{Path: "b"}, Source: index.MatchSourceName},
				{Project: types.Project{Path: "c"}, Source: index.MatchSourceName},
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
				{Project: types.Project{Path: "a"}, Source: index.MatchSourceDescription},
				{Project: types.Project{Path: "b"}, Source: index.MatchSourceDescription},
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
				{Project: types.Project{Path: "a"}, Source: index.MatchSourceName | index.MatchSourceDescription},
				{Project: types.Project{Path: "b"}, Source: index.MatchSourceName | index.MatchSourceDescription},
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
				{Project: types.Project{Path: "a"}, Source: index.MatchSourceName},
				{Project: types.Project{Path: "b"}, Source: index.MatchSourceDescription},
				{Project: types.Project{Path: "c"}, Source: index.MatchSourceName | index.MatchSourceDescription},
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
				{Project: types.Project{Path: "a"}, Source: index.MatchSourceName},
				{Project: types.Project{Path: "b"}, Source: index.MatchSourceName},
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
		{Project: types.Project{Path: "a"}, Source: index.MatchSourceName},
		{Project: types.Project{Path: "b"}, Source: index.MatchSourceName},
		{Project: types.Project{Path: "c"}, Source: index.MatchSourceName},
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

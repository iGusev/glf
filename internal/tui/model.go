package tui

import (
	"fmt"
	"path/filepath"
	"strings"
	"unicode"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/igusev/glf/internal/config"
	"github.com/igusev/glf/internal/history"
	"github.com/igusev/glf/internal/index"
	"github.com/igusev/glf/internal/search"
	"github.com/igusev/glf/internal/types"
)

// SyncStartMsg is sent when sync starts
type SyncStartMsg struct{}

// SyncCompleteMsg is sent when sync completes
type SyncCompleteMsg struct {
	Err      error
	Projects []types.Project
}

// HistoryLoadedMsg is sent when history finishes loading
type HistoryLoadedMsg struct {
	Err error
}

// Model represents the TUI state
type Model struct {
	textInput   textinput.Model       // Search input field
	styles      Styles                // Pre-configured styles
	projects    []types.Project       // All projects (full list)
	filtered    []index.CombinedMatch // Filtered projects with match data (fuzzy + description)
	selected    string                // Selected project path (when user presses Enter)
	cacheDir    string                // Cache directory for description index
	gitlabURL   string                // GitLab server URL (for header display)
	username    string                // GitLab username (for header display)
	version     string                // Application version
	syncError   error                 // Sync error if any
	history     *history.History      // Selection frequency tracker
	config      *config.Config        // Application config (for exclusions)
	colorScheme *ColorScheme          // Adaptive color scheme
	onSync      func() tea.Cmd        // Callback to trigger sync
	cursor      int                   // Current cursor position in filtered list
	width       int                   // Terminal width
	height      int                   // Terminal height
	quitting       bool // Whether user is quitting
	syncing        bool // Whether sync is in progress
	autoSync       bool // Whether to auto-sync on start
	historyLoading bool // Whether history is being loaded
	showExcluded   bool // Whether to show excluded projects
	showScores     bool // Whether to show score breakdown
	showHelp       bool // Whether to show help text
}

// New creates a new TUI model with the given projects and optional initial query
func New(projects []types.Project, initialQuery string, onSync func() tea.Cmd, cacheDir string, cfg *config.Config, showScores bool, username string, version string) Model {
	// Initialize color scheme
	colorScheme := NewColorScheme()
	styles := colorScheme.GetStyles()

	ti := textinput.New()
	ti.Placeholder = "Search projects..."
	ti.Focus()
	ti.CharLimit = 156
	ti.Width = 50

	// Set prompt with adaptive color
	ti.Prompt = "> "
	ti.PromptStyle = styles.Prompt

	// Set initial query if provided
	if initialQuery != "" {
		ti.SetValue(initialQuery)
	}

	// Initialize history
	historyPath := filepath.Join(cacheDir, "history.gob")
	hist := history.New(historyPath)

	// Extract GitLab URL for display (remove protocol and trailing slash)
	gitlabURL := cfg.GitLab.URL
	gitlabURL = strings.TrimPrefix(gitlabURL, "https://")
	gitlabURL = strings.TrimPrefix(gitlabURL, "http://")
	gitlabURL = strings.TrimSuffix(gitlabURL, "/")

	m := Model{
		textInput:      ti,
		projects:       projects,
		filtered:       []index.CombinedMatch{}, // Will be set by filter()
		cursor:         0,
		onSync:         onSync,
		autoSync:       true, // Enable auto-sync on start
		history:        hist,
		historyLoading: true, // Will be loaded async
		config:         cfg,
		showExcluded:   false, // Hide excluded projects by default
		cacheDir:       cacheDir,
		showScores:     showScores, // Show score breakdown if requested
		colorScheme:    colorScheme,
		styles:         styles,
		gitlabURL:      gitlabURL,
		username:       username,
		version:        version, // Injected from build-time ldflags
		showHelp:       false,   // Hide help by default
	}

	// Always apply filter on initialization to respect exclusions
	m.filter()

	return m
}

// autoSyncMsg is sent on startup to trigger auto-sync
type autoSyncMsg struct{}

// Init initializes the model (required by tea.Model interface)
func (m Model) Init() tea.Cmd {
	cmds := []tea.Cmd{textinput.Blink}

	// Start async history loading
	if m.history != nil {
		cmds = append(cmds, func() tea.Msg {
			errCh := m.history.LoadAsync()
			err := <-errCh
			return HistoryLoadedMsg{Err: err}
		})
	}

	// If auto-sync is enabled, trigger it
	if m.autoSync && m.onSync != nil {
		cmds = append(cmds, func() tea.Msg {
			return autoSyncMsg{}
		})
	}

	return tea.Batch(cmds...)
}

// Update handles messages and updates the model
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "esc":
			m.quitting = true
			// Save history before quitting
			if m.history != nil {
				if err := m.history.Save(); err != nil {
					// Silently fail - don't prevent quit
					_ = err // explicitly ignore error
				}
			}
			return m, tea.Quit

		case "ctrl+r":
			// Trigger sync (only if not already syncing)
			if m.onSync != nil && !m.syncing {
				m.syncing = true
				m.syncError = nil
				return m, m.onSync()
			}

		case "enter":
			// Select current project
			if len(m.filtered) > 0 && m.cursor < len(m.filtered) {
				selectedProject := m.filtered[m.cursor].Project
				m.selected = selectedProject.Path

				// Record selection in history with query context for smart boosting
				if m.history != nil && m.selected != "" {
					query := strings.TrimSpace(m.textInput.Value())
					m.history.RecordSelectionWithQuery(query, m.selected)
					if err := m.history.Save(); err != nil {
						// Silently fail - don't prevent selection
					_ = err // explicitly ignore error
					}
				}
			}
			m.quitting = true
			return m, tea.Quit

		case "ctrl+x":
			// Toggle exclusion: exclude if visible, un-exclude if already excluded
			if m.config != nil && len(m.filtered) > 0 && m.cursor < len(m.filtered) {
				projectPath := m.filtered[m.cursor].Project.Path
				if m.config.IsExcluded(projectPath) {
					// Already excluded - un-exclude it
					if err := m.config.RemoveExclusionForPath(projectPath); err != nil {
						_ = err // explicitly ignore error
						// Silently fail - don't prevent UI operation
					}
				} else {
					// Not excluded - exclude it
					if err := m.config.AddExclusion(projectPath); err != nil {
						_ = err // explicitly ignore error
						// Silently fail - don't prevent UI operation
					}
				}
				// Re-filter to apply changes
				m.filter()
				// Adjust cursor if needed
				if m.cursor >= len(m.filtered) && m.cursor > 0 {
					m.cursor = len(m.filtered) - 1
				}
			}

		case "ctrl+h":
			// Toggle showing excluded projects
			m.showExcluded = !m.showExcluded
			m.filter()
			// Reset cursor
			if m.cursor >= len(m.filtered) && m.cursor > 0 {
				m.cursor = len(m.filtered) - 1
			}

		case "?":
			// Toggle help text
			m.showHelp = !m.showHelp

		case "down", "ctrl+n":
			if m.cursor < len(m.filtered)-1 {
				m.cursor++
			}

		case "up", "ctrl+p":
			if m.cursor > 0 {
				m.cursor--
			}

		default:
			// Update text input and filter
			m.textInput, cmd = m.textInput.Update(msg)
			m.filter()
			m.cursor = 0 // Reset cursor when query changes
		}

	case autoSyncMsg:
		// Trigger background sync on startup
		if m.onSync != nil && !m.syncing {
			m.syncing = true
			m.syncError = nil
			return m, m.onSync()
		}

	case SyncCompleteMsg:
		m.syncing = false
		if msg.Err != nil {
			m.syncError = msg.Err
		} else {
			// Update projects list
			m.projects = msg.Projects
			m.filter()
			m.syncError = nil
		}

	case HistoryLoadedMsg:
		m.historyLoading = false
		if msg.Err != nil {
			// Log error but don't fail - history is optional
			// Could add error display here if needed
		} else {
			// Re-filter with history loaded
			m.filter()
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
	}

	return m, cmd
}

// filter filters projects using combined search (fuzzy + description full-text)
func (m *Model) filter() {
	query := strings.TrimSpace(m.textInput.Value())

	// Get query-specific history scores if available (includes global + query-specific boost)
	var historyScores map[string]int
	if m.history != nil && !m.historyLoading {
		historyScores = m.history.GetAllScoresForQuery(query)
	} else {
		historyScores = make(map[string]int)
	}

	// Use combined search (Bleve unified search)
	allMatches, err := search.CombinedSearch(query, m.projects, historyScores, m.cacheDir)
	if err != nil {
		// Search failed - show empty results
		// User should run 'glf sync' to build/rebuild the index
		allMatches = []index.CombinedMatch{}
	}

	// Apply exclusion filter if needed
	if m.config != nil && !m.showExcluded {
		filtered := make([]index.CombinedMatch, 0, len(allMatches))
		for _, match := range allMatches {
			if !m.config.IsExcluded(match.Project.Path) {
				filtered = append(filtered, match)
			}
		}
		m.filtered = filtered
	} else {
		m.filtered = allMatches
	}
}

// renderMatch renders a matched project with visual indicators and optional snippet
// Returns multiple lines if snippet is present and item is selected
func renderMatch(match index.CombinedMatch, style lipgloss.Style, highlightStyle lipgloss.Style, snippetStyle lipgloss.Style, query string, isSelected bool, showScores bool) string {
	var result strings.Builder

	// Get display string
	displayStr := match.Project.DisplayString()

	// Render project name with highlighting if matched by name
	if match.Source&index.MatchSourceName != 0 {
		// Fuzzy match - need to compute and highlight matched positions
		result.WriteString(renderFuzzyMatch(displayStr, query, style, highlightStyle))
	} else {
		// Description-only match - no highlighting
		result.WriteString(style.Render(displayStr))
	}

	// Add score breakdown if requested
	if showScores {
		scoreStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("241")) // Gray
		scoreText := fmt.Sprintf(" [S:%.3f H:%d T:%.2f]",
			match.SearchScore,
			match.HistoryScore,
			match.TotalScore)
		result.WriteString(scoreStyle.Render(scoreText))
	}

	// Add snippet if available (description match) - always show if present
	if match.Snippet != "" {
		// Truncate snippet to 60 runes at word boundary for UTF-8 safety
		snippet := truncateSnippet(match.Snippet, 60)
		result.WriteString("\n") // Newline for snippet (indent handled by caller)
		result.WriteString(snippetStyle.Render(snippet))
	}

	return result.String()
}

// renderFuzzyMatch performs substring highlighting on display string
func renderFuzzyMatch(displayStr, query string, style lipgloss.Style, highlightStyle lipgloss.Style) string {
	if query == "" {
		return style.Render(displayStr)
	}

	// For multi-token queries, just use first token for highlighting
	tokens := strings.Fields(query)
	if len(tokens) == 0 {
		return style.Render(displayStr)
	}
	matchToken := tokens[0]

	// Find substring match (case-insensitive)
	lowerDisplay := strings.ToLower(displayStr)
	lowerToken := strings.ToLower(matchToken)

	idx := strings.Index(lowerDisplay, lowerToken)
	if idx < 0 {
		// No match found - return unstyled
		return style.Render(displayStr)
	}

	// Highlight the matched substring
	before := displayStr[:idx]
	matched := displayStr[idx : idx+len(matchToken)]
	after := displayStr[idx+len(matchToken):]

	return style.Render(before) + highlightStyle.Render(matched) + style.Render(after)
}

// View renders the TUI
func (m Model) View() string {
	if m.quitting {
		return ""
	}

	// Build UI
	var b strings.Builder

	// Status indicator: ○ idle, ● active (green) or error (red)
	var statusIndicator string
	if m.syncing || m.historyLoading {
		statusIndicator = m.styles.StatusActive.Render("●")
	} else if m.syncError != nil {
		statusIndicator = m.styles.StatusError.Render("●")
	} else {
		statusIndicator = m.styles.StatusIdle.Render("○")
	}

	// Title line: wave + app name + version on left
	titleLeft := fmt.Sprintf("%s %s %s",
		m.colorScheme.GitLabWave,
		m.styles.Title.Render("glf"),
		m.styles.Version.Render(m.version))

	// Project count (always shown)
	projectCount := fmt.Sprintf("%d/%d projects",
		len(m.filtered),
		len(m.projects))

	// Additional info (for wider screens)
	serverInfo := fmt.Sprintf("[ @%s on %s ]", m.username, m.gitlabURL)
	helpIndicator := m.styles.Help.Render("[?] Help")

	// Adaptive layout based on terminal width
	leftWidth := lipgloss.Width(titleLeft)
	countWidth := lipgloss.Width(projectCount)
	serverWidth := lipgloss.Width(serverInfo)
	statusWidth := lipgloss.Width(statusIndicator)

	var titleRight string

	// Minimum width: just count + status (e.g., "36/648 projects ○")
	minWidth := leftWidth + countWidth + statusWidth + 4 // +4 for spacing

	if m.width < minWidth+30 {
		// Very narrow: only glf + count + status
		titleRight = fmt.Sprintf("%s %s",
			m.styles.Count.Render(projectCount),
			statusIndicator)
	} else if m.width < minWidth+serverWidth+30 {
		// Medium: glf + count + help + status
		titleRight = fmt.Sprintf("%s %s %s",
			m.styles.Count.Render(projectCount),
			helpIndicator,
			statusIndicator)
	} else {
		// Wide: full display with server info
		titleRight = fmt.Sprintf("%s %s %s %s",
			m.styles.Count.Render(projectCount),
			m.styles.ServerInfo.Render(serverInfo),
			helpIndicator,
			statusIndicator)
	}

	// Calculate spacing to align right
	rightWidth := lipgloss.Width(titleRight)
	spacing := ""
	if m.width > leftWidth+rightWidth {
		spacing = strings.Repeat(" ", m.width-leftWidth-rightWidth)
	}

	b.WriteString(titleLeft)
	b.WriteString(spacing)
	b.WriteString(titleRight)
	b.WriteString("\n")

	// Separator line (full width)
	if m.width > 0 {
		separator := strings.Repeat("─", m.width)
		b.WriteString(m.styles.Help.Render(separator))
		b.WriteString("\n")
	}

	// Search input (fixed at top, after header)
	b.WriteString("\n")
	b.WriteString(m.textInput.View())
	b.WriteString("\n\n")

	// Calculate available lines for project list precisely
	usedLines := 0
	usedLines++    // Title line
	usedLines++    // Separator
	usedLines++    // Empty line before search input
	usedLines++    // Search input
	usedLines += 2 // Empty lines after search input
	if m.showHelp {
		usedLines += 3 // Help text + spacing (bottom)
	}

	maxAvailableLines := m.height - usedLines - 7 // -7 for safety margin (ensure header stays visible)
	if maxAvailableLines < 1 {
		maxAvailableLines = 1
	}

	// Render projects, counting actual lines to stay within viewport
	renderedLines := 0
	start := 0

	// Simple scrolling: show items from start, make sure cursor is visible
	// Calculate how many items fit before cursor
	if m.cursor > 0 && len(m.filtered) > 0 && m.cursor < len(m.filtered) {
		// First, calculate how many lines the cursor item needs
		cursorItemLines := 1
		if m.filtered[m.cursor].Snippet != "" {
			cursorItemLines++
		}

		// Try to fit items before cursor
		lineCount := cursorItemLines // Start with cursor item
		itemsBeforeCursor := 0

		for i := m.cursor - 1; i >= 0; i-- {
			itemLines := 1
			if m.filtered[i].Snippet != "" {
				itemLines++
			}
			if lineCount+itemLines > maxAvailableLines {
				break
			}
			lineCount += itemLines
			itemsBeforeCursor++
		}

		start = m.cursor - itemsBeforeCursor
		if start < 0 {
			start = 0
		}
	}

	// Render visible projects, stopping when we run out of space
	for i := start; i < len(m.filtered); i++ {
		match := m.filtered[i]

		// Calculate how many lines this item will take
		itemLines := 1 // Base line for project name
		if match.Snippet != "" {
			itemLines++ // Add snippet line
		}

		// Check if we have room for this item
		if renderedLines+itemLines > maxAvailableLines {
			break // Stop rendering if we'd exceed available space
		}

		isExcluded := m.config != nil && m.config.IsExcluded(match.Project.Path)

		// Indicator (rendered separately to preserve its color)
		if i == m.cursor {
			// Selected item: orange indicator
			b.WriteString(m.styles.Cursor.Render("▌"))
		} else {
			// Normal item: space instead of indicator
			b.WriteString(" ")
		}

		// Render project name (with visual indicators and optional snippet)
		query := strings.TrimSpace(m.textInput.Value())
		isSelected := (i == m.cursor)
		projectContent := renderMatch(match, lipgloss.NewStyle(), m.styles.Highlight, m.styles.Snippet, query, isSelected, m.showScores)

		// Split content by lines to apply background to each line separately
		lines := strings.Split(projectContent, "\n")
		for lineIdx, line := range lines {
			if lineIdx > 0 {
				// For subsequent lines (snippets), add newline and spacing
				b.WriteString("\n ")
			}

			// Build full line with prefix
			var lineContent string
			if lineIdx == 0 {
				// First line: add space and optional excluded indicator
				if isExcluded && m.showExcluded {
					lineContent = " [✕] " + line
				} else {
					lineContent = " " + line
				}
			} else {
				// Snippet lines: add indentation (1 space margin + 4 spaces indent)
				lineContent = "     " + line
			}

			// Choose style and apply
			if i == m.cursor {
				// Apply background with width to fill the terminal
				styledLine := m.styles.Selected.Width(m.width - 2).Render(lineContent) // -2 for cursor + initial space
				b.WriteString(styledLine)
			} else if isExcluded && m.showExcluded {
				b.WriteString(m.styles.Excluded.Render(lineContent))
			} else {
				b.WriteString(m.styles.Normal.Render(lineContent))
			}
		}
		b.WriteString("\n")

		// Update line counter
		renderedLines += itemLines
	}

	// Help text footer (only show if toggled with ?)
	if m.showHelp {
		b.WriteString("\n\n")

		// Build help text with exclusion status
		var helpText string
		if m.showExcluded {
			helpText = "↑/↓: navigate • enter: select • ctrl+x: toggle exclusion • ctrl+h: hide excluded • ctrl+r: sync • ?: toggle help"
		} else {
			helpText = "↑/↓: navigate • enter: select • ctrl+x: exclude • ctrl+h: show excluded • ctrl+r: sync • ?: toggle help"
		}
		b.WriteString(m.styles.Help.Render(helpText))
	}

	return b.String()
}

// Selected returns the selected project (or empty string if none)
func (m Model) Selected() string {
	return m.selected
}

// truncateSnippet truncates text at word boundary respecting UTF-8
func truncateSnippet(text string, maxRunes int) string {
	runes := []rune(text)

	// If text fits - return as is
	if len(runes) <= maxRunes {
		return text
	}

	// Cut at maxRunes
	truncated := runes[:maxRunes]

	// Find last word boundary (space, comma, period, etc.)
	lastSpace := -1
	for i := len(truncated) - 1; i >= 0; i-- {
		if unicode.IsSpace(truncated[i]) || truncated[i] == ',' || truncated[i] == '.' || truncated[i] == ';' {
			lastSpace = i
			break
		}
	}

	// Use word boundary if found in last 20% to avoid losing too much text
	if lastSpace > int(float64(maxRunes)*0.8) {
		truncated = truncated[:lastSpace]
	}

	return string(truncated) + "..."
}

// formatCountWithBreakdown formats the count display with source breakdown
func formatCountWithBreakdown(matches []index.CombinedMatch, total int, countStyle lipgloss.Style, activeStyle lipgloss.Style) string {
	filtered := len(matches)

	// Count by source
	nameOnly := 0
	descriptionOnly := 0
	both := 0
	for _, m := range matches {
		if m.Source&index.MatchSourceName != 0 && m.Source&index.MatchSourceDescription != 0 {
			both++
		} else if m.Source&index.MatchSourceDescription != 0 {
			descriptionOnly++
		} else if m.Source&index.MatchSourceName != 0 {
			nameOnly++
		}
	}

	if filtered == total {
		return countStyle.Render(lipgloss.JoinHorizontal(lipgloss.Left,
			"",
			lipgloss.NewStyle().Bold(true).Inherit(countStyle).Render(formatNumber(total)),
			" projects"))
	}

	// Build breakdown if we have a query
	breakdown := ""
	if filtered < total && filtered > 0 {
		parts := []string{}
		if nameOnly > 0 {
			parts = append(parts, fmt.Sprintf("%d by name", nameOnly))
		}
		if descriptionOnly > 0 {
			parts = append(parts, fmt.Sprintf("%d by description", descriptionOnly))
		}
		if both > 0 {
			parts = append(parts, fmt.Sprintf("%d both", both))
		}
		if len(parts) > 0 {
			breakdown = " (" + strings.Join(parts, ", ") + ")"
		}
	}

	return countStyle.Render(lipgloss.JoinHorizontal(lipgloss.Left,
		"",
		activeStyle.Render(formatNumber(filtered)),
		"/",
		lipgloss.NewStyle().Bold(true).Inherit(countStyle).Render(formatNumber(total)),
		" projects",
		breakdown))
}

func formatNumber(n int) string {
	if n < 1000 {
		return fmt.Sprintf("%d", n)
	}
	return fmt.Sprintf("%d,%03d", n/1000, n%1000)
}

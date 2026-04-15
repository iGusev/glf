package tui

import (
	"fmt"
	"path/filepath"
	"strings"
	"time"
	"unicode"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/igusev/glf/internal/config"
	"github.com/igusev/glf/internal/history"
	"github.com/igusev/glf/internal/index"
	"github.com/igusev/glf/internal/model"
	"github.com/igusev/glf/internal/search"
)

// SyncStartMsg is sent when sync starts
type SyncStartMsg struct{}

// SyncCompleteMsg is sent when sync completes
type SyncCompleteMsg struct {
	Err      error
	Projects []model.Project
}

// HistoryLoadedMsg is sent when history finishes loading
type HistoryLoadedMsg struct {
	Err error
}

// debounceTickMsg is sent after a debounce delay to trigger filtering
type debounceTickMsg struct {
	version int
}

// indexReopenedMsg is sent when the index has been reopened after sync
type indexReopenedMsg struct {
	descIndex *index.DescriptionIndex
	err       error
}

// Model represents the TUI state
type Model struct {
	textInput      textinput.Model              // Search input field
	styles         Styles                       // Pre-configured styles
	projects       []model.Project              // All projects (full list)
	filtered       []index.CombinedMatch        // Filtered projects with match data (fuzzy + description)
	selected       string                       // Selected project path (when user presses Enter)
	cacheDir       string                       // Cache directory for description index
	gitlabURL      string                       // GitLab server URL (for header display)
	username       string                       // GitLab username (for header display)
	version        string                       // Application version
	syncError      error                        // Sync error if any
	history        *history.History              // Selection frequency tracker
	config         *config.Config               // Application config (for exclusions)
	colorScheme    *ColorScheme                 // Adaptive color scheme
	descIndex          *index.DescriptionIndex  // Persistent Bleve index (kept open during session)
	cachedEmptyResults []index.CombinedMatch   // Cached results for empty query (all projects sorted by history)
	onSync             func() tea.Cmd          // Callback to trigger sync
	cursor             int                     // Current cursor position in filtered list
	viewportStart      int                     // Index of first visible item in viewport
	width              int                     // Terminal width
	height             int                     // Terminal height
	filterVersion      int                     // Monotonic counter for keystroke debouncing
	emptyResultsCached bool                    // Whether cachedEmptyResults is valid
	quitting       bool                         // Whether user is quitting
	syncing        bool                         // Whether sync is in progress
	autoSync       bool                         // Whether to auto-sync on start
	historyLoading bool                         // Whether history is being loaded
	showHidden     bool                         // Whether to show hidden projects (excluded, archived, non-member)
	showScores     bool                         // Whether to show score breakdown
	showHelp       bool                         // Whether to show help text
}

// New creates a new TUI model with the given projects and optional initial query
func New(projects []model.Project, initialQuery string, onSync func() tea.Cmd, cacheDir string, cfg *config.Config, showScores bool, showHidden bool, username string, version string, descIndex *index.DescriptionIndex) Model {
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
		showHidden:     showHidden, // Initial state from CLI flag - controls visibility of excluded, archived, and non-member
		cacheDir:       cacheDir,
		showScores:     showScores, // Show score breakdown if requested
		colorScheme:    colorScheme,
		styles:         styles,
		gitlabURL:      gitlabURL,
		username:       username,
		version:        version,   // Injected from build-time ldflags
		descIndex:      descIndex, // Persistent index for fast search
		showHelp:       false,     // Hide help by default
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
				// Close index to allow sync exclusive access
				if m.descIndex != nil {
					_ = m.descIndex.Close()
					m.descIndex = nil
				}
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
				m.emptyResultsCached = false
				m.filter()
				if m.cursor >= len(m.filtered) && m.cursor > 0 {
					m.cursor = len(m.filtered) - 1
				}
				m.viewportStart = 0
			}

		case "ctrl+h":
			m.showHidden = !m.showHidden
			m.emptyResultsCached = false
			m.filter()
			// Reset cursor and viewport
			if m.cursor >= len(m.filtered) && m.cursor > 0 {
				m.cursor = len(m.filtered) - 1
			}
			m.viewportStart = 0

		case "?":
			// Toggle help text
			m.showHelp = !m.showHelp

		case "down", "ctrl+n":
			if m.cursor < len(m.filtered)-1 {
				m.cursor++
				// Adjust viewport if cursor scrolled below visible area
				// Calculate available lines for the viewport
				usedLines := 6 // Title, separator, empty, search, 2 empty
				if m.showHelp {
					usedLines += 3
				}
				maxAvailableLines := m.height - usedLines
				if maxAvailableLines < 1 {
					maxAvailableLines = 1
				}
				m.ensureCursorVisible(maxAvailableLines)
			}

		case "up", "ctrl+p":
			if m.cursor > 0 {
				m.cursor--
				// Adjust viewport if cursor scrolled above visible area
				if m.cursor < m.viewportStart {
					m.viewportStart = m.cursor
				}
			}

		default:
			// Update text input
			prevValue := m.textInput.Value()
			m.textInput, cmd = m.textInput.Update(msg)

			// Only debounce filter if text actually changed
			if m.textInput.Value() != prevValue {
				m.cursor = 0
				m.viewportStart = 0
				m.filterVersion++
				currentVersion := m.filterVersion
				tickCmd := tea.Tick(150*time.Millisecond, func(_ time.Time) tea.Msg {
					return debounceTickMsg{version: currentVersion}
				})
				cmd = tea.Batch(cmd, tickCmd)
			}
		}

	case autoSyncMsg:
		// Trigger background sync on startup
		if m.onSync != nil && !m.syncing {
			m.syncing = true
			m.syncError = nil
			// Close index to allow sync exclusive access
			if m.descIndex != nil {
				_ = m.descIndex.Close()
				m.descIndex = nil
			}
			return m, m.onSync()
		}

	case SyncCompleteMsg:
		m.syncing = false
		m.emptyResultsCached = false
		if msg.Err != nil {
			m.syncError = msg.Err
		} else {
			m.projects = msg.Projects
			m.syncError = nil
		}
		// Reopen index after sync (regardless of success/failure)
		cacheDir := m.cacheDir
		return m, func() tea.Msg {
			indexPath := filepath.Join(cacheDir, "description.bleve")
			di, _, err := index.NewDescriptionIndexWithAutoRecreate(indexPath)
			return indexReopenedMsg{descIndex: di, err: err}
		}

	case indexReopenedMsg:
		if msg.err == nil {
			m.descIndex = msg.descIndex
		}
		m.filter()

	case debounceTickMsg:
		if msg.version == m.filterVersion {
			m.filter()
		}

	case HistoryLoadedMsg:
		m.historyLoading = false
		m.emptyResultsCached = false
		if msg.Err != nil {
			// Log error but don't fail - history is optional
		} else {
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

	// For empty queries, use cached results if available
	if query == "" && m.emptyResultsCached {
		m.filtered = m.cachedEmptyResults
		return
	}

	var allMatches []index.CombinedMatch
	var err error
	if m.descIndex != nil {
		allMatches, err = search.CombinedSearchWithIndex(query, m.projects, historyScores, m.cacheDir, m.descIndex)
	} else if m.syncing {
		return
	} else {
		allMatches, err = search.CombinedSearch(query, m.projects, historyScores, m.cacheDir)
	}
	if err != nil {
		allMatches = []index.CombinedMatch{}
	}

	// Apply hidden projects filter if needed (unless showHidden is true)
	// Filter out: excluded, archived, and non-member projects
	filtered := allMatches
	if !m.showHidden {
		temp := make([]index.CombinedMatch, 0, len(filtered))
		for _, match := range filtered {
			// Skip if excluded by config
			if m.config != nil && m.config.IsExcluded(match.Project.Path) {
				continue
			}
			// Skip if archived
			if match.Project.Archived {
				continue
			}
			// Skip if non-member (Member field is false)
			if !match.Project.Member {
				continue
			}
			temp = append(temp, match)
		}
		filtered = temp
	}

	m.filtered = filtered

	if query == "" {
		m.cachedEmptyResults = filtered
		m.emptyResultsCached = true
	}
}

// ensureCursorVisible adjusts viewportStart if cursor is not visible in viewport
func (m *Model) ensureCursorVisible(maxAvailableLines int) {
	if len(m.filtered) == 0 {
		return
	}

	// If cursor is above viewport, move viewport up
	if m.cursor < m.viewportStart {
		m.viewportStart = m.cursor
		return
	}

	// Calculate how many items fit from viewportStart
	linesUsed := 0
	visibleItems := 0
	for i := m.viewportStart; i < len(m.filtered) && linesUsed < maxAvailableLines; i++ {
		itemLines := 1
		if m.filtered[i].Snippet != "" {
			itemLines = 2
		}
		if linesUsed+itemLines > maxAvailableLines {
			break
		}
		linesUsed += itemLines
		visibleItems++
	}

	// If cursor is beyond visible items, adjust viewport down
	if m.cursor >= m.viewportStart+visibleItems {
		// Move viewport so cursor is at bottom of screen
		m.viewportStart = m.cursor - visibleItems + 1
		if m.viewportStart < 0 {
			m.viewportStart = 0
		}
	}
}

// renderMatch renders a matched project with visual indicators and optional snippet
// Uses pre-computed styles from the Styles struct to avoid per-render allocations
func renderMatch(match index.CombinedMatch, s Styles, query string, showScores bool, isHidden bool) string {
	var result strings.Builder

	style := lipgloss.NewStyle()
	highlightStyle := s.Highlight
	snippetStyle := s.Snippet

	if match.Project.Starred {
		if isHidden {
			style = s.HiddenStarredText
			highlightStyle = s.HiddenStarredHighlight
			result.WriteString(s.HiddenStarredHeart.Render("❤ "))
		} else {
			style = s.StarredText
			highlightStyle = s.StarredHighlight
			result.WriteString(s.StarredHeart.Render("❤ "))
		}
	}

	displayStr := match.Project.DisplayString()

	if match.Source&index.MatchSourceName != 0 {
		result.WriteString(renderFuzzyMatch(displayStr, query, style, highlightStyle))
	} else {
		result.WriteString(style.Render(displayStr))
	}

	if showScores {
		var scoreStyle lipgloss.Style
		if match.Project.Starred {
			if isHidden {
				scoreStyle = s.HiddenStarredScore
			} else {
				scoreStyle = s.StarredScore
			}
		} else {
			scoreStyle = s.ScoreText
		}
		if match.StarredBonus > 0 {
			scoreText := fmt.Sprintf(" [S:%.3f H:%d St:%d T:%.2f]",
				match.SearchScore, match.HistoryScore, match.StarredBonus, match.TotalScore)
			result.WriteString(scoreStyle.Render(scoreText))
		} else {
			scoreText := fmt.Sprintf(" [S:%.3f H:%d T:%.2f]",
				match.SearchScore, match.HistoryScore, match.TotalScore)
			result.WriteString(scoreStyle.Render(scoreText))
		}
	}

	if match.Snippet != "" {
		snippet := truncateSnippet(match.Snippet, 60)
		result.WriteString("\n")

		if match.Project.Starred {
			if isHidden {
				snippetStyle = s.HiddenStarredSnippet
			} else {
				snippetStyle = s.StarredSnippet
			}
		} else if isHidden {
			snippetStyle = s.HiddenSnippet
		}
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

	maxAvailableLines := m.height - usedLines // No safety margin - maximize list space
	if maxAvailableLines < 1 {
		maxAvailableLines = 1
	}

	// Render projects, counting actual lines to stay within viewport
	renderedLines := 0

	// Use viewportStart for scrolling - no recalculation needed
	// viewportStart is maintained in Update() when cursor moves
	start := m.viewportStart

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
		isArchived := match.Project.Archived
		isNonMember := !match.Project.Member
		isHidden := isExcluded || isArchived || isNonMember // Any type of hidden project

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
		projectContent := renderMatch(match, m.styles, query, m.showScores, isHidden)

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
				// First line: add space and optional hidden project indicators
				prefix := " "
				if m.showHidden {
					// Show visual indicators for different types of hidden projects
					if isExcluded {
						prefix += "[✕] " // Excluded by user (config)
					} else if isArchived {
						prefix += "[A] " // Archived
					} else if isNonMember {
						prefix += "[G] " // Non-member (guest - visible but not a member)
					}
				}
				lineContent = prefix + line
			} else {
				// Snippet lines: add indentation (1 space margin + 4 spaces indent)
				lineContent = "     " + line
			}

			// Choose style and apply
			if i == m.cursor {
				// Apply background with width to fill the terminal
				styledLine := m.styles.Selected.Width(m.width - 2).Render(lineContent) // -2 for cursor + initial space
				b.WriteString(styledLine)
			} else if isHidden && m.showHidden {
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

		// Build help text with hidden projects status
		var helpText string
		if m.showHidden {
			helpText = "↑/↓: navigate • enter: select • ctrl+x: toggle exclusion • ctrl+h: hide hidden (✕=excluded A=archived G=guest) • ctrl+r: sync • ?: toggle help"
		} else {
			helpText = "↑/↓: navigate • enter: select • ctrl+x: exclude • ctrl+h: show hidden • ctrl+r: sync • ?: toggle help"
		}
		b.WriteString(m.styles.Help.Render(helpText))
	}

	return b.String()
}

// Selected returns the selected project (or empty string if none)
func (m Model) Selected() string {
	return m.selected
}

// CloseIndex closes the persistent Bleve index if it is open
func (m Model) CloseIndex() {
	if m.descIndex != nil {
		_ = m.descIndex.Close()
	}
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

package tui

import (
	"fmt"

	"github.com/charmbracelet/lipgloss"
)

// ColorScheme holds all adaptive color definitions for the TUI
type ColorScheme struct {
	// Title and branding
	Title      lipgloss.AdaptiveColor
	GitLabWave string // Pre-rendered gradient wave
	Version    lipgloss.AdaptiveColor
	ServerInfo lipgloss.AdaptiveColor

	// Input prompt
	Prompt lipgloss.AdaptiveColor

	// Project list
	Normal     lipgloss.AdaptiveColor
	Selected   lipgloss.AdaptiveColor
	SelectedBg lipgloss.AdaptiveColor
	Highlight  lipgloss.AdaptiveColor // For fuzzy match highlighting
	Snippet    lipgloss.AdaptiveColor

	// Status and counts
	Count       lipgloss.AdaptiveColor
	CountActive lipgloss.AdaptiveColor // Active/filtered count

	// Indicators
	Cursor   lipgloss.AdaptiveColor
	Excluded lipgloss.AdaptiveColor

	// Status indicators
	StatusActive lipgloss.AdaptiveColor // Green for loading/syncing
	StatusError  lipgloss.AdaptiveColor // Red for errors
	StatusIdle   lipgloss.AdaptiveColor // Gray for idle

	// Help text
	Help lipgloss.AdaptiveColor
}

// NewColorScheme creates a new color scheme with adaptive colors for terminal theme
func NewColorScheme() *ColorScheme {
	return &ColorScheme{
		// Title: bright cyan for dark, darker blue for light
		Title: lipgloss.AdaptiveColor{
			Light: "#0066CC", // Darker blue for light backgrounds
			Dark:  "#5AD4E6", // Bright cyan for dark backgrounds
		},

		// GitLab gradient wave (generated once)
		GitLabWave: renderGitLabWave(),

		// Version info: muted for both
		Version: lipgloss.AdaptiveColor{
			Light: "#666666",
			Dark:  "#6967A3",
		},

		// Server info: same as version
		ServerInfo: lipgloss.AdaptiveColor{
			Light: "#666666",
			Dark:  "#6967A3",
		},

		// Prompt: GitLab orange (works on both)
		Prompt: lipgloss.AdaptiveColor{
			Light: "#FC6D25", // GitLab orange
			Dark:  "#FC9867", // Slightly lighter orange
		},

		// Normal text: dark on light, light on dark
		Normal: lipgloss.AdaptiveColor{
			Light: "#1A1A1A", // Almost black for light backgrounds
			Dark:  "#F7F1FF", // Off-white for dark backgrounds
		},

		// Selected item text: ensure contrast
		Selected: lipgloss.AdaptiveColor{
			Light: "#000000", // Black text on light selection
			Dark:  "#E4E4E4", // Light gray text on dark selection
		},

		// Selected item background
		SelectedBg: lipgloss.AdaptiveColor{
			Light: "#E0E0E0", // Light gray for light theme
			Dark:  "#303030", // Dark gray for dark theme
		},

		// Fuzzy match highlight: yellow/orange
		Highlight: lipgloss.AdaptiveColor{
			Light: "#D97706", // Orange for light backgrounds
			Dark:  "#FCE566", // Yellow for dark backgrounds
		},

		// Snippet text: muted gray italic
		Snippet: lipgloss.AdaptiveColor{
			Light: "#737373",
			Dark:  "#999999",
		},

		// Count: muted
		Count: lipgloss.AdaptiveColor{
			Light: "#666666",
			Dark:  "#6967A3",
		},

		// Active count: highlighted yellow
		CountActive: lipgloss.AdaptiveColor{
			Light: "#D97706", // Orange
			Dark:  "#FCE566", // Yellow
		},

		// Cursor indicator: orange-red (GitLab accent)
		Cursor: lipgloss.AdaptiveColor{
			Light: "#E6704E", // Orange-red for visibility
			Dark:  "#E6704E", // Orange-red for visibility
		},

		// Excluded projects: very muted
		Excluded: lipgloss.AdaptiveColor{
			Light: "#A3A3A3",
			Dark:  "#5A5A5A",
		},

		// Status active: green
		StatusActive: lipgloss.AdaptiveColor{
			Light: "#16A34A", // Darker green
			Dark:  "#7BD88F", // Bright green
		},

		// Status error: red/pink
		StatusError: lipgloss.AdaptiveColor{
			Light: "#DC2626", // Darker red
			Dark:  "#FC618D", // Bright pink
		},

		// Status idle: gray
		StatusIdle: lipgloss.AdaptiveColor{
			Light: "#737373",
			Dark:  "#666666",
		},

		// Help text: muted
		Help: lipgloss.AdaptiveColor{
			Light: "#737373",
			Dark:  "#666666",
		},
	}
}

// renderGitLabWave creates the GitLab gradient wave █▓▒░
// Colors: #E24328 (0%) → #FC6D25 (50%) → #FDA326 (100%)
func renderGitLabWave() string {
	// Define gradient stops (GitLab brand colors)
	stops := []struct {
		position float64
		color    [3]int // RGB
	}{
		{0.0, [3]int{0xE2, 0x43, 0x28}}, // #E24328 - GitLab Red
		{0.5, [3]int{0xFC, 0x6D, 0x25}}, // #FC6D25 - GitLab Orange
		{1.0, [3]int{0xFD, 0xA3, 0x26}}, // #FDA326 - GitLab Yellow
	}

	// Characters for wave (from darkest to lightest)
	chars := []string{"█", "▓", "▒", "░"}

	// Calculate colors for each character position
	var result string
	for i, char := range chars {
		// Position in gradient (0.0 to 1.0)
		position := float64(i) / float64(len(chars)-1)

		// Find the two stops to interpolate between
		var start, end int
		for j := 0; j < len(stops)-1; j++ {
			if position >= stops[j].position && position <= stops[j+1].position {
				start = j
				end = j + 1
				break
			}
		}

		// Calculate local position between the two stops
		localPos := (position - stops[start].position) / (stops[end].position - stops[start].position)

		// Interpolate RGB values
		r := int(float64(stops[start].color[0]) + float64(stops[end].color[0]-stops[start].color[0])*localPos)
		g := int(float64(stops[start].color[1]) + float64(stops[end].color[1]-stops[start].color[1])*localPos)
		b := int(float64(stops[start].color[2]) + float64(stops[end].color[2]-stops[start].color[2])*localPos)

		// Create color and apply to character
		color := lipgloss.Color(fmt.Sprintf("#%02X%02X%02X", r, g, b))
		style := lipgloss.NewStyle().Foreground(color)
		result += style.Render(char)
	}

	return result
}

// GetStyles returns pre-configured lipgloss styles using the color scheme
func (cs *ColorScheme) GetStyles() Styles {
	return Styles{
		Title: lipgloss.NewStyle().
			Bold(true).
			Foreground(cs.Title),

		Version: lipgloss.NewStyle().
			Foreground(cs.Version),

		ServerInfo: lipgloss.NewStyle().
			Foreground(cs.ServerInfo),

		Prompt: lipgloss.NewStyle().
			Foreground(cs.Prompt),

		Normal: lipgloss.NewStyle().
			Foreground(cs.Normal),

		Selected: lipgloss.NewStyle().
			Foreground(cs.Selected).
			Background(cs.SelectedBg).
			Bold(false),

		Highlight: lipgloss.NewStyle().
			Foreground(cs.Highlight).
			Bold(true),

		Snippet: lipgloss.NewStyle().
			Foreground(cs.Snippet).
			Italic(true),

		Count: lipgloss.NewStyle().
			Foreground(cs.Count),

		CountActive: lipgloss.NewStyle().
			Bold(true).
			Foreground(cs.CountActive),

		Cursor: lipgloss.NewStyle().
			Foreground(cs.Cursor).
			Bold(true),

		Excluded: lipgloss.NewStyle().
			Foreground(cs.Excluded),

		StatusActive: lipgloss.NewStyle().
			Foreground(cs.StatusActive),

		StatusError: lipgloss.NewStyle().
			Foreground(cs.StatusError),

		StatusIdle: lipgloss.NewStyle().
			Foreground(cs.StatusIdle),

		Help: lipgloss.NewStyle().
			Foreground(cs.Help),
	}
}

// Styles holds pre-configured lipgloss styles
type Styles struct {
	Title        lipgloss.Style
	Version      lipgloss.Style
	ServerInfo   lipgloss.Style
	Prompt       lipgloss.Style
	Normal       lipgloss.Style
	Selected     lipgloss.Style
	Highlight    lipgloss.Style
	Snippet      lipgloss.Style
	Count        lipgloss.Style
	CountActive  lipgloss.Style
	Cursor       lipgloss.Style
	Excluded     lipgloss.Style
	StatusActive lipgloss.Style
	StatusError  lipgloss.Style
	StatusIdle   lipgloss.Style
	Help         lipgloss.Style
}

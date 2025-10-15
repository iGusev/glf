package main

import (
	"fmt"

	"github.com/charmbracelet/lipgloss"
)

// GitLab brand colors
var (
	// GitLab orange: #FC6D26
	gitlabOrange = lipgloss.Color("#FC6D26")
	// GitLab dark: #2E2E2E
	gitlabDark = lipgloss.Color("#2E2E2E")
	// Success green
	successGreen = lipgloss.Color("#00C853")
	// Warning yellow
	warningYellow = lipgloss.Color("#FFC107")
	// Error red
	errorRed = lipgloss.Color("#F44336")
	// Info blue
	infoBlue = lipgloss.Color("#2196F3")
	// Muted gray
	mutedGray = lipgloss.Color("#9E9E9E")
)

// Style definitions
var (
	// Logo style - bold orange
	logoStyle = lipgloss.NewStyle().
			Foreground(gitlabOrange).
			Bold(true)

	// Title style - bold with orange accent
	titleStyle = lipgloss.NewStyle().
			Foreground(gitlabOrange).
			Bold(true)

	// Section header style
	sectionStyle = lipgloss.NewStyle().
			Foreground(gitlabOrange).
			Bold(true)

	// Success style
	successStyle = lipgloss.NewStyle().
			Foreground(successGreen).
			Bold(true)

	// Warning style
	warningStyle = lipgloss.NewStyle().
			Foreground(warningYellow).
			Bold(true)

	// Error style
	errorStyle = lipgloss.NewStyle().
			Foreground(errorRed).
			Bold(true)

	// Muted text style
	mutedStyle = lipgloss.NewStyle().
			Foreground(mutedGray)

	// Input prompt style
	promptStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#E6704E"))

	// Example text style
	exampleStyle = lipgloss.NewStyle().
			Foreground(mutedGray).
			Italic(true)

	// URL style
	urlStyle = lipgloss.NewStyle().
			Foreground(infoBlue)

	// Highlight style
	highlightStyle = lipgloss.NewStyle().
			Foreground(gitlabOrange).
			Bold(true)
)

// printLogo prints the styled GLF logo with version
func printLogo(ver string) {
	// Gradient blocks █▓▒░
	gradient := lipgloss.NewStyle().Foreground(gitlabOrange).Render("█▓▒░")
	title := lipgloss.NewStyle().Foreground(gitlabOrange).Bold(true).Render("glf")
	versionText := lipgloss.NewStyle().Foreground(mutedGray).Render(ver)

	fmt.Printf("%s %s %s\n", gradient, title, versionText)
	fmt.Println(mutedStyle.Render("GitLab Fuzzy Finder"))
	fmt.Println()
}

// printTitle prints a styled section title with separator
func printTitle(text string) {
	fmt.Println(titleStyle.Render(text))
	fmt.Println()
}

// printSection prints a styled section header
func printSection(emoji, text string) {
	fmt.Println(sectionStyle.Render(emoji + " " + text))
}

// printSuccess prints a success message
func printSuccess(text string) {
	fmt.Println(successStyle.Render("✓ " + text))
}

// printWarning prints a warning message
func printWarning(text string) {
	fmt.Println(warningStyle.Render("⚠️  " + text))
}

// printError prints an error message
func printError(text string) {
	fmt.Println(errorStyle.Render("❌ " + text))
}

// printMuted prints muted text
func printMuted(text string) {
	fmt.Println(mutedStyle.Render(text))
}

// printExample prints example text
func printExample(text string) {
	fmt.Println(exampleStyle.Render(text))
}

// printURL prints a styled URL
func printURL(url string) {
	fmt.Println(urlStyle.Render(url))
}

// printPrompt prints an input prompt on same line
func printPrompt(text string) {
	fmt.Print(promptStyle.Render(text))
}

// printHighlight prints highlighted text
func printHighlight(text string) {
	fmt.Print(highlightStyle.Render(text))
}

// printSeparator prints a visual separator
func printSeparator() {
	fmt.Println(mutedStyle.Render("─────────────────────────────────────────"))
}

// printBullet prints a bullet point
func printBullet(text string) {
	fmt.Println("• " + text)
}

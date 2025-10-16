package main

import (
	"fmt"

	"github.com/charmbracelet/lipgloss"
)

// GitLab brand colors
var (
	// GitLab orange: #FC6D26
	gitlabOrange = lipgloss.Color("#FC6D26")
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

// printBullet prints a bullet point
func printBullet(text string) {
	fmt.Println("• " + text)
}

// showColorPalette displays example starred project lines with various gold color shades
// This helps visualize and choose the best gold color for starred projects
func showColorPalette() {
	// Gold color palette from user's image (100+ shades of gold)
	goldColors := []struct {
		name string
		hex  string
	}{
		{"GOLD", "#FFD700"},
		{"METALLIC GOLD", "#D4AF37"},
		{"GOLDEN YELLOW", "#FFDF00"},
		{"BURNT SIENNA", "#E97451"},
		{"GOLDEN BROWN", "#996515"},
		{"LIGHT GOLD", "#FDDC5C"},
		{"AMBER", "#FFBF00"},
		{"OLD GOLD", "#CFB53B"},
		{"ROSE GOLD", "#B76E79"},
		{"GOLDEN", "#F5BF03"},
		{"CALIFORNIA GOLD", "#FDB515"},
		{"SUNGLOW", "#FFCC33"},
		{"GOLDEN PUPPY", "#FCC200"},
		{"HARVEST GOLD", "#DA9100"},
		{"PALE GOLD", "#E6BE8A"},
		{"HONEY GOLD", "#FFC30B"},
		{"VEGAS GOLD", "#C5B358"},
		{"CYBER YELLOW", "#FFD300"},
		{"MUSTARD", "#FFDB58"},
		{"SANDY BROWN", "#F4A460"},
		{"GOLDENROD", "#DAA520"},
		{"DARK GOLDENROD", "#B8860B"},
		{"PERU", "#CD853F"},
		{"CHOCOLATE", "#D2691E"},
		{"SADDLE BROWN", "#8B4513"},
		{"SIENNA", "#A0522D"},
		{"BRONZE", "#CD7F32"},
		{"TAN", "#D2B48C"},
		{"KHAKI", "#F0E68C"},
		{"DARK KHAKI", "#BDB76B"},
		{"OLIVE", "#808000"},
		{"BRASS", "#B5A642"},
		{"SATIN GOLD", "#CDA434"},
		{"PALE GOLDENROD", "#EEE8AA"},
		{"CHAMPAGNE", "#F7E7CE"},
		{"BEIGE GOLD", "#E1C16E"},
		{"BLONDE", "#FAF0BE"},
		{"SUNSET GOLD", "#FDBE02"},
		{"MACARONI CHEESE", "#FFB97B"},
		{"GOLD FUSION", "#85754E"},
		{"PALE BROWN", "#987654"},
		{"CAMEL", "#C19A6B"},
		{"DESERT SAND", "#EDC9AF"},
		{"SAND", "#C2B280"},
		{"EARTH YELLOW", "#E1A95F"},
		{"AZTEC GOLD", "#C39953"},
		{"GOLD SAND", "#E6C86E"},
		{"MEAT BROWN", "#E5B73B"},
		{"UNIVERSITY OF CALIFORNIA", "#FDB515"},
		{"SELECTIVE YELLOW", "#FFBA00"},
		{"MIKADO YELLOW", "#FFC40C"},
		{"GOLDEN POPPY", "#FCC200"},
		{"SCHOOL BUS YELLOW", "#FFD800"},
		{"CHROME YELLOW", "#FFA700"},
		{"ORANGE YELLOW", "#FFAE42"},
		{"RAJAH", "#FBAB60"},
		{"MARIGOLD", "#EAA221"},
		{"BRIGHT SUN", "#FED33C"},
		{"GOLDEN TAINOI", "#FFC152"},
		{"TULIP TREE", "#E9B824"},
		{"ROB ROY", "#EAC674"},
		{"CONFETTI", "#E9D75A"},
		{"GOLDEN DREAM", "#F0D52D"},
		{"GOLD TIPS", "#DEBA13"},
		{"RONCHI", "#ECC54E"},
		{"OLD LACE", "#FDF5E6"},
		{"WITCH HAZE", "#FFFC99"},
		{"BUTTERMILK", "#FFF1B5"},
		{"PINK LACE", "#FFDDF4"},
		{"ANTIQUE BRASS", "#CD9575"},
		{"DESERT", "#C19A6B"},
		{"PALE TAUPE", "#BC987E"},
		{"TAN HIDE", "#FA9D5A"},
		{"FAWN", "#E5AA70"},
		{"WOOD BROWN", "#C19A6B"},
		{"FALL LEAF BROWN", "#C8B560"},
		{"SANDY TAN", "#FDD9B5"},
		{"SEPIA", "#704214"},
		{"SIENNA OCHRE", "#C88141"},
		{"MOCCASIN", "#FFE4B5"},
		{"NAVAJO WHITE", "#FFDEAD"},
		{"PEACH PUFF", "#FFDAB9"},
		{"BISQUE", "#FFE4C4"},
		{"BLANCHED ALMOND", "#FFEBCD"},
		{"PAPAYA WHIP", "#FFEFD5"},
		{"ANTIQUE WHITE", "#FAEBD7"},
		{"LINEN", "#FAF0E6"},
		{"OLD GOLD PAPER", "#EED9C4"},
		{"FLAX", "#EEDC82"},
		{"JASMINE", "#F8DE7E"},
		{"LEMON CHIFFON", "#FFFACD"},
		{"CORNSILK", "#FFF8DC"},
		{"CREAM", "#FFFDD0"},
		{"COSMIC LATTE", "#FFF8E7"},
		{"EGGSHELL", "#F0EAD6"},
		{"IVORY", "#FFFFF0"},
		{"PEARL", "#EAE0C8"},
		{"SEASHELL", "#FFF5EE"},
		{"VANILLA", "#F3E5AB"},
		{"WHEAT", "#F5DEB3"},
	}

	// Print header
	fmt.Println()
	fmt.Println(titleStyle.Render("🌟 Gold Color Palette Test"))
	fmt.Println()
	fmt.Println(mutedStyle.Render("Example starred project lines with different gold shades:"))
	fmt.Println(mutedStyle.Render("Currently using: CALIFORNIA GOLD #FDB515 for starred projects"))
	fmt.Println()

	// Display each color with example
	exampleProject := "❤ backend/api-server"

	for i, color := range goldColors {
		// Create style with this gold color
		starStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(color.hex))

		// Format color info (name + hex)
		colorInfo := lipgloss.NewStyle().
			Foreground(mutedGray).
			Width(35).
			Render(fmt.Sprintf("%d. %-20s %s", i+1, color.name, color.hex))

		// Render example with this color
		example := starStyle.Render(exampleProject)

		// Print line: "1. COLOR NAME          #HEXCODE    ★ backend/api-server"
		fmt.Printf("%s  %s\n", colorInfo, example)
	}

	fmt.Println()
	fmt.Println(mutedStyle.Render("Choose your favorite color and update the code in internal/tui/model.go"))
	fmt.Println()
}

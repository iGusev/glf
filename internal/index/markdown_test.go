package index

import (
	"strings"
	"testing"
)

func TestCleanMarkdown_Headings(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "h1 heading",
			input:    "# Main Title",
			expected: "Main Title",
		},
		{
			name:     "h2 heading",
			input:    "## Section",
			expected: "Section",
		},
		{
			name:     "multiple headings",
			input:    "# Title\n\n## Subtitle\n\nContent",
			expected: "Title\n\nSubtitle\nContent",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := CleanMarkdown(tt.input)
			if result != tt.expected {
				t.Errorf("CleanMarkdown() = %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestCleanMarkdown_Paragraphs(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "single paragraph",
			input:    "This is a paragraph.",
			expected: "This is a paragraph.",
		},
		{
			name:     "multiple paragraphs",
			input:    "First paragraph.\n\nSecond paragraph.",
			expected: "First paragraph.\nSecond paragraph.",
		},
		{
			name:     "paragraphs with extra newlines",
			input:    "First.\n\n\n\nSecond.",
			expected: "First.\nSecond.",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := CleanMarkdown(tt.input)
			if result != tt.expected {
				t.Errorf("CleanMarkdown() = %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestCleanMarkdown_CodeBlocks(t *testing.T) {
	// Test that code blocks are removed
	input := "# Title\n\nText before code\n\n```go\ncode here\n```\n\nText after code"
	result := CleanMarkdown(input)

	// Should contain title and text but not code
	if !strings.Contains(result, "Title") {
		t.Error("Result should contain Title")
	}
	if strings.Contains(result, "code here") {
		t.Error("Result should not contain code block content")
	}
}

func TestCleanMarkdown_InlineCode(t *testing.T) {
	// Test that inline code is removed
	input := "# Title\n\nUse `git commit` to save changes"
	result := CleanMarkdown(input)

	// Should contain title and text but not inline code
	if !strings.Contains(result, "Title") {
		t.Error("Result should contain Title")
	}
	if strings.Contains(result, "git commit") {
		t.Error("Result should not contain inline code content")
	}
	if !strings.Contains(result, "Use") {
		t.Error("Result should preserve text around inline code")
	}
}

func TestCleanMarkdown_Links(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "link text preserved",
			input:    "Check [documentation](https://example.com) for details",
			expected: "Check documentation  for details",
		},
		{
			name:     "multiple links",
			input:    "[First](url1) and [Second](url2) links",
			expected: "First  and Second  links",
		},
		{
			name:     "link with title",
			input:    "[Link](url \"title\")",
			expected: "Link",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := CleanMarkdown(tt.input)
			if result != tt.expected {
				t.Errorf("CleanMarkdown() = %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestCleanMarkdown_Images(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "image removed",
			input:    "Text before ![alt text](image.png) text after",
			expected: "Text before  text after",
		},
		{
			name:     "multiple images",
			input:    "![img1](1.png) middle ![img2](2.png)",
			expected: "middle",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := CleanMarkdown(tt.input)
			if result != tt.expected {
				t.Errorf("CleanMarkdown() = %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestCleanMarkdown_Lists(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "unordered list",
			input:    "- Item 1\n- Item 2\n- Item 3",
			expected: "• Item 1\n\n• Item 2\n\n• Item 3",
		},
		{
			name:     "ordered list",
			input:    "1. First\n2. Second\n3. Third",
			expected: "• First\n\n• Second\n\n• Third",
		},
		{
			name:     "list with text",
			input:    "List:\n\n- Item 1\n- Item 2",
			expected: "List:\n\n• Item 1\n\n• Item 2",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := CleanMarkdown(tt.input)
			if result != tt.expected {
				t.Errorf("CleanMarkdown() = %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestCleanMarkdown_Emphasis(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "italic removed",
			input:    "This is *italic* text",
			expected: "This is italic text",
		},
		{
			name:     "bold removed",
			input:    "This is **bold** text",
			expected: "This is bold text",
		},
		{
			name:     "bold italic removed",
			input:    "This is ***bold italic*** text",
			expected: "This is bold italic text",
		},
		{
			name:     "underscore emphasis",
			input:    "This is _italic_ and __bold__",
			expected: "This is italic and bold",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := CleanMarkdown(tt.input)
			if result != tt.expected {
				t.Errorf("CleanMarkdown() = %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestCleanMarkdown_HTML(t *testing.T) {
	// Test that HTML is removed
	input := "# Title\n\nText with <div>HTML</div> content"
	result := CleanMarkdown(input)

	// Should contain title and text but not HTML
	if !strings.Contains(result, "Title") {
		t.Error("Result should contain Title")
	}
	if strings.Contains(result, "<div>") || strings.Contains(result, "</div>") {
		t.Error("Result should not contain HTML tags")
	}
}

func TestCleanMarkdown_Complex(t *testing.T) {
	// Test real-world markdown document with multiple formatting types
	input := `# Project Title

This is a **comprehensive** project for [GitLab](https://gitlab.com) integration.

## Features

- Feature 1 with *emphasis*
- Feature 2
- Feature 3`

	result := CleanMarkdown(input)

	// Verify key elements are preserved
	if !strings.Contains(result, "Project Title") {
		t.Error("Should contain main title")
	}
	if !strings.Contains(result, "comprehensive") {
		t.Error("Should contain text with removed bold formatting")
	}
	if !strings.Contains(result, "GitLab") {
		t.Error("Should contain link text")
	}
	if !strings.Contains(result, "Features") {
		t.Error("Should contain section heading")
	}
	if !strings.Contains(result, "• Feature") {
		t.Error("Should contain list items with bullets")
	}
	if !strings.Contains(result, "emphasis") {
		t.Error("Should contain text with removed italic formatting")
	}

	// Verify formatting is removed
	if strings.Contains(result, "**") || strings.Contains(result, "*") {
		t.Error("Should not contain markdown formatting characters")
	}
	if strings.Contains(result, "[") || strings.Contains(result, "]") || strings.Contains(result, "(http") {
		t.Error("Should not contain link syntax")
	}
}

func TestCleanMarkdown_EmptyInput(t *testing.T) {
	result := CleanMarkdown("")
	if result != "" {
		t.Errorf("CleanMarkdown(\"\") = %q, want \"\"", result)
	}
}

func TestCleanMarkdown_WhitespaceOnly(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{"spaces", "   "},
		{"newlines", "\n\n\n"},
		{"tabs", "\t\t\t"},
		{"mixed", " \n \t \n "},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := CleanMarkdown(tt.input)
			if result != "" {
				t.Errorf("CleanMarkdown(%q) = %q, want \"\"", tt.input, result)
			}
		})
	}
}

func TestCleanMarkdown_NewlineNormalization(t *testing.T) {
	// Test that multiple consecutive newlines are normalized to max 2
	input := "Line 1\n\n\n\n\nLine 2\n\n\nLine 3"
	result := CleanMarkdown(input)

	// Count max consecutive newlines
	parts := strings.Split(result, "Line")
	for i, part := range parts {
		if i == 0 || i == len(parts)-1 {
			continue
		}
		newlineCount := strings.Count(part, "\n")
		if newlineCount > 2 {
			t.Errorf("Found %d consecutive newlines, want max 2 in: %q", newlineCount, part)
		}
	}
}

func TestCleanMarkdown_MixedFormatting(t *testing.T) {
	// Test combination of multiple formatting types
	input := "**Bold** with *italic* and [link](url) and `code`"
	expected := "Bold with italic and link  and"

	result := CleanMarkdown(input)
	if result != expected {
		t.Errorf("CleanMarkdown() = %q, want %q", result, expected)
	}
}

func TestCleanMarkdown_CyrillicText(t *testing.T) {
	// Test that Cyrillic characters are preserved
	input := "# Заголовок\n\nТекст с **выделением** и [ссылкой](url)."
	expected := "Заголовок\nТекст с выделением и ссылкой ."

	result := CleanMarkdown(input)
	if result != expected {
		t.Errorf("CleanMarkdown() = %q, want %q", result, expected)
	}
}

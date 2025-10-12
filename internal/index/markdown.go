package index

import (
	"bytes"
	"strings"

	"github.com/gomarkdown/markdown"
	"github.com/gomarkdown/markdown/ast"
)

// CleanMarkdown removes Markdown formatting and extracts plain text
// Preserves headings, paragraphs, and lists but removes code blocks, links syntax, etc.
func CleanMarkdown(md string) string {
	// Parse markdown to AST
	doc := markdown.Parse([]byte(md), nil)

	var buf bytes.Buffer
	visitor := &textExtractor{buf: &buf}
	ast.Walk(doc, visitor)

	// Clean up extra whitespace
	text := buf.String()
	text = strings.TrimSpace(text)

	// Normalize multiple newlines to double newline
	lines := strings.Split(text, "\n")
	var cleaned []string
	prevEmpty := false

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			if !prevEmpty {
				cleaned = append(cleaned, "")
				prevEmpty = true
			}
		} else {
			cleaned = append(cleaned, line)
			prevEmpty = false
		}
	}

	return strings.Join(cleaned, "\n")
}

// textExtractor is an AST visitor that extracts plain text
type textExtractor struct {
	buf          *bytes.Buffer
	inCodeBlock  bool
	inLink       bool
	skipChildren bool
}

// Visit implements ast.NodeVisitor interface
func (te *textExtractor) Visit(node ast.Node, entering bool) ast.WalkStatus {
	if te.skipChildren && entering {
		return ast.SkipChildren
	}
	te.skipChildren = false

	switch n := node.(type) {
	case *ast.Heading:
		if entering {
			te.buf.WriteString("\n")
		} else {
			te.buf.WriteString("\n")
		}

	case *ast.Paragraph:
		if !entering {
			te.buf.WriteString("\n")
		}

	case *ast.Text:
		if entering && !te.inCodeBlock {
			te.buf.Write(n.Literal)
		}

	case *ast.Softbreak, *ast.Hardbreak:
		if !te.inCodeBlock {
			te.buf.WriteString(" ")
		}

	case *ast.CodeBlock:
		// Skip code blocks entirely
		te.inCodeBlock = entering
		if entering {
			te.skipChildren = true
		}
		return ast.SkipChildren

	case *ast.Code:
		// Skip inline code
		if entering {
			te.skipChildren = true
		}
		return ast.SkipChildren

	case *ast.Link:
		te.inLink = entering
		// Extract link text but not URL
		if !entering {
			te.buf.WriteString(" ")
		}

	case *ast.Image:
		// Skip images entirely
		if entering {
			te.skipChildren = true
		}
		return ast.SkipChildren

	case *ast.List:
		if !entering {
			te.buf.WriteString("\n")
		}

	case *ast.ListItem:
		if entering {
			te.buf.WriteString("\n• ")
		}

	case *ast.Emph, *ast.Strong:
		// Keep emphasized/strong text but remove formatting

	case *ast.HTMLBlock, *ast.HTMLSpan:
		// Skip HTML
		if entering {
			te.skipChildren = true
		}
		return ast.SkipChildren
	}

	return ast.GoToNext
}

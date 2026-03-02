package replay

import (
	"strings"

	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/glamour/ansi"
	"github.com/charmbracelet/glamour/styles"
)

func boolPtr(b bool) *bool    { return &b }
func stringPtr(s string) *string { return &s }
func uintPtr(u uint) *uint    { return &u }

var mdRenderer *glamour.TermRenderer

func init() {
	// Start from dark style and strip it down to match Claude Code's
	// minimal markdown rendering: bold-only headers, dash bullets,
	// inline code with color only (no background), minimal margins.
	style := styles.DarkStyleConfig

	// Document: no extra margin, keep text color
	style.Document = ansi.StyleBlock{
		StylePrimitive: ansi.StylePrimitive{
			BlockPrefix: "\n",
			BlockSuffix: "\n",
			Color:       stringPtr("252"),
		},
		Margin: uintPtr(0),
	}

	// Headings: bold only, no color, no prefix markers
	style.Heading = ansi.StyleBlock{
		StylePrimitive: ansi.StylePrimitive{
			BlockSuffix: "\n",
			Bold:        boolPtr(true),
		},
	}
	style.H1 = ansi.StyleBlock{
		StylePrimitive: ansi.StylePrimitive{
			Bold: boolPtr(true),
		},
	}
	style.H2 = ansi.StyleBlock{
		StylePrimitive: ansi.StylePrimitive{},
	}
	style.H3 = ansi.StyleBlock{
		StylePrimitive: ansi.StylePrimitive{},
	}
	style.H4 = ansi.StyleBlock{
		StylePrimitive: ansi.StylePrimitive{},
	}
	style.H5 = ansi.StyleBlock{
		StylePrimitive: ansi.StylePrimitive{},
	}
	style.H6 = ansi.StyleBlock{
		StylePrimitive: ansi.StylePrimitive{},
	}

	// List items: use "-" instead of "•"
	style.Item = ansi.StylePrimitive{
		BlockPrefix: "- ",
	}

	// Inline code: soft blue-lavender, no background (matches Claude Code)
	purple := "#A9B1D6"
	style.Code = ansi.StyleBlock{
		StylePrimitive: ansi.StylePrimitive{
			Color: &purple,
		},
	}

	// Code blocks: no extra margin
	style.CodeBlock.Margin = uintPtr(0)

	// Bold/strong text: just bold, no special color (matches Claude Code)
	style.Strong = ansi.StylePrimitive{
		Bold: boolPtr(true),
	}

	// Paragraph: no extra block prefix/suffix beyond what document provides
	style.Paragraph = ansi.StyleBlock{}

	var err error
	mdRenderer, err = glamour.NewTermRenderer(
		glamour.WithStyles(style),
		glamour.WithWordWrap(100),
	)
	if err != nil {
		// Fallback: no markdown rendering
		mdRenderer = nil
	}
}

// RenderMarkdown renders markdown text with syntax highlighting.
func RenderMarkdown(text string, width int) string {
	if mdRenderer == nil || text == "" {
		return text
	}

	rendered, err := mdRenderer.Render(text)
	if err != nil {
		return text
	}

	// Glamour adds leading/trailing newlines, trim them
	return strings.Trim(rendered, "\n")
}

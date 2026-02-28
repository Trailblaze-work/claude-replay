package replay

import (
	"strings"

	"github.com/charmbracelet/glamour"
)

var mdRenderer *glamour.TermRenderer

func init() {
	var err error
	mdRenderer, err = glamour.NewTermRenderer(
		glamour.WithAutoStyle(),
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

	// Glamour adds trailing newlines, trim them
	return strings.TrimRight(rendered, "\n")
}

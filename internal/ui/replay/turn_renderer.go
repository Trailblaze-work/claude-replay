package replay

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/Trailblaze-work/claude-replay/internal/session"
	"github.com/Trailblaze-work/claude-replay/internal/ui/theme"
)

// RenderTurn renders a complete turn (user message + all blocks).
func RenderTurn(turn session.Turn, showThinking bool, expandedTools map[string]bool, width int) string {
	var parts []string

	// User message
	userPrefix := lipgloss.NewStyle().
		Foreground(theme.ColorUser).
		Bold(true).
		Render("> ")

	userText := lipgloss.NewStyle().
		Foreground(theme.ColorUser).
		Width(width - 4).
		Render(turn.UserText)

	parts = append(parts, lipgloss.NewStyle().PaddingLeft(1).Render(userPrefix+userText))
	parts = append(parts, "") // blank line

	// Content blocks
	for _, block := range turn.Blocks {
		rendered := RenderBlock(block, showThinking, expandedTools, width)
		if rendered != "" {
			parts = append(parts, rendered)
			parts = append(parts, "") // spacing between blocks
		}
	}

	return strings.Join(parts, "\n")
}

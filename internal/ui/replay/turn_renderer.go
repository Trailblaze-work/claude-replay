package replay

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/Trailblaze-work/claude-replay/internal/session"
	"github.com/Trailblaze-work/claude-replay/internal/ui/theme"
)

var durationVerbs = []string{
	"Brewed", "Baked", "Crafted", "Forged", "Distilled",
	"Composed", "Conjured", "Summoned", "Woven", "Stirred",
	"Blended", "Cooked",
}

// RenderTurn renders a complete turn (user message + all blocks).
func RenderTurn(turn session.Turn, allExpanded bool, width int, cwd string) string {
	var parts []string

	// User message
	userPrefix := lipgloss.NewStyle().
		Foreground(theme.ColorUser).
		Bold(true).
		Render("❯ ")

	userText := lipgloss.NewStyle().
		Foreground(theme.ColorUser).
		Width(width - 4).
		Render(turn.UserText)

	userRendered := lipgloss.NewStyle().PaddingLeft(2).Render(userPrefix + userText)
	parts = append(parts, userRendered)
	parts = append(parts, "") // blank line

	// Build tool_use info lookup for rendering tool results with context
	toolInputs := map[string]toolUseInfo{}
	for _, block := range turn.Blocks {
		if block.Type == session.BlockToolUse {
			toolInputs[block.ToolID] = toolUseInfo{Name: block.ToolName, Input: block.ToolInput}
		}
	}

	// Build readContents: map file paths to content from Read tool results,
	// used to compute diffs for Write operations.
	readContents := map[string]string{}
	for i, block := range turn.Blocks {
		if block.Type == session.BlockToolUse && block.ToolName == "Read" {
			if path, _ := block.ToolInput["file_path"].(string); path != "" {
				// Find matching tool_result
				for _, next := range turn.Blocks[i+1:] {
					if next.Type == session.BlockToolResult && next.ToolID == block.ToolID && !next.IsError {
						readContents[path] = next.Text
						break
					}
				}
			}
		}
	}

	// Content blocks
	for i, block := range turn.Blocks {
		rendered := RenderBlock(block, allExpanded, width, cwd, toolInputs, readContents)
		if rendered != "" {
			parts = append(parts, rendered)

			// Skip blank line between tool_use and its matching tool_result
			addSpacing := true
			if block.Type == session.BlockToolUse && i+1 < len(turn.Blocks) {
				next := turn.Blocks[i+1]
				if next.Type == session.BlockToolResult && next.ToolID == block.ToolID {
					addSpacing = false
				}
			}
			if addSpacing {
				parts = append(parts, "")
			}
		}
	}

	// Duration at the end of the turn (matches Claude Code placement)
	if turn.Duration > 0 {
		durLine := renderDuration(turn.Duration, turn.Number)
		parts = append(parts, durLine)
	}

	return strings.Join(parts, "\n")
}

func renderDuration(d time.Duration, turnNumber int) string {
	verb := durationVerbs[turnNumber%len(durationVerbs)]
	formatted := formatDuration(d)
	star := lipgloss.NewStyle().Foreground(theme.ColorDim).Render("*")
	text := lipgloss.NewStyle().
		Foreground(theme.ColorDim).
		Italic(true).
		Render(fmt.Sprintf("%s for %s", verb, formatted))
	return fmt.Sprintf("    %s %s", star, text)
}

func formatDuration(d time.Duration) string {
	if d < time.Second {
		return fmt.Sprintf("%dms", d.Milliseconds())
	}
	if d < time.Minute {
		secs := d.Seconds()
		if secs == float64(int(secs)) {
			return fmt.Sprintf("%ds", int(secs))
		}
		return fmt.Sprintf("%.1fs", secs)
	}
	minutes := int(d.Minutes())
	seconds := int(d.Seconds()) % 60
	if seconds == 0 {
		return fmt.Sprintf("%dm", minutes)
	}
	return fmt.Sprintf("%dm %ds", minutes, seconds)
}

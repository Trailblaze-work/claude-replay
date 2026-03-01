package replay

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/trailblaze/claude-replay/internal/session"
	"github.com/trailblaze/claude-replay/internal/ui/theme"
)

const shortResultThreshold = 3

// RenderBlock renders a single content block.
func RenderBlock(block session.Block, showThinking bool, expandedTools map[string]bool, width int) string {
	contentWidth := width - 4
	if contentWidth < 20 {
		contentWidth = 20
	}

	switch block.Type {
	case session.BlockText:
		return renderTextBlock(block.Text, contentWidth)
	case session.BlockThinking:
		return renderThinkingBlock(block.Text, showThinking, contentWidth)
	case session.BlockToolUse:
		expanded := expandedTools != nil && expandedTools[block.ToolID]
		return renderToolUseBlock(block, expanded, contentWidth)
	case session.BlockToolResult:
		expanded := expandedTools != nil && expandedTools[block.ToolID]
		return renderToolResultBlock(block, expanded, contentWidth)
	default:
		return ""
	}
}

func renderTextBlock(text string, width int) string {
	rendered := RenderMarkdown(text, width)
	return lipgloss.NewStyle().
		PaddingLeft(2).
		Width(width + 2).
		Render(rendered)
}

func renderThinkingBlock(text string, expanded bool, width int) string {
	charCount := len(text)
	header := lipgloss.NewStyle().
		Foreground(theme.ColorThinking).
		Italic(true).
		PaddingLeft(2).
		Render(fmt.Sprintf("thinking (%d chars)  [t:toggle]", charCount))

	if !expanded {
		return header
	}

	displayText := text
	if len(displayText) > 2000 {
		displayText = displayText[:2000] + "\n... (truncated)"
	}

	body := lipgloss.NewStyle().
		Foreground(theme.ColorDim).
		PaddingLeft(4).
		Width(width).
		Render(displayText)

	return header + "\n" + body
}

func renderToolUseBlock(block session.Block, expanded bool, width int) string {
	icon := toolIcon(block.ToolName)
	header := lipgloss.NewStyle().
		Foreground(theme.ColorToolUse).
		Bold(true).
		PaddingLeft(2).
		Render(fmt.Sprintf("%s %s", icon, block.ToolName))

	detail := renderToolInput(block, expanded, width)
	if detail != "" {
		return header + "\n" + detail
	}
	return header
}

func renderToolInput(block session.Block, expanded bool, width int) string {
	input := block.ToolInput
	if input == nil {
		return ""
	}

	style := lipgloss.NewStyle().
		Foreground(theme.ColorSecondary).
		PaddingLeft(4).
		Width(width)

	switch block.ToolName {
	case "Bash":
		cmd, _ := input["command"].(string)
		desc, _ := input["description"].(string)
		if !expanded {
			if desc != "" {
				return style.Render(desc)
			}
			return style.Render(truncateLines(cmd, 1))
		}
		if desc != "" {
			return style.Render(desc) + "\n" + style.Foreground(theme.ColorDim).Render(truncateLines(cmd, 20))
		}
		return style.Render(truncateLines(cmd, 20))

	case "Read":
		path, _ := input["file_path"].(string)
		return style.Render(path)

	case "Write":
		path, _ := input["file_path"].(string)
		content, _ := input["content"].(string)
		lines := strings.Count(content, "\n") + 1
		return style.Render(fmt.Sprintf("%s (%d lines)", path, lines))

	case "Edit":
		path, _ := input["file_path"].(string)
		if !expanded {
			return style.Render(path)
		}
		oldStr, _ := input["old_string"].(string)
		newStr, _ := input["new_string"].(string)
		result := path + "\n"
		if oldStr != "" {
			result += lipgloss.NewStyle().Foreground(theme.ColorError).Render("- "+truncateLines(oldStr, 3)) + "\n"
		}
		if newStr != "" {
			result += lipgloss.NewStyle().Foreground(theme.ColorSuccess).Render("+ "+truncateLines(newStr, 3))
		}
		return lipgloss.NewStyle().PaddingLeft(4).Width(width).Render(result)

	case "Glob":
		pattern, _ := input["pattern"].(string)
		return style.Render(pattern)

	case "Grep":
		pattern, _ := input["pattern"].(string)
		path, _ := input["path"].(string)
		if path != "" {
			return style.Render(fmt.Sprintf("/%s/ in %s", pattern, path))
		}
		return style.Render(fmt.Sprintf("/%s/", pattern))

	case "WebFetch":
		url, _ := input["url"].(string)
		return style.Render(url)

	case "WebSearch":
		query, _ := input["query"].(string)
		return style.Render(fmt.Sprintf("\"%s\"", query))

	case "Task", "Agent":
		desc, _ := input["description"].(string)
		prompt, _ := input["prompt"].(string)
		if desc != "" {
			return style.Render(desc)
		}
		if !expanded {
			return style.Render(truncateLines(prompt, 1))
		}
		return style.Render(truncateLines(prompt, 3))

	default:
		b, err := json.MarshalIndent(input, "", "  ")
		if err != nil {
			return ""
		}
		if !expanded {
			return style.Render(truncateLines(string(b), 2))
		}
		return style.Render(truncateLines(string(b), 8))
	}
}

func renderToolResultBlock(block session.Block, expanded bool, width int) string {
	text := block.Text

	if block.IsError {
		errorStyle := lipgloss.NewStyle().
			Foreground(theme.ColorError).
			PaddingLeft(4).
			Width(width)
		return errorStyle.Render("✗ Error: " + truncateLines(text, 5))
	}

	if text == "" {
		return lipgloss.NewStyle().
			Foreground(theme.ColorDim).
			PaddingLeft(4).
			Render("(empty result)")
	}

	lines := strings.Split(text, "\n")
	style := lipgloss.NewStyle().
		Foreground(theme.ColorDim).
		PaddingLeft(4).
		Width(width)

	if !expanded && len(lines) > shortResultThreshold {
		return lipgloss.NewStyle().
			Foreground(theme.ColorWarning).
			PaddingLeft(4).
			Render(fmt.Sprintf("↳ %d lines [enter to expand]", len(lines)))
	}

	return style.Render(text)
}

func toolIcon(name string) string {
	switch name {
	case "Bash":
		return ">"
	case "Read":
		return "#"
	case "Write":
		return "+"
	case "Edit":
		return "~"
	case "Glob":
		return "*"
	case "Grep":
		return "/"
	case "WebFetch", "WebSearch":
		return "@"
	case "Task", "Agent":
		return "&"
	default:
		return ">"
	}
}

func truncateLines(s string, maxLines int) string {
	lines := strings.Split(s, "\n")
	if len(lines) <= maxLines {
		return s
	}
	return strings.Join(lines[:maxLines], "\n") + fmt.Sprintf("\n... (%d more lines)", len(lines)-maxLines)
}

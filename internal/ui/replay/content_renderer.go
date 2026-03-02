package replay

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/Trailblaze-work/claude-replay/internal/session"
	"github.com/Trailblaze-work/claude-replay/internal/ui/theme"
)

const shortResultThreshold = 3

// toolUseInfo stores tool_use block info for rendering tool results with context.
type toolUseInfo struct {
	Name  string
	Input map[string]interface{}
}

// toolDisplayNames maps internal tool names to display names matching Claude Code.
var toolDisplayNames = map[string]string{
	"Edit": "Update",
}

func toolDisplayName(name string) string {
	if dn, ok := toolDisplayNames[name]; ok {
		return dn
	}
	return name
}

// shortenPath strips the CWD prefix from a path to show relative paths.
func shortenPath(path, cwd string) string {
	if cwd != "" && strings.HasPrefix(path, cwd) {
		rel := strings.TrimPrefix(path, cwd)
		return strings.TrimPrefix(rel, "/")
	}
	return path
}

// RenderBlock renders a single content block.
// readContents maps file paths to their content from earlier Read results,
// used to compute diffs for Write operations.
func RenderBlock(block session.Block, allExpanded bool, width int, cwd string, toolInputs map[string]toolUseInfo, readContents map[string]string) string {
	contentWidth := width - 4
	if contentWidth < 20 {
		contentWidth = 20
	}

	switch block.Type {
	case session.BlockText:
		return renderTextBlock(block.Text, contentWidth)
	case session.BlockThinking:
		return renderThinkingBlock(block.Text, allExpanded, contentWidth)
	case session.BlockToolUse:
		return renderToolUseBlock(block, allExpanded, contentWidth, cwd, readContents)
	case session.BlockToolResult:
		return renderToolResultBlock(block, allExpanded, contentWidth, cwd, toolInputs, readContents)
	default:
		return ""
	}
}

func renderTextBlock(text string, width int) string {
	bullet := lipgloss.NewStyle().Foreground(theme.ColorSuccess).Render("●")
	rendered := RenderMarkdown(text, width-4)
	lines := strings.Split(rendered, "\n")
	for i, line := range lines {
		if i == 0 {
			lines[i] = fmt.Sprintf("  %s %s", bullet, line)
		} else {
			lines[i] = "    " + line
		}
	}
	return strings.Join(lines, "\n")
}

func renderThinkingBlock(text string, expanded bool, width int) string {
	charCount := len(text)
	header := lipgloss.NewStyle().
		Foreground(theme.ColorThinking).
		Italic(true).
		PaddingLeft(2).
		Render(fmt.Sprintf("thinking (%d chars)  [ctrl+o:toggle]", charCount))

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

func renderToolUseBlock(block session.Block, expanded bool, width int, cwd string, readContents map[string]string) string {
	bullet := lipgloss.NewStyle().
		Foreground(theme.ColorSuccess).
		Render("●")
	displayName := toolDisplayName(block.ToolName)
	name := lipgloss.NewStyle().
		Bold(true).
		Render(displayName)

	// Read collapsed: "Read 1 file (ctrl+o to expand)" — no path, no result
	if block.ToolName == "Read" && !expanded {
		hint := lipgloss.NewStyle().Foreground(theme.ColorDim).Render("(ctrl+o to expand)")
		return fmt.Sprintf("  %s %s %s %s", bullet, name, "1 file", hint)
	}

	brief := toolBriefParam(block, cwd)
	var header string
	if brief != "" {
		paramStyle := lipgloss.NewStyle()
		if block.ToolName != "Bash" {
			paramStyle = paramStyle.Foreground(theme.ColorDim)
		}
		param := paramStyle.Render("(" + brief + ")")
		header = fmt.Sprintf("  %s %s%s", bullet, name, param)
	} else {
		header = fmt.Sprintf("  %s %s", bullet, name)
	}

	// Edit/Write diffs are always shown; other tools only when expanded
	if expanded || block.ToolName == "Edit" || block.ToolName == "Write" {
		detail := renderToolInput(block, true, width, cwd, readContents)
		if detail != "" {
			return header + "\n" + detail
		}
	}
	return header
}

func toolBriefParam(block session.Block, cwd string) string {
	input := block.ToolInput
	if input == nil {
		return ""
	}

	switch block.ToolName {
	case "Bash":
		if cmd, _ := input["command"].(string); cmd != "" {
			if idx := strings.IndexByte(cmd, '\n'); idx >= 0 {
				cmd = cmd[:idx]
			}
			return cmd
		}
	case "Read":
		path, _ := input["file_path"].(string)
		return shortenPath(path, cwd)
	case "Write":
		path, _ := input["file_path"].(string)
		return shortenPath(path, cwd)
	case "Edit":
		path, _ := input["file_path"].(string)
		return shortenPath(path, cwd)
	case "Glob":
		pattern, _ := input["pattern"].(string)
		return pattern
	case "Grep":
		pattern, _ := input["pattern"].(string)
		path, _ := input["path"].(string)
		if path != "" {
			return fmt.Sprintf("/%s/ in %s", pattern, shortenPath(path, cwd))
		}
		return fmt.Sprintf("/%s/", pattern)
	case "WebFetch":
		url, _ := input["url"].(string)
		return url
	case "WebSearch":
		query, _ := input["query"].(string)
		return fmt.Sprintf("\"%s\"", query)
	case "Agent", "Task":
		if desc, _ := input["description"].(string); desc != "" {
			return truncateString(desc, 60)
		}
	}
	return ""
}

func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-1] + "…"
}

func renderToolInput(block session.Block, expanded bool, width int, cwd string, readContents map[string]string) string {
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
		return style.Render(shortenPath(path, cwd))

	case "Write":
		path, _ := input["file_path"].(string)
		content, _ := input["content"].(string)
		if oldContent, ok := readContents[path]; ok {
			return renderWriteDiff(oldContent, content, path, width, cwd)
		}
		// No prior Read: show as new file
		lines := strings.Count(content, "\n") + 1
		return style.Render(fmt.Sprintf("%s (%d lines)", shortenPath(path, cwd), lines))

	case "Edit":
		path, _ := input["file_path"].(string)
		if !expanded {
			return style.Render(shortenPath(path, cwd))
		}
		return renderEditDiff(input, width, cwd)

	case "Glob":
		pattern, _ := input["pattern"].(string)
		return style.Render(pattern)

	case "Grep":
		pattern, _ := input["pattern"].(string)
		path, _ := input["path"].(string)
		if path != "" {
			return style.Render(fmt.Sprintf("/%s/ in %s", pattern, shortenPath(path, cwd)))
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

func renderToolResultBlock(block session.Block, expanded bool, width int, cwd string, toolInputs map[string]toolUseInfo, readContents map[string]string) string {
	text := block.Text
	resultColor := theme.ColorSecondary
	bracket := lipgloss.NewStyle().
		Foreground(resultColor).
		Render("⎿")

	if block.IsError {
		errorText := lipgloss.NewStyle().
			Foreground(theme.ColorError).
			Render("✗ Error: " + truncateLines(text, 5))
		return fmt.Sprintf("    %s  %s", bracket, errorText)
	}

	// Check if this is an Edit tool result — show diff instead of raw text
	if info, ok := toolInputs[block.ToolID]; ok && info.Name == "Edit" && !block.IsError {
		return renderEditResultBlock(info.Input, expanded, width, cwd, bracket)
	}

	// Check if this is a Write tool result — show diff summary
	if info, ok := toolInputs[block.ToolID]; ok && info.Name == "Write" && !block.IsError {
		return renderWriteResultBlock(info.Input, readContents, bracket)
	}

	// Read tool result: collapsed = hidden (merged into tool_use header),
	// expanded = "Read N lines"
	if info, ok := toolInputs[block.ToolID]; ok && info.Name == "Read" {
		if !expanded {
			return ""
		}
		lineCount := len(strings.Split(text, "\n"))
		summary := lipgloss.NewStyle().
			Foreground(resultColor).
			Render(fmt.Sprintf("Read %d lines", lineCount))
		return fmt.Sprintf("    %s  %s", bracket, summary)
	}

	if text == "" {
		emptyText := lipgloss.NewStyle().
			Foreground(resultColor).
			Render("(No output)")
		return fmt.Sprintf("    %s  %s", bracket, emptyText)
	}

	lines := strings.Split(text, "\n")

	if !expanded && len(lines) > shortResultThreshold {
		hint := lipgloss.NewStyle().
			Foreground(resultColor).
			Render(fmt.Sprintf("… +%d lines (ctrl+o to expand)", len(lines)))
		return fmt.Sprintf("    %s  %s", bracket, hint)
	}

	// Short or expanded result: show with bracket prefix
	style := lipgloss.NewStyle().
		Foreground(resultColor).
		Width(width)
	return fmt.Sprintf("    %s  %s", bracket, style.Render(text))
}

// diffOp represents one line in a computed diff.
type diffOp struct {
	Kind byte   // ' ' context, '+' added, '-' removed
	Text string // the line content
}

// computeDiff computes a line-level diff between old and new text using LCS.
func computeDiff(oldStr, newStr string) []diffOp {
	oldLines := splitLines(oldStr)
	newLines := splitLines(newStr)

	// LCS table
	m, n := len(oldLines), len(newLines)
	dp := make([][]int, m+1)
	for i := range dp {
		dp[i] = make([]int, n+1)
	}
	for i := 1; i <= m; i++ {
		for j := 1; j <= n; j++ {
			if oldLines[i-1] == newLines[j-1] {
				dp[i][j] = dp[i-1][j-1] + 1
			} else if dp[i-1][j] >= dp[i][j-1] {
				dp[i][j] = dp[i-1][j]
			} else {
				dp[i][j] = dp[i][j-1]
			}
		}
	}

	// Backtrack to produce diff ops
	var ops []diffOp
	i, j := m, n
	for i > 0 || j > 0 {
		if i > 0 && j > 0 && oldLines[i-1] == newLines[j-1] {
			ops = append(ops, diffOp{' ', oldLines[i-1]})
			i--
			j--
		} else if j > 0 && (i == 0 || dp[i][j-1] >= dp[i-1][j]) {
			ops = append(ops, diffOp{'+', newLines[j-1]})
			j--
		} else {
			ops = append(ops, diffOp{'-', oldLines[i-1]})
			i--
		}
	}

	// Reverse (we built it backwards)
	for l, r := 0, len(ops)-1; l < r; l, r = l+1, r-1 {
		ops[l], ops[r] = ops[r], ops[l]
	}
	return ops
}

func splitLines(s string) []string {
	if s == "" {
		return nil
	}
	return strings.Split(s, "\n")
}

// countDiffChanges returns the number of added and removed lines.
func countDiffChanges(ops []diffOp) (added, removed int) {
	for _, op := range ops {
		switch op.Kind {
		case '+':
			added++
		case '-':
			removed++
		}
	}
	return
}

// renderEditDiff renders the old_string/new_string diff for an Edit tool_use block,
// using LCS-based diff with full-width background highlights and syntax highlighting.
func renderEditDiff(input map[string]interface{}, width int, cwd string) string {
	path, _ := input["file_path"].(string)
	oldStr, _ := input["old_string"].(string)
	newStr, _ := input["new_string"].(string)

	diffWidth := width - 4
	if diffWidth < 20 {
		diffWidth = 20
	}

	var out []string
	out = append(out, "    "+shortenPath(path, cwd))

	ops := computeDiff(oldStr, newStr)

	// Get lexer once for all lines
	lexer := getLexer(path)

	ctxStyle := lipgloss.NewStyle().
		Foreground(theme.ColorDiffCtx).
		Width(diffWidth)

	for _, op := range ops {
		var rendered string
		switch op.Kind {
		case '-':
			rendered = highlightDiffLine("- ", op.Text, lexer, theme.ColorDiffDelBg, theme.ColorDiffDelFg, diffWidth)
		case '+':
			rendered = highlightDiffLine("+ ", op.Text, lexer, theme.ColorDiffAddBg, theme.ColorDiffAddFg, diffWidth)
		default:
			rendered = ctxStyle.Render("  " + op.Text)
		}
		out = append(out, "    "+rendered)
	}

	return strings.Join(out, "\n")
}

func renderEditResultBlock(input map[string]interface{}, expanded bool, width int, cwd string, bracket string) string {
	oldStr, _ := input["old_string"].(string)
	newStr, _ := input["new_string"].(string)

	ops := computeDiff(oldStr, newStr)
	added, removed := countDiffChanges(ops)

	var summary string
	if removed == 0 {
		summary = fmt.Sprintf("Added %d lines", added)
	} else {
		summary = fmt.Sprintf("Added %d lines, removed %d lines", added, removed)
	}

	rendered := lipgloss.NewStyle().
		Foreground(theme.ColorSecondary).
		Render(summary)

	return fmt.Sprintf("    %s  %s", bracket, rendered)
}

// renderWriteDiff renders a diff between old file content (from a prior Read) and new Write content.
func renderWriteDiff(oldContent, newContent, path string, width int, cwd string) string {
	diffWidth := width - 4
	if diffWidth < 20 {
		diffWidth = 20
	}

	var out []string
	out = append(out, "    "+shortenPath(path, cwd))

	ops := computeDiff(oldContent, newContent)
	lexer := getLexer(path)

	ctxStyle := lipgloss.NewStyle().
		Foreground(theme.ColorDiffCtx).
		Width(diffWidth)

	for _, op := range ops {
		var rendered string
		switch op.Kind {
		case '-':
			rendered = highlightDiffLine("- ", op.Text, lexer, theme.ColorDiffDelBg, theme.ColorDiffDelFg, diffWidth)
		case '+':
			rendered = highlightDiffLine("+ ", op.Text, lexer, theme.ColorDiffAddBg, theme.ColorDiffAddFg, diffWidth)
		default:
			rendered = ctxStyle.Render("  " + op.Text)
		}
		out = append(out, "    "+rendered)
	}

	return strings.Join(out, "\n")
}

// renderWriteResultBlock renders a Write tool_result as a diff summary.
func renderWriteResultBlock(input map[string]interface{}, readContents map[string]string, bracket string) string {
	path, _ := input["file_path"].(string)
	content, _ := input["content"].(string)

	oldContent, hasOld := readContents[path]
	if !hasOld {
		// No prior Read: show line count
		lines := strings.Count(content, "\n") + 1
		summary := lipgloss.NewStyle().
			Foreground(theme.ColorSecondary).
			Render(fmt.Sprintf("Wrote %d lines", lines))
		return fmt.Sprintf("    %s  %s", bracket, summary)
	}

	ops := computeDiff(oldContent, content)
	added, removed := countDiffChanges(ops)

	var summary string
	if removed == 0 {
		summary = fmt.Sprintf("Added %d lines", added)
	} else {
		summary = fmt.Sprintf("Added %d lines, removed %d lines", added, removed)
	}

	rendered := lipgloss.NewStyle().
		Foreground(theme.ColorSecondary).
		Render(summary)

	return fmt.Sprintf("    %s  %s", bracket, rendered)
}

func truncateLines(s string, maxLines int) string {
	lines := strings.Split(s, "\n")
	if len(lines) <= maxLines {
		return s
	}
	return strings.Join(lines[:maxLines], "\n") + fmt.Sprintf("\n... (%d more lines)", len(lines)-maxLines)
}

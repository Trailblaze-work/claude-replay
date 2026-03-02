package replay

import (
	"fmt"
	"strings"
	"testing"

	"github.com/Trailblaze-work/claude-replay/internal/session"
)

func TestToolIcon(t *testing.T) {
	tests := []struct {
		name     string
		expected string
	}{
		{"Bash", ">"},
		{"Read", "#"},
		{"Write", "+"},
		{"Edit", "~"},
		{"Glob", "*"},
		{"Grep", "/"},
		{"WebFetch", "@"},
		{"WebSearch", "@"},
		{"Task", "&"},
		{"Agent", "&"},
		{"UnknownTool", ">"},
	}

	for _, tt := range tests {
		got := toolIcon(tt.name)
		if got != tt.expected {
			t.Errorf("toolIcon(%q) = %q, want %q", tt.name, got, tt.expected)
		}
	}
}

func TestTruncateLines_Short(t *testing.T) {
	input := "line1\nline2\nline3"
	got := truncateLines(input, 5)
	if got != input {
		t.Errorf("expected no truncation, got %q", got)
	}
}

func TestTruncateLines_Long(t *testing.T) {
	input := "line1\nline2\nline3\nline4\nline5\nline6\nline7"
	got := truncateLines(input, 3)
	if !strings.HasPrefix(got, "line1\nline2\nline3\n") {
		t.Errorf("expected first 3 lines preserved, got %q", got)
	}
	if !strings.Contains(got, "4 more lines") {
		t.Errorf("expected '4 more lines' suffix, got %q", got)
	}
}

func TestRenderTurn_ContainsUserText(t *testing.T) {
	turn := session.Turn{
		Number:   1,
		UserText: "What is Go?",
		Blocks: []session.Block{
			{Type: session.BlockText, Text: "Go is a programming language."},
		},
	}

	output := RenderTurn(turn, false, nil, 80)
	if !strings.Contains(output, "What is Go?") {
		t.Error("output should contain user text")
	}
}

func TestRenderTurn_ContainsBlocks(t *testing.T) {
	turn := session.Turn{
		Number:   1,
		UserText: "hello",
		Blocks: []session.Block{
			{Type: session.BlockText, Text: "response text here"},
			{Type: session.BlockToolUse, ToolName: "Bash", ToolInput: map[string]interface{}{"command": "ls"}},
		},
	}

	output := RenderTurn(turn, false, nil, 80)
	if !strings.Contains(output, "response text here") {
		t.Error("output should contain text block content")
	}
	if !strings.Contains(output, "Bash") {
		t.Error("output should contain tool name")
	}
}

func TestRenderBlock_TextBlock(t *testing.T) {
	block := session.Block{Type: session.BlockText, Text: "Hello world"}
	output := RenderBlock(block, false, nil, 80)
	if output == "" {
		t.Error("expected non-empty output for text block")
	}
	if !strings.Contains(output, "Hello world") {
		t.Error("output should contain block text")
	}
}

func TestRenderBlock_UnknownType(t *testing.T) {
	block := session.Block{Type: session.BlockType(99), Text: "unknown"}
	output := RenderBlock(block, false, nil, 80)
	if output != "" {
		t.Errorf("expected empty output for unknown block type, got %q", output)
	}
}

func TestRenderBlock_ThinkingCollapsed(t *testing.T) {
	block := session.Block{Type: session.BlockThinking, Text: "Let me think about this..."}
	output := RenderBlock(block, false, nil, 80)
	if output == "" {
		t.Error("expected non-empty output for thinking block")
	}
	if !strings.Contains(output, "thinking") {
		t.Error("output should contain 'thinking' header")
	}
	// When collapsed (showThinking=false), the thinking body should not appear
	if strings.Contains(output, "Let me think about this...") {
		t.Error("collapsed thinking should not show body text")
	}
}

func TestRenderBlock_ThinkingExpanded(t *testing.T) {
	block := session.Block{Type: session.BlockThinking, Text: "Deep thoughts here"}
	output := RenderBlock(block, true, nil, 80)
	if !strings.Contains(output, "Deep thoughts here") {
		t.Error("expanded thinking should show body text")
	}
}

func TestRenderBlock_ToolUse(t *testing.T) {
	block := session.Block{
		Type:      session.BlockToolUse,
		ToolName:  "Read",
		ToolInput: map[string]interface{}{"file_path": "/tmp/test.go"},
	}
	output := RenderBlock(block, false, nil, 80)
	if !strings.Contains(output, "Read") {
		t.Error("output should contain tool name")
	}
	if !strings.Contains(output, "/tmp/test.go") {
		t.Error("output should contain file path")
	}
}

func TestRenderBlock_ToolResult(t *testing.T) {
	block := session.Block{
		Type:   session.BlockToolResult,
		ToolID: "tool_1",
		Text:   "file contents here",
	}
	output := RenderBlock(block, false, nil, 80)
	if !strings.Contains(output, "file contents here") {
		t.Error("output should contain tool result text")
	}
}

func TestRenderBlock_ToolResultError(t *testing.T) {
	block := session.Block{
		Type:    session.BlockToolResult,
		ToolID:  "tool_1",
		Text:    "command not found",
		IsError: true,
	}
	output := RenderBlock(block, false, nil, 80)
	if !strings.Contains(output, "Error") {
		t.Error("error result should contain 'Error'")
	}
}

func TestRenderBlock_ToolResultEmpty(t *testing.T) {
	block := session.Block{
		Type:   session.BlockToolResult,
		ToolID: "tool_1",
		Text:   "",
	}
	output := RenderBlock(block, false, nil, 80)
	if !strings.Contains(output, "empty result") {
		t.Error("empty result should show '(empty result)'")
	}
}

func TestRenderBlock_ToolResultExpanded(t *testing.T) {
	// Create a long result that would normally be truncated
	lines := make([]string, 30)
	for i := range lines {
		lines[i] = "line content"
	}
	longText := strings.Join(lines, "\n")

	block := session.Block{
		Type:   session.BlockToolResult,
		ToolID: "tool_1",
		Text:   longText,
	}

	// Collapsed: should truncate
	collapsed := RenderBlock(block, false, nil, 80)
	if !strings.Contains(collapsed, "expand") {
		t.Error("long collapsed result should show expand hint")
	}

	// Expanded: should show all
	expanded := RenderBlock(block, false, map[string]bool{"tool_1": true}, 80)
	if strings.Contains(expanded, "expand") {
		t.Error("expanded result should not show expand hint")
	}
}

func TestRenderToolInput_Various(t *testing.T) {
	tests := []struct {
		name     string
		block    session.Block
		contains string
	}{
		{
			name: "Bash with description",
			block: session.Block{
				ToolName:  "Bash",
				ToolInput: map[string]interface{}{"command": "ls -la", "description": "list files"},
			},
			contains: "list files",
		},
		{
			name: "Write with content",
			block: session.Block{
				ToolName:  "Write",
				ToolInput: map[string]interface{}{"file_path": "/tmp/out.txt", "content": "line1\nline2\nline3"},
			},
			contains: "3 lines",
		},
		{
			name: "Grep with path",
			block: session.Block{
				ToolName:  "Grep",
				ToolInput: map[string]interface{}{"pattern": "TODO", "path": "src/"},
			},
			contains: "/TODO/",
		},
		{
			name: "Glob pattern",
			block: session.Block{
				ToolName:  "Glob",
				ToolInput: map[string]interface{}{"pattern": "**/*.go"},
			},
			contains: "**/*.go",
		},
		{
			name: "WebSearch query",
			block: session.Block{
				ToolName:  "WebSearch",
				ToolInput: map[string]interface{}{"query": "golang testing"},
			},
			contains: "golang testing",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output := renderToolInput(tt.block, false, 80)
			if !strings.Contains(output, tt.contains) {
				t.Errorf("expected output to contain %q, got %q", tt.contains, output)
			}
		})
	}
}

func TestRenderBlock_ToolResultCollapsed30Lines(t *testing.T) {
	lines := make([]string, 30)
	for i := range lines {
		lines[i] = fmt.Sprintf("content line %d", i+1)
	}
	longText := strings.Join(lines, "\n")

	block := session.Block{
		Type:   session.BlockToolResult,
		ToolID: "tool_1",
		Text:   longText,
	}

	collapsed := RenderBlock(block, false, nil, 80)
	// Should show summary only, no content lines
	if strings.Contains(collapsed, "content line") {
		t.Error("collapsed 30-line result should not show any content lines")
	}
	if !strings.Contains(collapsed, "30 lines") {
		t.Error("collapsed result should show '30 lines'")
	}
	if !strings.Contains(collapsed, "enter to expand") {
		t.Error("collapsed result should show expand hint")
	}
}

func TestRenderToolInput_EditCollapsedExpanded(t *testing.T) {
	block := session.Block{
		ToolName: "Edit",
		ToolInput: map[string]interface{}{
			"file_path":  "/tmp/test.go",
			"old_string": "old code here",
			"new_string": "new code here",
		},
	}

	// Collapsed: should show path only, no diff
	collapsed := renderToolInput(block, false, 80)
	if !strings.Contains(collapsed, "/tmp/test.go") {
		t.Error("collapsed Edit should show file path")
	}
	if strings.Contains(collapsed, "old code") || strings.Contains(collapsed, "new code") {
		t.Error("collapsed Edit should not show diff content")
	}

	// Expanded: should show path and diff
	expanded := renderToolInput(block, true, 80)
	if !strings.Contains(expanded, "/tmp/test.go") {
		t.Error("expanded Edit should show file path")
	}
	if !strings.Contains(expanded, "old code") {
		t.Error("expanded Edit should show old string")
	}
	if !strings.Contains(expanded, "new code") {
		t.Error("expanded Edit should show new string")
	}
}

func TestRenderMarkdown_Plain(t *testing.T) {
	output := RenderMarkdown("Hello **world**", 80)
	if output == "" {
		t.Error("expected non-empty markdown output")
	}
	// Should contain "world" regardless of formatting
	if !strings.Contains(output, "world") {
		t.Error("markdown output should contain the text")
	}
}

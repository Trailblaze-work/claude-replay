package replay

import (
	"fmt"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/Trailblaze-work/claude-replay/internal/session"
)

var ansiRe = regexp.MustCompile(`\x1b\[[0-9;]*m`)

// stripANSI removes ANSI escape codes for test assertions.
func stripANSI(s string) string {
	return ansiRe.ReplaceAllString(s, "")
}

func TestToolMarkerIsBullet(t *testing.T) {
	tools := []string{"Bash", "Read", "Write", "Edit", "Glob", "Grep", "WebFetch", "WebSearch", "Task", "Agent", "UnknownTool"}
	for _, name := range tools {
		block := session.Block{
			Type:     session.BlockToolUse,
			ToolName: name,
		}
		output := RenderBlock(block, false, 80, "", nil, nil)
		if !strings.Contains(output, "●") {
			t.Errorf("tool %q header should contain ● marker, got %q", name, output)
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

	output := RenderTurn(turn, false, 80, "")
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

	output := RenderTurn(turn, false, 80, "")
	plain := stripANSI(output)
	if !strings.Contains(plain, "response text here") {
		t.Error("output should contain text block content")
	}
	if !strings.Contains(output, "Bash") {
		t.Error("output should contain tool name")
	}
}

func TestRenderBlock_TextBlock(t *testing.T) {
	block := session.Block{Type: session.BlockText, Text: "Hello world"}
	output := RenderBlock(block, false, 80, "", nil, nil)
	if output == "" {
		t.Error("expected non-empty output for text block")
	}
	plain := stripANSI(output)
	if !strings.Contains(plain, "Hello world") {
		t.Error("output should contain block text")
	}
}

func TestRenderBlock_UnknownType(t *testing.T) {
	block := session.Block{Type: session.BlockType(99), Text: "unknown"}
	output := RenderBlock(block, false, 80, "", nil, nil)
	if output != "" {
		t.Errorf("expected empty output for unknown block type, got %q", output)
	}
}

func TestRenderBlock_ThinkingCollapsed(t *testing.T) {
	block := session.Block{Type: session.BlockThinking, Text: "Let me think about this..."}
	output := RenderBlock(block, false, 80, "", nil, nil)
	if output == "" {
		t.Error("expected non-empty output for thinking block")
	}
	if !strings.Contains(output, "thinking") {
		t.Error("output should contain 'thinking' header")
	}
	if strings.Contains(output, "Let me think about this...") {
		t.Error("collapsed thinking should not show body text")
	}
}

func TestRenderBlock_ThinkingExpanded(t *testing.T) {
	block := session.Block{Type: session.BlockThinking, Text: "Deep thoughts here"}
	output := RenderBlock(block, true, 80, "", nil, nil)
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
	// Collapsed Read shows summary, not path
	collapsed := RenderBlock(block, false, 80, "", nil, nil)
	if !strings.Contains(collapsed, "●") {
		t.Error("output should contain ● marker")
	}
	if !strings.Contains(collapsed, "Read") {
		t.Error("output should contain tool name")
	}
	if !strings.Contains(collapsed, "1 file") {
		t.Error("collapsed Read should show '1 file' summary")
	}

	// Expanded Read shows path
	expanded := RenderBlock(block, true, 80, "", nil, nil)
	if !strings.Contains(expanded, "/tmp/test.go") {
		t.Error("expanded Read should contain file path")
	}
}

func TestRenderBlock_ToolUseInlineParam(t *testing.T) {
	tests := []struct {
		name     string
		block    session.Block
		contains string
	}{
		{
			name: "Bash shows command inline",
			block: session.Block{
				Type:      session.BlockToolUse,
				ToolName:  "Bash",
				ToolInput: map[string]interface{}{"command": "ls -la"},
			},
			contains: "(ls -la)",
		},
		{
			name: "Bash always shows command even with description",
			block: session.Block{
				Type:      session.BlockToolUse,
				ToolName:  "Bash",
				ToolInput: map[string]interface{}{"command": "ls -la", "description": "list files"},
			},
			contains: "(ls -la)",
		},
		{
			name: "Grep with path",
			block: session.Block{
				Type:      session.BlockToolUse,
				ToolName:  "Grep",
				ToolInput: map[string]interface{}{"pattern": "TODO", "path": "src/"},
			},
			contains: "(/TODO/ in src/)",
		},
		{
			name: "WebSearch with query",
			block: session.Block{
				Type:      session.BlockToolUse,
				ToolName:  "WebSearch",
				ToolInput: map[string]interface{}{"query": "golang"},
			},
			contains: "(\"golang\")",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output := RenderBlock(tt.block, false, 80, "", nil, nil)
			if !strings.Contains(output, tt.contains) {
				t.Errorf("expected output to contain %q, got %q", tt.contains, output)
			}
		})
	}
}

func TestRenderBlock_ToolResult(t *testing.T) {
	block := session.Block{
		Type:   session.BlockToolResult,
		ToolID: "tool_1",
		Text:   "file contents here",
	}
	output := RenderBlock(block, false, 80, "", nil, nil)
	if !strings.Contains(output, "⎿") {
		t.Error("output should contain ⎿ bracket prefix")
	}
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
	output := RenderBlock(block, false, 80, "", nil, nil)
	if !strings.Contains(output, "⎿") {
		t.Error("error result should contain ⎿ bracket")
	}
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
	output := RenderBlock(block, false, 80, "", nil, nil)
	if !strings.Contains(output, "⎿") {
		t.Error("empty result should contain ⎿ bracket")
	}
	if !strings.Contains(output, "(No output)") {
		t.Error("empty result should show '(No output)'")
	}
}

func TestRenderBlock_ToolResultExpanded(t *testing.T) {
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
	collapsed := RenderBlock(block, false, 80, "", nil, nil)
	if !strings.Contains(collapsed, "expand") {
		t.Error("long collapsed result should show expand hint")
	}

	// Expanded: should show all
	expanded := RenderBlock(block, true, 80, "", nil, nil)
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
			output := renderToolInput(tt.block, false, 80, "", nil)
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

	collapsed := RenderBlock(block, false, 80, "", nil, nil)
	if strings.Contains(collapsed, "content line") {
		t.Error("collapsed 30-line result should not show any content lines")
	}
	if !strings.Contains(collapsed, "⎿") {
		t.Error("collapsed result should contain ⎿ bracket")
	}
	if !strings.Contains(collapsed, "+30 lines") {
		t.Error("collapsed result should show '+30 lines'")
	}
	if !strings.Contains(collapsed, "(ctrl+o to expand)") {
		t.Error("collapsed result should show '(ctrl+o to expand)' hint")
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
	collapsed := renderToolInput(block, false, 80, "", nil)
	if !strings.Contains(collapsed, "/tmp/test.go") {
		t.Error("collapsed Edit should show file path")
	}
	if strings.Contains(collapsed, "old code") || strings.Contains(collapsed, "new code") {
		t.Error("collapsed Edit should not show diff content")
	}

	// Expanded: should show path and diff
	expanded := renderToolInput(block, true, 80, "", nil)
	plainExpanded := stripANSI(expanded)
	if !strings.Contains(plainExpanded, "/tmp/test.go") {
		t.Error("expanded Edit should show file path")
	}
	if !strings.Contains(plainExpanded, "old code") {
		t.Error("expanded Edit should show old string")
	}
	if !strings.Contains(plainExpanded, "new code") {
		t.Error("expanded Edit should show new string")
	}
}

func TestRenderMarkdown_Plain(t *testing.T) {
	output := RenderMarkdown("Hello **world**", 80)
	if output == "" {
		t.Error("expected non-empty markdown output")
	}
	if !strings.Contains(output, "world") {
		t.Error("markdown output should contain the text")
	}
}

func TestBashBriefParam_AlwaysShowsCommand(t *testing.T) {
	block := session.Block{
		Type:      session.BlockToolUse,
		ToolName:  "Bash",
		ToolInput: map[string]interface{}{"command": "git status", "description": "Show git status"},
	}

	brief := toolBriefParam(block, "")
	if brief != "git status" {
		t.Errorf("expected command 'git status', got %q", brief)
	}
}

func TestBashBriefParam_MultilineCommand(t *testing.T) {
	block := session.Block{
		Type:      session.BlockToolUse,
		ToolName:  "Bash",
		ToolInput: map[string]interface{}{"command": "echo hello\necho world"},
	}

	brief := toolBriefParam(block, "")
	if brief != "echo hello" {
		t.Errorf("expected first line 'echo hello', got %q", brief)
	}
}

func TestFormatDuration(t *testing.T) {
	tests := []struct {
		d    time.Duration
		want string
	}{
		{500 * time.Millisecond, "500ms"},
		{5 * time.Second, "5s"},
		{5500 * time.Millisecond, "5.5s"},
		{2*time.Minute + 15*time.Second, "2m 15s"},
		{3 * time.Minute, "3m"},
	}
	for _, tt := range tests {
		got := formatDuration(tt.d)
		if got != tt.want {
			t.Errorf("formatDuration(%v) = %q, want %q", tt.d, got, tt.want)
		}
	}
}

func TestRenderDuration_ContainsVerb(t *testing.T) {
	output := renderDuration(2*time.Minute+15*time.Second, 1)
	plain := stripANSI(output)
	if !strings.Contains(plain, "for 2m 15s") {
		t.Errorf("expected duration text, got %q", plain)
	}
	if !strings.Contains(plain, "*") {
		t.Error("expected * prefix")
	}
}

func TestRenderTurn_ShowsDuration(t *testing.T) {
	turn := session.Turn{
		Number:   1,
		UserText: "explain this",
		Duration: 2*time.Minute + 15*time.Second,
		Blocks: []session.Block{
			{Type: session.BlockThinking, Text: "Let me think..."},
			{Type: session.BlockText, Text: "Here's my explanation."},
		},
	}

	output := RenderTurn(turn, false, 80, "")
	plain := stripANSI(output)
	if !strings.Contains(plain, "for 2m 15s") {
		t.Error("turn with thinking + duration should show duration")
	}
}

func TestRenderTurn_DurationAlwaysAtEnd(t *testing.T) {
	turn := session.Turn{
		Number:   1,
		UserText: "hello",
		Duration: 5 * time.Second,
		Blocks: []session.Block{
			{Type: session.BlockText, Text: "Hi there!"},
		},
	}

	output := RenderTurn(turn, false, 80, "")
	plain := stripANSI(output)
	if !strings.Contains(plain, "for 5s") {
		t.Error("turn with duration should show duration at end")
	}
	// Duration should be after the content
	hiIdx := strings.Index(plain, "Hi there!")
	durIdx := strings.Index(plain, "for 5s")
	if durIdx < hiIdx {
		t.Error("duration should appear after content")
	}
}

func TestHighlightDiffLine_NoLexer(t *testing.T) {
	result := highlightDiffLine("+ ", "some text", nil, "#1C3A2A", "#B8DB9A", 40)
	plain := stripANSI(result)
	if !strings.Contains(plain, "+ some text") {
		t.Errorf("fallback should contain text, got %q", plain)
	}
}

func TestHighlightDiffLine_GoFile(t *testing.T) {
	lexer := getLexer("test.go")
	result := highlightDiffLine("+ ", "func main() {", lexer, "#1C3A2A", "#B8DB9A", 60)
	if result == "" {
		t.Error("expected non-empty highlighted line")
	}
	plain := stripANSI(result)
	if !strings.Contains(plain, "func main()") {
		t.Errorf("highlighted line should contain code text, got %q", plain)
	}
}

func TestCtrlO_ExpandsEverything(t *testing.T) {
	turn := session.Turn{
		Number:   1,
		UserText: "hello",
		Blocks: []session.Block{
			{Type: session.BlockThinking, Text: "Deep thoughts here"},
			{Type: session.BlockText, Text: "response"},
		},
	}

	// Collapsed: thinking body hidden
	collapsed := RenderTurn(turn, false, 80, "")
	if strings.Contains(collapsed, "Deep thoughts here") {
		t.Error("collapsed turn should not show thinking body")
	}

	// Expanded: thinking body visible
	expanded := RenderTurn(turn, true, 80, "")
	if !strings.Contains(expanded, "Deep thoughts here") {
		t.Error("expanded turn should show thinking body")
	}
}

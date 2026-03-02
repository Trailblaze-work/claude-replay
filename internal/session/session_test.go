package session

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadSession_TurnSegmentation(t *testing.T) {
	// Create a temp JSONL file with known content
	dir := t.TempDir()
	path := filepath.Join(dir, "test-session.jsonl")

	lines := []string{
		// User message 1
		`{"type":"user","parentUuid":null,"uuid":"u1","sessionId":"s1","timestamp":"2026-02-13T12:00:00.000Z","cwd":"/test","gitBranch":"main","message":{"role":"user","content":"What is Go?"},"isSidechain":false,"slug":"test-slug"}`,
		// Assistant thinking
		`{"type":"assistant","parentUuid":"u1","uuid":"a1","sessionId":"s1","timestamp":"2026-02-13T12:00:01.000Z","message":{"model":"claude-opus-4-6","id":"msg_1","role":"assistant","content":[{"type":"thinking","thinking":"The user wants to know about Go."}]},"isSidechain":false}`,
		// Assistant text
		`{"type":"assistant","parentUuid":"a1","uuid":"a2","sessionId":"s1","timestamp":"2026-02-13T12:00:02.000Z","message":{"model":"claude-opus-4-6","id":"msg_1","role":"assistant","content":[{"type":"text","text":"Go is a programming language."}]},"isSidechain":false}`,
		// System turn duration
		`{"type":"system","parentUuid":"a2","uuid":"s1x","sessionId":"s1","timestamp":"2026-02-13T12:00:05.000Z","subtype":"turn_duration","durationMs":5000,"isSidechain":false}`,
		// User message 2
		`{"type":"user","parentUuid":"a2","uuid":"u2","sessionId":"s1","timestamp":"2026-02-13T12:00:10.000Z","cwd":"/test","gitBranch":"main","message":{"role":"user","content":"Show me an example"},"isSidechain":false}`,
		// Assistant with tool use
		`{"type":"assistant","parentUuid":"u2","uuid":"a3","sessionId":"s1","timestamp":"2026-02-13T12:00:11.000Z","message":{"model":"claude-opus-4-6","id":"msg_2","role":"assistant","content":[{"type":"tool_use","id":"tool1","name":"Write","input":{"file_path":"/tmp/main.go","content":"package main"}}]},"isSidechain":false}`,
		// Tool result
		`{"type":"user","parentUuid":"a3","uuid":"u3","sessionId":"s1","timestamp":"2026-02-13T12:00:12.000Z","message":{"role":"user","content":[{"tool_use_id":"tool1","type":"tool_result","content":"File written successfully"}]},"isSidechain":false}`,
		// Assistant follow-up text
		`{"type":"assistant","parentUuid":"u3","uuid":"a4","sessionId":"s1","timestamp":"2026-02-13T12:00:13.000Z","message":{"model":"claude-opus-4-6","id":"msg_3","role":"assistant","content":[{"type":"text","text":"I created the file for you."}]},"isSidechain":false}`,
		// System turn duration
		`{"type":"system","parentUuid":"a4","uuid":"s2x","sessionId":"s1","timestamp":"2026-02-13T12:00:15.000Z","subtype":"turn_duration","durationMs":5000,"isSidechain":false}`,
	}

	content := ""
	for _, l := range lines {
		content += l + "\n"
	}
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("writing test file: %v", err)
	}

	sess, err := LoadSession(path)
	if err != nil {
		t.Fatalf("LoadSession error: %v", err)
	}

	// Should have 2 turns
	if len(sess.Turns) != 2 {
		t.Fatalf("expected 2 turns, got %d", len(sess.Turns))
	}

	// Turn 1
	turn1 := sess.Turns[0]
	if turn1.UserText != "What is Go?" {
		t.Errorf("turn 1 user text: %s", turn1.UserText)
	}
	if turn1.Number != 1 {
		t.Errorf("turn 1 number: %d", turn1.Number)
	}
	if len(turn1.Blocks) != 2 { // thinking + text
		t.Errorf("turn 1 blocks: expected 2, got %d", len(turn1.Blocks))
	}
	if turn1.Blocks[0].Type != BlockThinking {
		t.Errorf("turn 1 block 0: expected thinking, got %d", turn1.Blocks[0].Type)
	}
	if turn1.Blocks[1].Type != BlockText {
		t.Errorf("turn 1 block 1: expected text, got %d", turn1.Blocks[1].Type)
	}
	if turn1.Duration.Milliseconds() != 5000 {
		t.Errorf("turn 1 duration: %v", turn1.Duration)
	}

	// Turn 2
	turn2 := sess.Turns[1]
	if turn2.UserText != "Show me an example" {
		t.Errorf("turn 2 user text: %s", turn2.UserText)
	}
	if len(turn2.Blocks) != 3 { // tool_use + tool_result + text
		t.Errorf("turn 2 blocks: expected 3, got %d", len(turn2.Blocks))
	}
	if turn2.Blocks[0].Type != BlockToolUse {
		t.Errorf("turn 2 block 0: expected tool_use, got %d", turn2.Blocks[0].Type)
	}
	if turn2.Blocks[0].ToolName != "Write" {
		t.Errorf("turn 2 block 0 tool: %s", turn2.Blocks[0].ToolName)
	}
	if turn2.Blocks[1].Type != BlockToolResult {
		t.Errorf("turn 2 block 1: expected tool_result, got %d", turn2.Blocks[1].Type)
	}
	if turn2.Blocks[1].Text != "File written successfully" {
		t.Errorf("turn 2 block 1 text: %s", turn2.Blocks[1].Text)
	}
	if turn2.Blocks[2].Type != BlockText {
		t.Errorf("turn 2 block 2: expected text, got %d", turn2.Blocks[2].Type)
	}

	// Session metadata
	if sess.ID != "s1" {
		t.Errorf("session ID: %s", sess.ID)
	}
	if sess.Slug != "test-slug" {
		t.Errorf("session slug: %s", sess.Slug)
	}
	if sess.Model != "claude-opus-4-6" {
		t.Errorf("session model: %s", sess.Model)
	}
	if sess.CWD != "/test" {
		t.Errorf("session cwd: %s", sess.CWD)
	}
	if sess.GitBranch != "main" {
		t.Errorf("session git branch: %s", sess.GitBranch)
	}
}

func TestLoadSession_CommandMessage(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "cmd-session.jsonl")

	lines := []string{
		// Regular user message
		`{"type":"user","parentUuid":null,"uuid":"u1","sessionId":"s1","timestamp":"2026-02-13T12:00:00.000Z","cwd":"/test","gitBranch":"main","message":{"role":"user","content":"hello"},"isSidechain":false}`,
		// Assistant response
		`{"type":"assistant","parentUuid":"u1","uuid":"a1","sessionId":"s1","timestamp":"2026-02-13T12:00:01.000Z","message":{"model":"claude-opus-4-6","id":"msg_1","role":"assistant","content":[{"type":"text","text":"hi there"}]},"isSidechain":false}`,
		// Command message (slash command invocation)
		`{"type":"user","parentUuid":"a1","uuid":"u2","sessionId":"s1","timestamp":"2026-02-13T12:01:00.000Z","cwd":"/test","gitBranch":"main","message":{"role":"user","content":"<command-message>session-trail:backfill</command-message>\n<command-name>/session-trail:backfill</command-name>"},"isSidechain":false}`,
		// Meta message (expanded skill prompt) - should be skipped
		`{"type":"user","parentUuid":"u2","uuid":"u3","sessionId":"s1","timestamp":"2026-02-13T12:01:00.000Z","cwd":"/test","gitBranch":"main","message":{"role":"user","content":[{"type":"text","text":"Base directory for this skill: /path/to/skill\n\nRun the backfill script:\nbash script.sh"}]},"isMeta":true,"isSidechain":false}`,
		// Assistant response to the command
		`{"type":"assistant","parentUuid":"u3","uuid":"a2","sessionId":"s1","timestamp":"2026-02-13T12:01:01.000Z","message":{"model":"claude-opus-4-6","id":"msg_2","role":"assistant","content":[{"type":"text","text":"Running the backfill..."}]},"isSidechain":false}`,
	}

	content := ""
	for _, l := range lines {
		content += l + "\n"
	}
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("writing test file: %v", err)
	}

	sess, err := LoadSession(path)
	if err != nil {
		t.Fatalf("LoadSession error: %v", err)
	}

	// Should have 2 turns: "hello" and "/session-trail:backfill"
	// The meta message should NOT create a third turn
	if len(sess.Turns) != 2 {
		t.Fatalf("expected 2 turns, got %d", len(sess.Turns))
	}

	// Turn 1: regular message
	if sess.Turns[0].UserText != "hello" {
		t.Errorf("turn 1 user text: got %q, want %q", sess.Turns[0].UserText, "hello")
	}

	// Turn 2: command should be rendered as the clean command name
	if sess.Turns[1].UserText != "/session-trail:backfill" {
		t.Errorf("turn 2 user text: got %q, want %q", sess.Turns[1].UserText, "/session-trail:backfill")
	}

	// Turn 2 should have the assistant's response block
	if len(sess.Turns[1].Blocks) != 1 {
		t.Errorf("turn 2 blocks: expected 1, got %d", len(sess.Turns[1].Blocks))
	}
	if sess.Turns[1].Blocks[0].Type != BlockText {
		t.Errorf("turn 2 block 0: expected text, got %d", sess.Turns[1].Blocks[0].Type)
	}
}

func TestLoadSession_MetaMessageOnly(t *testing.T) {
	// Test that a meta message without a preceding command doesn't create a turn
	dir := t.TempDir()
	path := filepath.Join(dir, "meta-only.jsonl")

	lines := []string{
		`{"type":"user","parentUuid":null,"uuid":"u1","sessionId":"s1","timestamp":"2026-02-13T12:00:00.000Z","cwd":"/test","message":{"role":"user","content":"hello"},"isSidechain":false}`,
		`{"type":"assistant","parentUuid":"u1","uuid":"a1","sessionId":"s1","timestamp":"2026-02-13T12:00:01.000Z","message":{"model":"claude-opus-4-6","id":"msg_1","role":"assistant","content":[{"type":"text","text":"hi"}]},"isSidechain":false}`,
		// Meta message without a preceding command message
		`{"type":"user","parentUuid":"a1","uuid":"u2","sessionId":"s1","timestamp":"2026-02-13T12:01:00.000Z","cwd":"/test","message":{"role":"user","content":[{"type":"text","text":"injected prompt"}]},"isMeta":true,"isSidechain":false}`,
	}

	content := ""
	for _, l := range lines {
		content += l + "\n"
	}
	os.WriteFile(path, []byte(content), 0644)

	sess, err := LoadSession(path)
	if err != nil {
		t.Fatalf("LoadSession error: %v", err)
	}

	// Only 1 turn - the meta message is skipped
	if len(sess.Turns) != 1 {
		t.Fatalf("expected 1 turn, got %d", len(sess.Turns))
	}
	if sess.Turns[0].UserText != "hello" {
		t.Errorf("turn 1 user text: got %q, want %q", sess.Turns[0].UserText, "hello")
	}
}

func TestLoadSession_EmptyFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "empty.jsonl")
	os.WriteFile(path, []byte(""), 0644)

	_, err := LoadSession(path)
	if err == nil {
		t.Error("expected error for empty file")
	}
}

func TestLoadSession_ShellEscapeCommand(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "shell-escape.jsonl")

	lines := []string{
		// Regular user message
		`{"type":"user","parentUuid":null,"uuid":"u1","sessionId":"s1","timestamp":"2026-02-13T12:00:00.000Z","cwd":"/test","gitBranch":"main","message":{"role":"user","content":"hello"},"isSidechain":false}`,
		// Assistant response
		`{"type":"assistant","parentUuid":"u1","uuid":"a1","sessionId":"s1","timestamp":"2026-02-13T12:00:01.000Z","message":{"model":"claude-opus-4-6","id":"msg_1","role":"assistant","content":[{"type":"text","text":"hi there"}]},"isSidechain":false}`,
		// Local command caveat (meta, should be skipped)
		`{"type":"user","parentUuid":"a1","uuid":"u2","sessionId":"s1","timestamp":"2026-02-13T12:01:00.000Z","cwd":"/test","gitBranch":"main","message":{"role":"user","content":"<local-command-caveat>Caveat: messages below were generated by running local commands.</local-command-caveat>"},"isMeta":true,"isSidechain":false}`,
		// Shell escape input (!git push)
		`{"type":"user","parentUuid":"u2","uuid":"u3","sessionId":"s1","timestamp":"2026-02-13T12:01:01.000Z","cwd":"/test","gitBranch":"main","message":{"role":"user","content":"<bash-input>git push</bash-input>"},"isSidechain":false}`,
		// Shell escape output
		`{"type":"user","parentUuid":"u3","uuid":"u4","sessionId":"s1","timestamp":"2026-02-13T12:01:02.000Z","cwd":"/test","gitBranch":"main","message":{"role":"user","content":"<bash-stdout>Everything up-to-date</bash-stdout><bash-stderr></bash-stderr>"},"isSidechain":false}`,
	}

	content := ""
	for _, l := range lines {
		content += l + "\n"
	}
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("writing test file: %v", err)
	}

	sess, err := LoadSession(path)
	if err != nil {
		t.Fatalf("LoadSession error: %v", err)
	}

	// Should have 2 turns: "hello" and "!git push"
	if len(sess.Turns) != 2 {
		t.Fatalf("expected 2 turns, got %d", len(sess.Turns))
	}

	// Turn 1: regular message
	if sess.Turns[0].UserText != "hello" {
		t.Errorf("turn 1 user text: got %q, want %q", sess.Turns[0].UserText, "hello")
	}

	// Turn 2: shell escape command
	if sess.Turns[1].UserText != "!git push" {
		t.Errorf("turn 2 user text: got %q, want %q", sess.Turns[1].UserText, "!git push")
	}

	// Turn 2 should have the output as a text block
	if len(sess.Turns[1].Blocks) != 1 {
		t.Fatalf("turn 2 blocks: expected 1, got %d", len(sess.Turns[1].Blocks))
	}
	if sess.Turns[1].Blocks[0].Type != BlockText {
		t.Errorf("turn 2 block 0: expected text, got %d", sess.Turns[1].Blocks[0].Type)
	}
	if sess.Turns[1].Blocks[0].Text != "Everything up-to-date" {
		t.Errorf("turn 2 block 0 text: got %q, want %q", sess.Turns[1].Blocks[0].Text, "Everything up-to-date")
	}
}

func TestLoadSession_ShellEscapeWithStderr(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "shell-stderr.jsonl")

	lines := []string{
		`{"type":"user","parentUuid":null,"uuid":"u1","sessionId":"s1","timestamp":"2026-02-13T12:00:00.000Z","cwd":"/test","message":{"role":"user","content":"<bash-input>git pull</bash-input>"},"isSidechain":false}`,
		`{"type":"user","parentUuid":"u1","uuid":"u2","sessionId":"s1","timestamp":"2026-02-13T12:00:01.000Z","cwd":"/test","message":{"role":"user","content":"<bash-stdout>Already up to date.</bash-stdout><bash-stderr>From github.com:user/repo\n * branch main -> FETCH_HEAD</bash-stderr>"},"isSidechain":false}`,
	}

	content := ""
	for _, l := range lines {
		content += l + "\n"
	}
	os.WriteFile(path, []byte(content), 0644)

	sess, err := LoadSession(path)
	if err != nil {
		t.Fatalf("LoadSession error: %v", err)
	}

	if len(sess.Turns) != 1 {
		t.Fatalf("expected 1 turn, got %d", len(sess.Turns))
	}

	if sess.Turns[0].UserText != "!git pull" {
		t.Errorf("user text: got %q, want %q", sess.Turns[0].UserText, "!git pull")
	}

	// Should combine stdout and stderr
	if len(sess.Turns[0].Blocks) != 1 {
		t.Fatalf("expected 1 block, got %d", len(sess.Turns[0].Blocks))
	}
	expected := "Already up to date.\nFrom github.com:user/repo\n * branch main -> FETCH_HEAD"
	if sess.Turns[0].Blocks[0].Text != expected {
		t.Errorf("block text: got %q, want %q", sess.Turns[0].Blocks[0].Text, expected)
	}
}

func TestLoadSession_ShellEscapeEmptyOutput(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "shell-empty.jsonl")

	lines := []string{
		`{"type":"user","parentUuid":null,"uuid":"u1","sessionId":"s1","timestamp":"2026-02-13T12:00:00.000Z","cwd":"/test","message":{"role":"user","content":"<bash-input>true</bash-input>"},"isSidechain":false}`,
		`{"type":"user","parentUuid":"u1","uuid":"u2","sessionId":"s1","timestamp":"2026-02-13T12:00:01.000Z","cwd":"/test","message":{"role":"user","content":"<bash-stdout></bash-stdout><bash-stderr></bash-stderr>"},"isSidechain":false}`,
	}

	content := ""
	for _, l := range lines {
		content += l + "\n"
	}
	os.WriteFile(path, []byte(content), 0644)

	sess, err := LoadSession(path)
	if err != nil {
		t.Fatalf("LoadSession error: %v", err)
	}

	if len(sess.Turns) != 1 {
		t.Fatalf("expected 1 turn, got %d", len(sess.Turns))
	}
	if sess.Turns[0].UserText != "!true" {
		t.Errorf("user text: got %q, want %q", sess.Turns[0].UserText, "!true")
	}
	// Empty output should not create any blocks
	if len(sess.Turns[0].Blocks) != 0 {
		t.Errorf("expected 0 blocks for empty output, got %d", len(sess.Turns[0].Blocks))
	}
}

// --- extractToolResultContent tests ---

func TestExtractToolResultContent_String(t *testing.T) {
	raw := []byte(`"file contents here"`)
	got := extractToolResultContent(raw)
	if got != "file contents here" {
		t.Errorf("got %q, want %q", got, "file contents here")
	}
}

func TestExtractToolResultContent_Array(t *testing.T) {
	raw := []byte(`[{"type":"text","text":"block one"},{"type":"text","text":"block two"}]`)
	got := extractToolResultContent(raw)
	if got != "block one\nblock two" {
		t.Errorf("got %q, want %q", got, "block one\nblock two")
	}
}

func TestExtractToolResultContent_Empty(t *testing.T) {
	got := extractToolResultContent(nil)
	if got != "" {
		t.Errorf("got %q, want empty string", got)
	}

	got = extractToolResultContent([]byte{})
	if got != "" {
		t.Errorf("got %q for empty slice, want empty string", got)
	}
}

func TestExtractToolResultContent_InvalidJSON(t *testing.T) {
	raw := []byte(`not json at all`)
	got := extractToolResultContent(raw)
	// Falls back to returning raw string
	if got != "not json at all" {
		t.Errorf("got %q, want %q", got, "not json at all")
	}
}

func TestLoadSession_ToolResultError(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "error.jsonl")

	lines := `{"type":"user","parentUuid":null,"uuid":"u1","sessionId":"s1","timestamp":"2026-02-13T12:00:00.000Z","message":{"role":"user","content":"run something"},"isSidechain":false}
{"type":"assistant","parentUuid":"u1","uuid":"a1","sessionId":"s1","timestamp":"2026-02-13T12:00:01.000Z","message":{"model":"claude-opus-4-6","id":"msg_1","role":"assistant","content":[{"type":"tool_use","id":"t1","name":"Bash","input":{"command":"false"}}]},"isSidechain":false}
{"type":"user","parentUuid":"a1","uuid":"u2","sessionId":"s1","timestamp":"2026-02-13T12:00:02.000Z","message":{"role":"user","content":[{"tool_use_id":"t1","type":"tool_result","content":"command not found","is_error":true}]},"isSidechain":false}
`
	os.WriteFile(path, []byte(lines), 0644)

	sess, err := LoadSession(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(sess.Turns) != 1 {
		t.Fatalf("expected 1 turn, got %d", len(sess.Turns))
	}

	// Should have tool_use + tool_result (with error)
	blocks := sess.Turns[0].Blocks
	if len(blocks) != 2 {
		t.Fatalf("expected 2 blocks, got %d", len(blocks))
	}
	if !blocks[1].IsError {
		t.Error("expected tool result to be an error")
	}
}

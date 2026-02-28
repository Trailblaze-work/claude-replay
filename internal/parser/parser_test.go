package parser

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestParse_UserRecord(t *testing.T) {
	input := `{"type":"user","parentUuid":null,"uuid":"abc-123","sessionId":"sess-1","timestamp":"2026-02-13T12:18:20.348Z","cwd":"/Users/test","gitBranch":"main","message":{"role":"user","content":"hello world"},"isSidechain":false}`

	records, err := Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(records) != 1 {
		t.Fatalf("expected 1 record, got %d", len(records))
	}

	rec := records[0]
	if rec.Type != RecordTypeUser {
		t.Errorf("expected type user, got %s", rec.Type)
	}
	if rec.UUID != "abc-123" {
		t.Errorf("expected uuid abc-123, got %s", rec.UUID)
	}

	msg, err := rec.ParseUserMessage()
	if err != nil {
		t.Fatalf("ParseUserMessage error: %v", err)
	}
	if msg.UserText() != "hello world" {
		t.Errorf("expected 'hello world', got '%s'", msg.UserText())
	}
	if msg.IsToolResults() {
		t.Error("expected not tool results")
	}
}

func TestParse_AssistantRecord(t *testing.T) {
	input := `{"type":"assistant","parentUuid":"abc-123","uuid":"def-456","sessionId":"sess-1","timestamp":"2026-02-13T12:18:22.000Z","message":{"model":"claude-opus-4-6","id":"msg_001","role":"assistant","content":[{"type":"text","text":"Hello! How can I help?"}]},"isSidechain":false}`

	records, err := Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(records) != 1 {
		t.Fatalf("expected 1 record, got %d", len(records))
	}

	msg, err := records[0].ParseAssistantMessage()
	if err != nil {
		t.Fatalf("ParseAssistantMessage error: %v", err)
	}
	if msg.Model != "claude-opus-4-6" {
		t.Errorf("expected model claude-opus-4-6, got %s", msg.Model)
	}
	if len(msg.Content) != 1 || msg.Content[0].Text != "Hello! How can I help?" {
		t.Error("unexpected content")
	}
}

func TestParse_FiltersProgressAndSnapshot(t *testing.T) {
	input := `{"type":"progress","parentUuid":"a","uuid":"b","timestamp":"2026-02-13T12:18:22.000Z","isSidechain":false}
{"type":"file-history-snapshot","messageId":"c","isSidechain":false}
{"type":"user","parentUuid":null,"uuid":"d","sessionId":"s","timestamp":"2026-02-13T12:18:22.000Z","message":{"role":"user","content":"test"},"isSidechain":false}`

	records, err := Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(records) != 1 {
		t.Fatalf("expected 1 record after filtering, got %d", len(records))
	}
	if records[0].Type != RecordTypeUser {
		t.Errorf("expected user record, got %s", records[0].Type)
	}
}

func TestParse_FiltersSidechain(t *testing.T) {
	input := `{"type":"user","parentUuid":null,"uuid":"a","sessionId":"s","timestamp":"2026-02-13T12:18:22.000Z","message":{"role":"user","content":"main"},"isSidechain":false}
{"type":"user","parentUuid":null,"uuid":"b","sessionId":"s","timestamp":"2026-02-13T12:18:22.000Z","message":{"role":"user","content":"side"},"isSidechain":true}`

	records, err := Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(records) != 1 {
		t.Fatalf("expected 1 record after sidechain filter, got %d", len(records))
	}
}

func TestParse_ToolResults(t *testing.T) {
	input := `{"type":"user","parentUuid":"x","uuid":"y","sessionId":"s","timestamp":"2026-02-13T12:18:22.000Z","message":{"role":"user","content":[{"tool_use_id":"tool1","type":"tool_result","content":"file contents here","is_error":false}]},"isSidechain":false}`

	records, err := Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	msg, err := records[0].ParseUserMessage()
	if err != nil {
		t.Fatalf("ParseUserMessage error: %v", err)
	}
	if !msg.IsToolResults() {
		t.Error("expected tool results")
	}

	results, err := msg.ParseToolResults()
	if err != nil {
		t.Fatalf("ParseToolResults error: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 tool result, got %d", len(results))
	}
	if results[0].ToolUseID != "tool1" {
		t.Errorf("expected tool1, got %s", results[0].ToolUseID)
	}
}

func TestParse_ThinkingBlock(t *testing.T) {
	input := `{"type":"assistant","parentUuid":"a","uuid":"b","sessionId":"s","timestamp":"2026-02-13T12:18:22.000Z","message":{"model":"claude-opus-4-6","id":"msg_002","role":"assistant","content":[{"type":"thinking","thinking":"Let me think about this...","signature":"sig123"}]},"isSidechain":false}`

	records, err := Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	msg, err := records[0].ParseAssistantMessage()
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	if len(msg.Content) != 1 {
		t.Fatalf("expected 1 content block, got %d", len(msg.Content))
	}
	if msg.Content[0].Type != "thinking" {
		t.Errorf("expected thinking, got %s", msg.Content[0].Type)
	}
	if msg.Content[0].Thinking != "Let me think about this..." {
		t.Errorf("unexpected thinking text: %s", msg.Content[0].Thinking)
	}
}

func TestParse_ToolUseBlock(t *testing.T) {
	input := `{"type":"assistant","parentUuid":"a","uuid":"b","sessionId":"s","timestamp":"2026-02-13T12:18:22.000Z","message":{"model":"claude-opus-4-6","id":"msg_003","role":"assistant","content":[{"type":"tool_use","id":"toolu_123","name":"Bash","input":{"command":"ls -la","description":"List files"}}]},"isSidechain":false}`

	records, err := Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	msg, err := records[0].ParseAssistantMessage()
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	cb := msg.Content[0]
	if cb.Type != "tool_use" {
		t.Errorf("expected tool_use, got %s", cb.Type)
	}
	if cb.Name != "Bash" {
		t.Errorf("expected Bash, got %s", cb.Name)
	}
	if cb.ID != "toolu_123" {
		t.Errorf("expected toolu_123, got %s", cb.ID)
	}
}

func TestParse_SystemRecord(t *testing.T) {
	input := `{"type":"system","parentUuid":"a","uuid":"b","sessionId":"s","timestamp":"2026-02-13T12:18:22.000Z","subtype":"turn_duration","durationMs":39540,"isSidechain":false}`

	records, err := Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(records) != 1 {
		t.Fatalf("expected 1 record, got %d", len(records))
	}
	if records[0].Subtype != "turn_duration" {
		t.Errorf("expected turn_duration, got %s", records[0].Subtype)
	}
	if records[0].DurationMs != 39540 {
		t.Errorf("expected 39540ms, got %f", records[0].DurationMs)
	}
}

func TestParse_SkipsMalformedLines(t *testing.T) {
	input := `not json at all
{"type":"user","parentUuid":null,"uuid":"a","sessionId":"s","timestamp":"2026-02-13T12:18:22.000Z","message":{"role":"user","content":"valid"},"isSidechain":false}
{broken json`

	records, err := Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(records) != 1 {
		t.Fatalf("expected 1 valid record, got %d", len(records))
	}
}

// --- QuickScan tests ---

func TestQuickScan_BasicMetadata(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "session.jsonl")

	content := `{"type":"user","parentUuid":null,"uuid":"u1","sessionId":"s1","timestamp":"2026-02-13T12:00:00.000Z","slug":"my-slug","message":{"role":"user","content":"hello"},"isSidechain":false}
{"type":"assistant","parentUuid":"u1","uuid":"a1","sessionId":"s1","timestamp":"2026-02-13T12:00:05.000Z","message":{"model":"claude-opus-4-6","id":"msg_1","role":"assistant","content":[{"type":"text","text":"hi"}]},"isSidechain":false}
`
	os.WriteFile(path, []byte(content), 0644)

	slug, model, firstTime, lastTime, turnCount, err := QuickScan(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if slug != "my-slug" {
		t.Errorf("slug: got %q, want %q", slug, "my-slug")
	}
	if model != "claude-opus-4-6" {
		t.Errorf("model: got %q, want %q", model, "claude-opus-4-6")
	}
	if firstTime != "2026-02-13T12:00:00.000Z" {
		t.Errorf("firstTime: got %q", firstTime)
	}
	if lastTime != "2026-02-13T12:00:05.000Z" {
		t.Errorf("lastTime: got %q", lastTime)
	}
	if turnCount != 1 {
		t.Errorf("turnCount: got %d, want 1", turnCount)
	}
}

func TestQuickScan_EmptyFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "empty.jsonl")
	os.WriteFile(path, []byte(""), 0644)

	slug, model, firstTime, lastTime, turnCount, err := QuickScan(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if slug != "" || model != "" || firstTime != "" || lastTime != "" {
		t.Errorf("expected empty strings, got slug=%q model=%q first=%q last=%q", slug, model, firstTime, lastTime)
	}
	if turnCount != 0 {
		t.Errorf("turnCount: got %d, want 0", turnCount)
	}
}

func TestQuickScan_OnlyAssistant(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "assistant-only.jsonl")

	content := `{"type":"assistant","parentUuid":"x","uuid":"a1","sessionId":"s1","timestamp":"2026-02-13T12:00:00.000Z","message":{"model":"claude-opus-4-6","id":"msg_1","role":"assistant","content":[{"type":"text","text":"hi"}]},"isSidechain":false}
`
	os.WriteFile(path, []byte(content), 0644)

	_, _, _, _, turnCount, err := QuickScan(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if turnCount != 0 {
		t.Errorf("turnCount: got %d, want 0 (no user messages)", turnCount)
	}
}

func TestQuickScan_MultipleTurns(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "multi.jsonl")

	content := `{"type":"user","parentUuid":null,"uuid":"u1","sessionId":"s1","timestamp":"2026-02-13T12:00:00.000Z","message":{"role":"user","content":"first"},"isSidechain":false}
{"type":"assistant","parentUuid":"u1","uuid":"a1","sessionId":"s1","timestamp":"2026-02-13T12:00:01.000Z","message":{"model":"claude-opus-4-6","id":"msg_1","role":"assistant","content":[{"type":"text","text":"r1"}]},"isSidechain":false}
{"type":"user","parentUuid":"a1","uuid":"u2","sessionId":"s1","timestamp":"2026-02-13T12:01:00.000Z","message":{"role":"user","content":"second"},"isSidechain":false}
{"type":"assistant","parentUuid":"u2","uuid":"a2","sessionId":"s1","timestamp":"2026-02-13T12:01:01.000Z","message":{"model":"claude-opus-4-6","id":"msg_2","role":"assistant","content":[{"type":"text","text":"r2"}]},"isSidechain":false}
{"type":"user","parentUuid":"a2","uuid":"u3","sessionId":"s1","timestamp":"2026-02-13T12:02:00.000Z","message":{"role":"user","content":"third"},"isSidechain":false}
`
	os.WriteFile(path, []byte(content), 0644)

	_, _, _, _, turnCount, err := QuickScan(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if turnCount != 3 {
		t.Errorf("turnCount: got %d, want 3", turnCount)
	}
}

func TestQuickScan_MalformedLines(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "mixed.jsonl")

	content := `not valid json
{"type":"user","parentUuid":null,"uuid":"u1","sessionId":"s1","timestamp":"2026-02-13T12:00:00.000Z","slug":"recovered","message":{"role":"user","content":"hello"},"isSidechain":false}
{broken
{"type":"assistant","parentUuid":"u1","uuid":"a1","sessionId":"s1","timestamp":"2026-02-13T12:00:05.000Z","message":{"model":"claude-opus-4-6","id":"msg_1","role":"assistant","content":[{"type":"text","text":"hi"}]},"isSidechain":false}
`
	os.WriteFile(path, []byte(content), 0644)

	slug, model, _, _, turnCount, err := QuickScan(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if slug != "recovered" {
		t.Errorf("slug: got %q, want %q", slug, "recovered")
	}
	if model != "claude-opus-4-6" {
		t.Errorf("model: got %q, want %q", model, "claude-opus-4-6")
	}
	if turnCount != 1 {
		t.Errorf("turnCount: got %d, want 1", turnCount)
	}
}

package export

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/trailblaze/claude-replay/internal/session"
)

func TestGenerateCast(t *testing.T) {
	dir := t.TempDir()
	output := filepath.Join(dir, "test.cast")

	sess := &session.Session{
		ID:        "test-session",
		Slug:      "test-slug",
		CWD:       "/test",
		GitBranch: "main",
		Model:     "claude-opus-4-6",
		StartTime: time.Date(2026, 2, 13, 12, 0, 0, 0, time.UTC),
		EndTime:   time.Date(2026, 2, 13, 12, 5, 0, 0, time.UTC),
		Turns: []session.Turn{
			{
				Number:    1,
				UserText:  "Hello",
				Timestamp: time.Date(2026, 2, 13, 12, 0, 0, 0, time.UTC),
				Model:     "claude-opus-4-6",
				Blocks: []session.Block{
					{Type: session.BlockText, Text: "Hi there!"},
				},
			},
			{
				Number:    2,
				UserText:  "How are you?",
				Timestamp: time.Date(2026, 2, 13, 12, 1, 0, 0, time.UTC),
				Model:     "claude-opus-4-6",
				Blocks: []session.Block{
					{Type: session.BlockText, Text: "I'm doing well!"},
				},
			},
		},
	}

	opts := Options{
		TimingMode: TimingCompressed,
		Width:      80,
		Height:     24,
		Output:     output,
	}

	err := GenerateCast(sess, opts)
	if err != nil {
		t.Fatalf("GenerateCast error: %v", err)
	}

	// Read and validate the output
	data, err := os.ReadFile(output)
	if err != nil {
		t.Fatalf("reading output: %v", err)
	}

	lines := strings.Split(strings.TrimSpace(string(data)), "\n")
	if len(lines) != 3 { // 1 header + 2 frames
		t.Fatalf("expected 3 lines, got %d", len(lines))
	}

	// Validate header
	var header struct {
		Version int `json:"version"`
		Width   int `json:"width"`
		Height  int `json:"height"`
	}
	if err := json.Unmarshal([]byte(lines[0]), &header); err != nil {
		t.Fatalf("parsing header: %v", err)
	}
	if header.Version != 2 {
		t.Errorf("expected version 2, got %d", header.Version)
	}
	if header.Width != 80 {
		t.Errorf("expected width 80, got %d", header.Width)
	}

	// Validate event format
	var event []interface{}
	if err := json.Unmarshal([]byte(lines[1]), &event); err != nil {
		t.Fatalf("parsing event: %v", err)
	}
	if len(event) != 3 {
		t.Fatalf("expected 3 event fields, got %d", len(event))
	}
	if event[1] != "o" {
		t.Errorf("expected event type 'o', got %v", event[1])
	}
}

func TestRenderFrame_Basic(t *testing.T) {
	sess := &session.Session{
		ID:        "test-session",
		Slug:      "my-slug",
		CWD:       "/home/user/project",
		GitBranch: "main",
		Model:     "claude-opus-4-6",
		Turns: []session.Turn{
			{
				Number:    1,
				UserText:  "Hello world",
				Timestamp: time.Date(2026, 2, 13, 12, 0, 0, 0, time.UTC),
				Model:     "claude-opus-4-6",
				Blocks: []session.Block{
					{Type: session.BlockText, Text: "Hi there!"},
				},
			},
		},
	}

	frame := RenderFrame(sess, 0, 80, 24)
	if frame == "" {
		t.Fatal("expected non-empty frame")
	}
	if !strings.Contains(frame, "Hello world") {
		t.Error("frame should contain user text")
	}
	if !strings.Contains(frame, "Hi there!") {
		t.Error("frame should contain assistant text")
	}
}

func TestRenderFrame_OutOfBounds(t *testing.T) {
	sess := &session.Session{
		Turns: []session.Turn{
			{Number: 1, UserText: "test"},
		},
	}

	if got := RenderFrame(sess, -1, 80, 24); got != "" {
		t.Errorf("negative index: got %q, want empty", got)
	}
	if got := RenderFrame(sess, 5, 80, 24); got != "" {
		t.Errorf("too-large index: got %q, want empty", got)
	}
}

func TestFormatCastInfo_ValidFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.cast")

	// Write a fake .cast file: 1 header line + 3 frame lines
	content := "{\"version\":2}\n[0.0,\"o\",\"frame1\"]\n[1.0,\"o\",\"frame2\"]\n[2.0,\"o\",\"frame3\"]\n"
	os.WriteFile(path, []byte(content), 0644)

	info := FormatCastInfo(path)
	if !strings.Contains(info, "3 frames") {
		t.Errorf("expected '3 frames' in info, got: %s", info)
	}
	if !strings.Contains(info, "test.cast") {
		t.Errorf("expected path in info, got: %s", info)
	}
}

func TestFormatCastInfo_MissingFile(t *testing.T) {
	got := FormatCastInfo("/nonexistent/file.cast")
	if got != "/nonexistent/file.cast" {
		t.Errorf("got %q, want path only", got)
	}
}

func TestFormatFileSize(t *testing.T) {
	tests := []struct {
		bytes    int64
		expected string
	}{
		{0, "0B"},
		{512, "512B"},
		{1024, "1KB"},
		{1536 * 1024, "1.5MB"},
		{2 * 1024 * 1024, "2.0MB"},
	}

	for _, tt := range tests {
		got := formatFileSize(tt.bytes)
		if got != tt.expected {
			t.Errorf("formatFileSize(%d) = %q, want %q", tt.bytes, got, tt.expected)
		}
	}
}

func TestDefaultOptions(t *testing.T) {
	opts := DefaultOptions()
	if opts.TimingMode != TimingCompressed {
		t.Errorf("TimingMode: got %q, want %q", opts.TimingMode, TimingCompressed)
	}
	if opts.Width != 120 {
		t.Errorf("Width: got %d, want 120", opts.Width)
	}
	if opts.Height != 40 {
		t.Errorf("Height: got %d, want 40", opts.Height)
	}
	if opts.Format != "cast" {
		t.Errorf("Format: got %q, want %q", opts.Format, "cast")
	}
}

func TestTurnDelay_ZeroDuration(t *testing.T) {
	// Realtime with 0 duration falls back to 2s
	opts := Options{TimingMode: TimingRealtime}
	delay := opts.TurnDelay(0, 1)
	if delay != 2*time.Second {
		t.Errorf("realtime 0 dur: got %v, want 2s", delay)
	}

	// Fast with 0 duration falls back to 1s
	opts.TimingMode = TimingFast
	delay = opts.TurnDelay(0, 1)
	if delay != time.Second {
		t.Errorf("fast 0 dur: got %v, want 1s", delay)
	}
}

func TestGenerateCast_FrameLineEndings(t *testing.T) {
	dir := t.TempDir()
	output := filepath.Join(dir, "test.cast")

	sess := &session.Session{
		ID:        "test-session",
		Slug:      "test-slug",
		CWD:       "/test",
		Model:     "claude-opus-4-6",
		StartTime: time.Date(2026, 2, 13, 12, 0, 0, 0, time.UTC),
		Turns: []session.Turn{
			{
				Number:    1,
				UserText:  "Hello",
				Timestamp: time.Date(2026, 2, 13, 12, 0, 0, 0, time.UTC),
				Model:     "claude-opus-4-6",
				Blocks: []session.Block{
					{Type: session.BlockText, Text: "Hi there!"},
				},
			},
		},
	}

	opts := Options{
		TimingMode: TimingInstant,
		Width:      80,
		Height:     24,
		Output:     output,
	}

	if err := GenerateCast(sess, opts); err != nil {
		t.Fatalf("GenerateCast error: %v", err)
	}

	data, err := os.ReadFile(output)
	if err != nil {
		t.Fatalf("reading output: %v", err)
	}

	// Parse the first frame event
	fileLines := strings.Split(strings.TrimSpace(string(data)), "\n")
	if len(fileLines) < 2 {
		t.Fatal("expected at least header + 1 frame")
	}

	var event []interface{}
	if err := json.Unmarshal([]byte(fileLines[1]), &event); err != nil {
		t.Fatalf("parsing event: %v", err)
	}

	frameContent := event[2].(string)

	// Frame content must use \r\n line endings for proper terminal rendering.
	// Bare \n causes diagonal text in asciinema players because VT100 LF
	// only moves the cursor down without returning to column 0.
	if strings.Contains(frameContent, "\n") && !strings.Contains(frameContent, "\r\n") {
		t.Error("frame content uses bare \\n; must use \\r\\n for proper terminal rendering")
	}

	// Verify no bare \n exists (every \n must be preceded by \r)
	for i, c := range frameContent {
		if c == '\n' && (i == 0 || frameContent[i-1] != '\r') {
			t.Errorf("bare \\n at position %d; all newlines must be \\r\\n", i)
			break
		}
	}
}

func TestGenerateCast_FirstFrameAtZero(t *testing.T) {
	dir := t.TempDir()
	output := filepath.Join(dir, "test.cast")

	sess := &session.Session{
		ID:        "test-session",
		Slug:      "test-slug",
		StartTime: time.Date(2026, 2, 13, 12, 0, 0, 0, time.UTC),
		Turns: []session.Turn{
			{
				Number:    1,
				UserText:  "Hello",
				Timestamp: time.Date(2026, 2, 13, 12, 0, 0, 0, time.UTC),
				Blocks:    []session.Block{{Type: session.BlockText, Text: "Hi"}},
			},
			{
				Number:    2,
				UserText:  "Bye",
				Timestamp: time.Date(2026, 2, 13, 12, 1, 0, 0, time.UTC),
				Blocks:    []session.Block{{Type: session.BlockText, Text: "Bye!"}},
			},
		},
	}

	opts := Options{
		TimingMode: TimingCompressed,
		Width:      80,
		Height:     24,
		Output:     output,
	}

	if err := GenerateCast(sess, opts); err != nil {
		t.Fatalf("GenerateCast error: %v", err)
	}

	data, err := os.ReadFile(output)
	if err != nil {
		t.Fatalf("reading output: %v", err)
	}

	fileLines := strings.Split(strings.TrimSpace(string(data)), "\n")
	if len(fileLines) < 2 {
		t.Fatal("expected at least header + 1 frame")
	}

	// First frame should start at time 0
	var event []interface{}
	if err := json.Unmarshal([]byte(fileLines[1]), &event); err != nil {
		t.Fatalf("parsing first event: %v", err)
	}
	timestamp := event[0].(float64)
	if timestamp != 0.0 {
		t.Errorf("first frame should start at time 0, got %.6f", timestamp)
	}

	// Second frame should be > 0
	if len(fileLines) >= 3 {
		var event2 []interface{}
		if err := json.Unmarshal([]byte(fileLines[2]), &event2); err != nil {
			t.Fatalf("parsing second event: %v", err)
		}
		ts2 := event2[0].(float64)
		if ts2 <= 0.0 {
			t.Errorf("second frame should be > 0, got %.6f", ts2)
		}
	}
}

func TestTimingModes(t *testing.T) {
	opts := Options{TimingMode: TimingCompressed}
	delay := opts.TurnDelay(10*time.Second, 1)
	if delay != 2*time.Second {
		t.Errorf("compressed: expected 2s, got %v", delay)
	}

	opts.TimingMode = TimingRealtime
	delay = opts.TurnDelay(10*time.Second, 1)
	if delay != 10*time.Second {
		t.Errorf("realtime: expected 10s, got %v", delay)
	}

	opts.TimingMode = TimingFast
	delay = opts.TurnDelay(10*time.Second, 1)
	if delay != 5*time.Second {
		t.Errorf("fast: expected 5s, got %v", delay)
	}

	opts.TimingMode = TimingInstant
	delay = opts.TurnDelay(10*time.Second, 1)
	if delay != 100*time.Millisecond {
		t.Errorf("instant: expected 100ms, got %v", delay)
	}
}

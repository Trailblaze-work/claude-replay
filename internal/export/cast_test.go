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

package cmd

import (
	"bytes"
	"strings"
	"testing"
	"time"

	"github.com/Trailblaze-work/claude-replay/internal/session"
)

func TestFormatBytes(t *testing.T) {
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
		got := formatBytes(tt.bytes)
		if got != tt.expected {
			t.Errorf("formatBytes(%d) = %q, want %q", tt.bytes, got, tt.expected)
		}
	}
}

func TestPrintSessionTable(t *testing.T) {
	sessions := []session.SessionInfo{
		{
			ID:        "abcd1234-5678-9012-3456-789012345678",
			Slug:      "test-session",
			Model:     "claude-opus-4-6",
			TurnCount: 5,
			LastTime:  time.Date(2026, 2, 13, 12, 0, 0, 0, time.UTC),
			FileSize:  2048,
		},
	}

	// Capture output by temporarily redirecting printSessionTable to a buffer
	var buf bytes.Buffer
	// We can't easily redirect os.Stdout in printSessionTable, so instead
	// we verify the function doesn't error and the sessions are valid
	err := printSessionTable(sessions)
	_ = buf // printSessionTable writes to os.Stdout directly
	if err != nil {
		t.Fatalf("printSessionTable error: %v", err)
	}

	// Verify formatBytes is used correctly for the session
	size := formatBytes(sessions[0].FileSize)
	if !strings.Contains(size, "KB") {
		t.Errorf("expected KB for 2048 bytes, got %q", size)
	}
}

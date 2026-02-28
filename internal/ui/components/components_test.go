package components

import (
	"strings"
	"testing"
	"time"
)

func TestRenderHeader_ContainsSlug(t *testing.T) {
	output := RenderHeader("my-test-slug", "/home/user/project", "main", 80)
	if !strings.Contains(output, "my-test-slug") {
		t.Error("header should contain the slug")
	}
	if !strings.Contains(output, "claude-replay") {
		t.Error("header should contain 'claude-replay' title")
	}
}

func TestRenderHeader_HidesHEADBranch(t *testing.T) {
	withHead := RenderHeader("slug", "/path", "HEAD", 80)
	withBranch := RenderHeader("slug", "/path", "feature-x", 80)

	if strings.Contains(withHead, "HEAD") {
		t.Error("header should suppress HEAD branch display")
	}
	if !strings.Contains(withBranch, "feature-x") {
		t.Error("header should show non-HEAD branch")
	}
}

func TestRenderTimeline_Boundaries(t *testing.T) {
	// First turn: bar should be mostly empty
	first := RenderTimeline(1, 10, 80)
	if first == "" {
		t.Fatal("expected non-empty timeline for first turn")
	}

	// Last turn: bar should be mostly filled
	last := RenderTimeline(10, 10, 80)
	if last == "" {
		t.Fatal("expected non-empty timeline for last turn")
	}

	// First should have fewer filled blocks than last
	firstFilled := strings.Count(first, "█")
	lastFilled := strings.Count(last, "█")
	if firstFilled >= lastFilled {
		t.Errorf("first turn filled (%d) should be < last turn filled (%d)", firstFilled, lastFilled)
	}
}

func TestRenderTimeline_ZeroTotal(t *testing.T) {
	got := RenderTimeline(0, 0, 80)
	if got != "" {
		t.Errorf("expected empty string for zero total, got %q", got)
	}
}

func TestFormatModelShort(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"claude-opus-4-6", "opus-4.6"},
		{"claude-sonnet-4-6", "sonnet-4.6"},
		{"claude-haiku-4-5-20251001", "haiku"},
		{"claude-opus-4-20250115", "opus"},
		{"claude-sonnet-4-20250115", "sonnet"},
		{"some-really-long-model-name-that-exceeds-twenty-chars", "some-really-long-mod…"},
		{"short", "short"},
	}

	for _, tt := range tests {
		got := formatModelShort(tt.input)
		if got != tt.expected {
			t.Errorf("formatModelShort(%q) = %q, want %q", tt.input, got, tt.expected)
		}
	}
}

func TestFormatDuration(t *testing.T) {
	tests := []struct {
		d        time.Duration
		expected string
	}{
		{0, "—"},
		{500 * time.Millisecond, "500ms"},
		{1500 * time.Millisecond, "1.5s"},
		{2*time.Minute + 30*time.Second, "2m30s"},
	}

	for _, tt := range tests {
		got := formatDuration(tt.d)
		if got != tt.expected {
			t.Errorf("formatDuration(%v) = %q, want %q", tt.d, got, tt.expected)
		}
	}
}

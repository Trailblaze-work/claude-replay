package export

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/trailblaze/claude-replay/internal/session"
)

// castHeader is the asciinema v2 header.
type castHeader struct {
	Version   int               `json:"version"`
	Width     int               `json:"width"`
	Height    int               `json:"height"`
	Timestamp int64             `json:"timestamp"`
	Title     string            `json:"title,omitempty"`
	Env       map[string]string `json:"env,omitempty"`
}

// GenerateCast creates an asciinema .cast file from a session.
func GenerateCast(sess *session.Session, opts Options) error {
	f, err := os.Create(opts.Output)
	if err != nil {
		return fmt.Errorf("creating output file: %w", err)
	}
	defer f.Close()

	// Write header
	title := sess.Slug
	if title == "" && len(sess.ID) > 8 {
		title = sess.ID[:8]
	}

	header := castHeader{
		Version:   2,
		Width:     opts.Width,
		Height:    opts.Height,
		Timestamp: sess.StartTime.Unix(),
		Title:     title,
		Env: map[string]string{
			"SHELL": "/bin/zsh",
			"TERM":  "xterm-256color",
		},
	}

	headerJSON, err := json.Marshal(header)
	if err != nil {
		return fmt.Errorf("marshaling header: %w", err)
	}
	fmt.Fprintf(f, "%s\n", headerJSON)

	// Generate frames
	var elapsed time.Duration

	for i := range sess.Turns {
		// Calculate delay
		var realDuration time.Duration
		if i > 0 {
			realDuration = sess.Turns[i].Timestamp.Sub(sess.Turns[i-1].Timestamp)
		}
		delay := opts.TurnDelay(realDuration, i)
		elapsed += delay

		// Render frame
		frame := RenderFrame(sess, i, opts.Width, opts.Height)

		// Clear screen + render
		output := "\033[2J\033[H" + frame

		// Write event: [time, "o", data]
		timestamp := float64(elapsed) / float64(time.Second)
		eventData, err := json.Marshal(output)
		if err != nil {
			continue
		}
		fmt.Fprintf(f, "[%.6f, \"o\", %s]\n", timestamp, eventData)

		// Add a small delay after the frame appears for readability
		if opts.TimingMode != TimingInstant {
			elapsed += 500 * time.Millisecond
		}
	}

	return nil
}

// ConvertToGif converts a .cast file to .gif using agg if available.
func ConvertToGif(castPath, gifPath string) error {
	// Check if agg is available
	return fmt.Errorf("GIF conversion requires 'agg' (https://github.com/asciinema/agg). Install with: cargo install agg")
}

// ConvertToMP4 converts a .gif to .mp4 using ffmpeg if available.
func ConvertToMP4(gifPath, mp4Path string) error {
	return fmt.Errorf("MP4 conversion requires 'ffmpeg'. Install with: brew install ffmpeg")
}

// FormatCastInfo returns info about a generated .cast file.
func FormatCastInfo(path string) string {
	info, err := os.Stat(path)
	if err != nil {
		return path
	}

	var lines int
	data, err := os.ReadFile(path)
	if err == nil {
		lines = strings.Count(string(data), "\n")
	}

	return fmt.Sprintf("%s (%d frames, %s)", path, lines-1, formatFileSize(info.Size()))
}

func formatFileSize(bytes int64) string {
	switch {
	case bytes >= 1024*1024:
		return fmt.Sprintf("%.1fMB", float64(bytes)/(1024*1024))
	case bytes >= 1024:
		return fmt.Sprintf("%.0fKB", float64(bytes)/1024)
	default:
		return fmt.Sprintf("%dB", bytes)
	}
}

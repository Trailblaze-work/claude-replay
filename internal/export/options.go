package export

import "time"

// TimingMode controls how timing is applied to exported frames.
type TimingMode string

const (
	TimingRealtime   TimingMode = "realtime"   // Use actual timestamps
	TimingCompressed TimingMode = "compressed"  // Fixed 2s between turns
	TimingFast       TimingMode = "fast"        // 2x speed of real timestamps
	TimingInstant    TimingMode = "instant"     // No delays
)

// Options configures the export.
type Options struct {
	TimingMode TimingMode
	Width      int
	Height     int
	Output     string
	Format     string // "cast", "gif", "mp4"
}

// DefaultOptions returns sensible defaults.
func DefaultOptions() Options {
	return Options{
		TimingMode: TimingCompressed,
		Width:      120,
		Height:     40,
		Format:     "cast",
	}
}

// TurnDelay calculates the delay before showing a turn.
func (o Options) TurnDelay(realDuration time.Duration, turnIndex int) time.Duration {
	switch o.TimingMode {
	case TimingRealtime:
		if realDuration > 0 {
			return realDuration
		}
		return 2 * time.Second
	case TimingCompressed:
		return 2 * time.Second
	case TimingFast:
		if realDuration > 0 {
			return realDuration / 2
		}
		return time.Second
	case TimingInstant:
		return 100 * time.Millisecond // minimal delay for readability
	default:
		return 2 * time.Second
	}
}

package segments

import (
	"time"
)

// TimecodeWithDuration represents a timecode and its duration
type TimecodeWithDuration struct {
	StartTime time.Duration
	Duration  time.Duration
}

// Segment represents a clip segment with timing and text
type Segment struct {
	StartTime time.Duration
	EndTime   time.Duration
	Text      string
}
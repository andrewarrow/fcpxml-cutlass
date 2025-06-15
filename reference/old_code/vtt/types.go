package vtt

import "time"

type Segment struct {
	StartTime time.Duration
	EndTime   time.Duration
	Text      string
}

type SilenceGap struct {
	Start    time.Duration
	End      time.Duration
	Duration time.Duration
}

// TimecodeWithDuration represents a timecode with its duration
type TimecodeWithDuration struct {
	Start    time.Duration
	Duration time.Duration
}

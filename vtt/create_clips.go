package vtt

import (
	"cutlass/fcp"
	"fmt"
	"strings"
	"time"
)

// CreateClipsFromTimecodesWithDuration creates clips with specified start times and durations
func CreateClipsFromTimecodesWithDuration(segments []Segment, timecodes []TimecodeWithDuration) ([]fcp.Clip, error) {
	var clips []fcp.Clip

	for i, timecode := range timecodes {
		startTime := timecode.Start
		endTime := startTime + timecode.Duration

		// Find all segments that overlap with this time range
		var clipText []string
		var firstSegmentText string
		found := false

		for _, segment := range segments {
			// Check if segment overlaps with our clip time range
			if segment.EndTime > startTime && segment.StartTime < endTime {
				clipText = append(clipText, segment.Text)
				if !found {
					firstSegmentText = segment.Text
					found = true
				}
			}
		}

		// If no segments found within the exact time range, find the closest segment
		if !found {
			var closestSegment *Segment
			minDistance := time.Duration(1<<63 - 1) // Max duration

			for _, segment := range segments {
				// Calculate distance from segment to our start time
				var distance time.Duration
				if segment.EndTime < startTime {
					distance = startTime - segment.EndTime
				} else if segment.StartTime > endTime {
					distance = segment.StartTime - endTime
				} else {
					distance = 0 // Overlaps
				}

				if distance < minDistance {
					minDistance = distance
					closestSegment = &segment
				}
			}

			if closestSegment != nil {
				clipText = append(clipText, closestSegment.Text)
				firstSegmentText = closestSegment.Text
			} else {
				return nil, fmt.Errorf("no VTT segment found near timecode %v", startTime)
			}
		}

		clip := fcp.Clip{
			StartTime:        startTime,
			EndTime:          endTime,
			Duration:         timecode.Duration,
			Text:             strings.Join(clipText, " "),
			FirstSegmentText: firstSegmentText,
			ClipNum:          i + 1,
		}

		clips = append(clips, clip)
	}

	return clips, nil
}

// CreateClipsFromTimecodes creates clips starting at specified timecodes
func CreateClipsFromTimecodes(segments []Segment, timecodes []time.Duration) ([]fcp.Clip, error) {
	var clips []fcp.Clip

	for i, startTime := range timecodes {
		// Find segments around this timecode
		var clipSegments []Segment
		var clipText []string

		// Find the first segment at or after the timecode
		for _, segment := range segments {
			if segment.StartTime >= startTime {
				clipSegments = append(clipSegments, segment)
				clipText = append(clipText, segment.Text)
				break
			}
		}

		if len(clipSegments) == 0 {
			// If no segment found at or after timecode, find the closest one before
			var closestSegment *Segment
			for _, segment := range segments {
				if segment.EndTime <= startTime {
					if closestSegment == nil || segment.StartTime > closestSegment.StartTime {
						closestSegment = &segment
					}
				}
			}
			if closestSegment != nil {
				clipSegments = append(clipSegments, *closestSegment)
				clipText = append(clipText, closestSegment.Text)
			} else {
				return nil, fmt.Errorf("no VTT segment found near timecode %v", startTime)
			}
		}

		// Determine clip duration and end time
		var endTime time.Duration
		if i < len(timecodes)-1 {
			// End at the next timecode
			endTime = timecodes[i+1]
		} else {
			// For the last clip, extend by default duration or to the end of segments
			if len(clipSegments) > 0 {
				endTime = clipSegments[len(clipSegments)-1].EndTime + 10*time.Second
			} else {
				endTime = startTime + 30*time.Second // Default 30s clip
			}
		}

		// Ensure minimum clip duration
		minDuration := 5 * time.Second
		if endTime-startTime < minDuration {
			endTime = startTime + minDuration
		}

		firstSegmentText := ""
		if len(clipSegments) > 0 {
			firstSegmentText = clipSegments[0].Text
		}

		clip := fcp.Clip{
			StartTime:        startTime,
			EndTime:          endTime,
			Duration:         endTime - startTime,
			Text:             strings.Join(clipText, " "),
			FirstSegmentText: firstSegmentText,
			ClipNum:          i + 1,
		}

		clips = append(clips, clip)
	}

	return clips, nil
}

package vtt

import (
	"cutlass/fcp"
	"sort"
	"strings"
	"time"
)

// postProcessSegments removes any segment that is a full substring of the very next
// (longer) segment – this happens when the sliding-window ended a sentence midway and
// we emitted it, but the following chunk already contains it fully.
func postProcessSegments(segs []Segment) []Segment {
	if len(segs) < 2 {
		return segs
	}
	var out []Segment
	for i := 0; i < len(segs); i++ {
		wordCount := len(strings.Fields(segs[i].Text))
		if wordCount < 5 {
			// Too short – likely artifact unless the next segment is simply a continuation.
			if i < len(segs)-1 {
				continue
			}
		}

		if i < len(segs)-1 {
			cur := strings.ToLower(strings.TrimSpace(segs[i].Text))
			nxt := strings.ToLower(strings.TrimSpace(segs[i+1].Text))
			if len(cur) < 60 && strings.Contains(nxt, cur) {
				// Likely redundant stub – drop.
				continue
			}

			// Fuzzy containment – if at least 80% of current words are present in
			// the next longer segment, treat as duplicate.
			curWords := strings.Fields(cur)
			nxtWords := strings.Fields(nxt)
			if len(curWords) > 0 && len(nxtWords) > len(curWords) {
				wordSet := make(map[string]struct{}, len(nxtWords))
				for _, w := range nxtWords {
					wordSet[w] = struct{}{}
				}
				kept := 0
				for _, w := range curWords {
					if _, ok := wordSet[w]; ok {
						kept++
					}
				}
				if kept*5 >= len(curWords)*4 { // >=80%
					continue
				}
			}
		}
		out = append(out, segs[i])
	}
	return out
}
func SegmentIntoClips(segments []Segment, minDuration, maxDuration time.Duration) []fcp.Clip {
	var clips []fcp.Clip
	clipNum := 1

	// Sort segments by start time
	sort.Slice(segments, func(i, j int) bool {
		return segments[i].StartTime < segments[j].StartTime
	})

	i := 0
	for i < len(segments) {
		clipStart := segments[i].StartTime
		var clipTexts []string
		clipTexts = append(clipTexts, segments[i].Text)

		// Extend clip duration by adding consecutive segments
		j := i + 1
		for j < len(segments) {
			// Check if adding this segment would exceed max duration
			proposedDuration := segments[j].EndTime - clipStart
			if proposedDuration > maxDuration {
				break
			}

			// Add this segment to the clip
			clipTexts = append(clipTexts, segments[j].Text)
			j++

			// Check if we have minimum duration and there's a natural break
			currentDuration := segments[j-1].EndTime - clipStart
			if currentDuration >= minDuration {
				// Look for sentence endings or pauses
				lastText := segments[j-1].Text
				if strings.HasSuffix(lastText, ".") || strings.HasSuffix(lastText, "!") || strings.HasSuffix(lastText, "?") {
					break
				}
			}
		}

		clipEnd := segments[j-1].EndTime
		clipDuration := clipEnd - clipStart

		// Ensure minimum duration
		if clipDuration < minDuration && j < len(segments) {
			clipEnd = clipStart + minDuration
			// Also need to capture any additional text that falls within this extended duration
			for k := j; k < len(segments) && segments[k].StartTime < clipEnd; k++ {
				clipTexts = append(clipTexts, segments[k].Text)
			}
		}

		clips = append(clips, fcp.Clip{
			StartTime:        clipStart,
			EndTime:          clipEnd,
			Duration:         clipEnd - clipStart,
			Text:             strings.Join(clipTexts, " "),
			FirstSegmentText: segments[i].Text, // Just the first segment for textblock preview
			ClipNum:          clipNum,
		})

		clipNum++
		i = j
	}

	return clips
}

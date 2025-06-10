package vtt

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

func ParseTime(timeStr string) (time.Duration, error) {
	// Parse format like "00:00:02.350"
	parts := strings.Split(timeStr, ":")
	if len(parts) != 3 {
		return 0, fmt.Errorf("invalid time format: %s", timeStr)
	}

	hours, _ := strconv.Atoi(parts[0])
	minutes, _ := strconv.Atoi(parts[1])
	secondsParts := strings.Split(parts[2], ".")
	seconds, _ := strconv.Atoi(secondsParts[0])
	milliseconds := 0
	if len(secondsParts) > 1 {
		// Pad or truncate to 3 digits
		msStr := secondsParts[1]
		if len(msStr) > 3 {
			msStr = msStr[:3]
		} else {
			for len(msStr) < 3 {
				msStr += "0"
			}
		}
		milliseconds, _ = strconv.Atoi(msStr)
	}

	return time.Duration(hours)*time.Hour +
		time.Duration(minutes)*time.Minute +
		time.Duration(seconds)*time.Second +
		time.Duration(milliseconds)*time.Millisecond, nil
}

// isSentenceTerminator returns true if the supplied word concludes with a
// sentence-ending punctuation mark.
func isSentenceTerminator(word string) bool {
	trimmed := strings.TrimSpace(word)
	if trimmed == "" {
		return false
	}
	last := trimmed[len(trimmed)-1]
	return last == '.' || last == '!' || last == '?'
}

// cleanRepeatedWords removes immediate duplicate words (case-insensitive) and also trims
// redundant "the the", "it's it's" style glitches that survive YouTube’s sliding window
// captions.
func cleanRepeatedWords(s string) string {
	if s == "" {
		return s
	}
	words := strings.Fields(s)
	if len(words) < 2 {
		return s
	}
	out := make([]string, 0, len(words))
	prev := ""
	for _, w := range words {
		low := strings.ToLower(w)
		if low == prev {
			// skip duplicate
			continue
		}
		out = append(out, w)
		prev = low
	}
	return strings.Join(out, " ")
}

// findOverlapLength finds how many words from the beginning of newWords
// match the end of accumulatedWords
func findOverlapLength(accumulatedWords, newWords []string) int {
	maxOverlap := min(len(accumulatedWords), len(newWords))

	for overlapLen := maxOverlap; overlapLen > 0; overlapLen-- {
		// Check if the last overlapLen words of accumulated match
		// the first overlapLen words of new
		match := true
		for i := 0; i < overlapLen; i++ {
			accWord := strings.ToLower(strings.Trim(accumulatedWords[len(accumulatedWords)-overlapLen+i], ".,!?;:"))
			newWord := strings.ToLower(strings.Trim(newWords[i], ".,!?;:"))
			if accWord != newWord {
				match = false
				break
			}
		}
		if match {
			return overlapLen
		}
	}

	return 0
}

// displayCleanedText displays the cleaned segments with timestamps
func displayCleanedText(segments []Segment) {
	for _, segment := range segments {
		text := strings.TrimSpace(segment.Text)
		if text != "" {
			fmt.Printf("[%v] %s\n", formatDuration(segment.StartTime), text)
		}
	}
	fmt.Println()
}

// formatDuration formats a time.Duration as MM:SS with absolute seconds
func formatDuration(d time.Duration) string {
	minutes := int(d.Minutes())
	seconds := int(d.Seconds()) % 60
	totalSeconds := int(d.Seconds())
	return fmt.Sprintf("%02d:%02d (%ds)", minutes, seconds, totalSeconds)
}

// refineClipBoundariesWithAudio adjusts clip timing to align with natural speech pauses
func refineClipBoundariesWithAudio(clip Segment, silenceGaps []SilenceGap) Segment {
	if len(silenceGaps) == 0 {
		return addNaturalPadding(clip)
	}

	// Look for silence gaps near the start and end of the clip
	searchWindow := 2 * time.Second

	// Refine start time
	for _, gap := range silenceGaps {
		if gap.End >= clip.StartTime-searchWindow && gap.End <= clip.StartTime+searchWindow {
			if gap.Duration >= 300*time.Millisecond {
				// Start the clip just after this silence gap
				clip.StartTime = gap.End + 50*time.Millisecond
				break
			}
		}
	}

	// Refine end time
	for _, gap := range silenceGaps {
		if gap.Start >= clip.EndTime-searchWindow && gap.Start <= clip.EndTime+searchWindow {
			if gap.Duration >= 300*time.Millisecond {
				// End the clip just before this silence gap
				clip.EndTime = gap.Start - 50*time.Millisecond
				break
			}
		}
	}

	// Ensure we don't go negative or create invalid clips
	if clip.StartTime < 0 {
		clip.StartTime = 0
	}
	if clip.EndTime <= clip.StartTime {
		clip.EndTime = clip.StartTime + 2*time.Second
	}

	return clip
}

// containsInterestingWords checks for emotionally engaging or interesting content
func containsInterestingWords(text string) bool {
	lower := strings.ToLower(text)
	interestingWords := []string{
		"nervous", "scared", "funny", "love", "hate", "amazing", "terrible",
		"excited", "surprised", "shocked", "dream", "favorite", "worst",
		"best", "never", "always", "remember", "forget", "secret", "truth",
	}

	for _, word := range interestingWords {
		if strings.Contains(lower, word) {
			return true
		}
	}
	return false
}

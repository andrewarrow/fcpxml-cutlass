package vtt

import (
	"strings"
	"time"
)

// Helper functions for smart clip analysis

func endsWithCompletePunctuation(text string) bool {
	text = strings.TrimSpace(text)
	if len(text) == 0 {
		return false
	}
	last := text[len(text)-1]
	return last == '.' || last == '!' || last == '?' || last == '"'
}

func isTopicChange(current, next string) bool {
	// Simple heuristic: if next starts with question words, it's likely a topic change
	nextLower := strings.ToLower(strings.TrimSpace(next))
	topicStarters := []string{"so,", "do you", "would you", "have you", "what", "where", "when", "why", "how"}

	for _, starter := range topicStarters {
		if strings.HasPrefix(nextLower, starter) {
			return true
		}
	}
	return false
}

func addNaturalPadding(segment Segment) Segment {
	// Add small padding to ensure we don't cut off words
	segment.StartTime = segment.StartTime - 200*time.Millisecond
	segment.EndTime = segment.EndTime + 300*time.Millisecond

	// Don't go negative
	if segment.StartTime < 0 {
		segment.StartTime = 0
	}

	return segment
}


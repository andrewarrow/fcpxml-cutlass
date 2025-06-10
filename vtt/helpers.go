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

func containsStorytellingElements(text string) bool {
	lower := strings.ToLower(text)
	indicators := []string{
		"once", "remember", "story", "told", "happened", "saw", "went",
		"growing up", "one time", "i was", "there was", "back when",
	}

	for _, indicator := range indicators {
		if strings.Contains(lower, indicator) {
			return true
		}
	}
	return false
}

func containsHumorIndicators(text string) bool {
	lower := strings.ToLower(text)
	indicators := []string{
		"funny", "hilarious", "laugh", "haha", "lol", "joke", "kidding",
		"ridiculous", "crazy", "insane", "weird", "awkward", "embarrassing",
	}

	for _, indicator := range indicators {
		if strings.Contains(lower, indicator) {
			return true
		}
	}

	// Check for self-deprecating humor patterns
	if strings.Contains(lower, "i'm") && (strings.Contains(lower, "terrible") ||
		strings.Contains(lower, "awful") || strings.Contains(lower, "bad")) {
		return true
	}

	return false
}

func containsEmotionalContent(text string) bool {
	lower := strings.ToLower(text)
	emotions := []string{
		"love", "hate", "excited", "scared", "nervous", "amazing", "incredible",
		"beautiful", "terrible", "awful", "wonderful", "shocking", "surprised",
		"crying", "tears", "heart", "feel", "emotional", "touched",
	}

	for _, emotion := range emotions {
		if strings.Contains(lower, emotion) {
			return true
		}
	}
	return false
}

func containsDialogueFlow(text string) bool {
	hasQuestion := strings.Contains(text, "?")
	hasResponse := strings.Contains(strings.ToLower(text), "no") ||
		strings.Contains(strings.ToLower(text), "yes") ||
		strings.Contains(strings.ToLower(text), "well") ||
		strings.Contains(strings.ToLower(text), "actually")

	return hasQuestion && hasResponse
}

func isQuotable(text string) bool {
	words := strings.Fields(text)

	// Short, punchy statements
	if len(words) >= 3 && len(words) <= 12 {
		lower := strings.ToLower(text)
		if strings.Contains(lower, "i think") || strings.Contains(lower, "i believe") ||
			strings.Contains(lower, "the truth is") || strings.Contains(lower, "honestly") {
			return true
		}
	}

	// Contains quotation marks (direct quotes)
	if strings.Contains(text, "\"") {
		return true
	}

	return false
}

func containsPersonalRevelation(text string) bool {
	lower := strings.ToLower(text)
	revelations := []string{
		"to be honest", "honestly", "truth is", "confession", "secret",
		"never told", "first time", "admit", "confess", "reveal",
		"i've never", "nobody knows", "between you and me",
	}

	for _, revelation := range revelations {
		if strings.Contains(lower, revelation) {
			return true
		}
	}
	return false
}

func containsReactions(text string) bool {
	lower := strings.ToLower(text)
	reactions := []string{
		"oh my god", "omg", "wow", "whoa", "really?", "seriously?",
		"no way", "are you kidding", "what?", "huh?", "wait",
		"hold on", "pause", "stop", "that's crazy", "unbelievable",
	}

	for _, reaction := range reactions {
		if strings.Contains(lower, reaction) {
			return true
		}
	}
	return false
}

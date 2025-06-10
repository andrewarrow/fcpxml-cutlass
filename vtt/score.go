package vtt

import "strings"

// calculateAdvancedScore uses sophisticated metrics to score clip quality
func calculateAdvancedScore(text string, duration int) float64 {
	if text == "" {
		return 0
	}

	words := strings.Fields(text)
	wordCount := len(words)

	score := 0.0

	// Base score: balanced duration and content density
	wordsPerSecond := float64(wordCount) / float64(duration)
	if wordsPerSecond >= 2.0 && wordsPerSecond <= 4.0 { // Natural speaking pace
		score += 3.0
	} else {
		score += 1.0
	}

	// Content quality bonuses

	// 1. Storytelling elements
	if containsStorytellingElements(text) {
		score += 4.0
	}

	// 2. Humor and entertainment
	if containsHumorIndicators(text) {
		score += 3.5
	}

	// 3. Emotional engagement
	if containsEmotionalContent(text) {
		score += 3.0
	}

	// 4. Question-answer dynamics
	if containsDialogueFlow(text) {
		score += 2.5
	}

	// 5. Quotable moments
	if isQuotable(text) {
		score += 2.0
	}

	// 6. Personal revelations/confessions
	if containsPersonalRevelation(text) {
		score += 2.5
	}

	// 7. Complete thoughts bonus
	if endsWithCompletePunctuation(text) && wordCount >= 8 {
		score += 1.5
	}

	// 8. Reaction moments
	if containsReactions(text) {
		score += 1.0
	}

	// Duration sweet spot (5-15 seconds optimal)
	if duration >= 5 && duration <= 15 {
		score += 2.0
	} else if duration >= 3 && duration <= 20 {
		score += 1.0
	}

	// Penalty for very short or very long clips
	if duration < 3 {
		score *= 0.5
	}
	if duration > 25 {
		score *= 0.7
	}

	return score
}

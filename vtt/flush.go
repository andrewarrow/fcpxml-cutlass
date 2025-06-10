package vtt

import (
	"strings"
	"time"
)

// flushChunk moves the words that are being built up into the results slice,
// respecting global de-duplication.
func flushChunk(result *[]Segment, accumulatedWords *[]string, sentenceCount *int, startTime *time.Duration, endTime time.Duration, seen map[string]struct{}, prevChunkTailPtr *[]string) {
	if len(*accumulatedWords) == 0 {
		*sentenceCount = 0
		return
	}

	text := strings.Join(*accumulatedWords, " ")
	cleaned := cleanRepeatedWords(strings.TrimSpace(text))
	if cleaned == "" {
		*accumulatedWords = nil
		*sentenceCount = 0
		return
	}

	if _, dup := seen[cleaned]; dup {
		// Skip duplicate chunk.
		*accumulatedWords = nil
		*sentenceCount = 0
		*startTime = endTime
		return
	}

	*result = append(*result, Segment{
		StartTime: *startTime,
		EndTime:   endTime,
		Text:      cleaned,
	})

	seen[cleaned] = struct{}{}

	// Update the previous-chunk tail (last 20 words) so we can use it for
	// cross-chunk de-duplication on the next pass.
	words := strings.Fields(cleaned)
	if len(words) > 20 {
		words = words[len(words)-20:]
	}
	*prevChunkTailPtr = words

	// Reset builders.
	*accumulatedWords = nil
	*sentenceCount = 0
	*startTime = endTime
}

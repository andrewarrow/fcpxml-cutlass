package vtt

import "strings"

// removeOverlappingText processes VTT segments to remove overlapping text
// removeOverlappingTextImproved removes duplicate sliding–window captions *and*
// splits the incoming stream into logical chunks that contain at most
// maxSentencesPerSegment sentences (defaults to 1 when set to 0).
//
// The original implementation only looked at the *last* word of every incoming
// VTT fragment to decide whether it should flush the buffer.  Because YouTube
// (and other) captions often repeat an early-sentence word after the timestamp
// break, the sentence-terminating punctuation rarely lands on the last token –
// which resulted in buffers that collected dozens of sentences.  We now inspect
// every word that is appended, flush whenever we hit a sentence terminator and
// – optionally – after `maxSentencesPerSegment` sentences have been collected.
//
// We also globally deduplicate *whole* segments so identical sentences that are
// emitted multiple times (very common with YouTube auto-captions) no longer
// survive the cleaning pass.
func removeOverlappingTextImproved(segments []Segment, maxSentencesPerSegment int) []Segment {
	if len(segments) == 0 {
		return segments
	}

	if maxSentencesPerSegment <= 0 {
		maxSentencesPerSegment = 1
	}

	var result []Segment
	var accumulatedWords []string        // words for the currently building chunk
	var sentenceCount int                // how many sentence terminators we've met in the current chunk
	currentTime := segments[0].StartTime // start time for the current chunk

	// Tail of the *previous* emitted chunk – used so that we can remove any
	// leading overlap from the *next* chunk.  We keep a small window (20 words)
	// which is plenty for typical captions.
	var prevChunkTail []string

	// Keeps track of *entire* cleaned chunks we have already emitted so we can
	// avoid duplicates like "to have who to have watched this match." which are
	// sometimes repeated verbatim three or four times in a row.
	seenChunks := make(map[string]struct{})

	for _, segment := range segments {
		text := strings.TrimSpace(segment.Text)
		if text == "" {
			continue
		}

		words := strings.Fields(text)
		if len(words) == 0 {
			continue
		}

		// If we're at the very beginning of a fresh chunk (accumulatedWords is
		// empty) we additionally check for overlap with the *tail* of the
		// previously emitted chunk so we don't start the new one with text
		// that we literally just output.
		var overlapLen int
		if len(accumulatedWords) == 0 && len(prevChunkTail) > 0 {
			overlapLen = findOverlapLength(prevChunkTail, words)
		} else {
			overlapLen = findOverlapLength(accumulatedWords, words)
		}
		if overlapLen < len(words) {
			words = words[overlapLen:]
		} else {
			// Entire set of words already present – skip.
			continue
		}

		// Check if the majority of the remaining words are already present in
		// our current buffer – this is a strong indicator that we are looking
		// at a pure repetition caused by the sliding-window nature of the
		// captions.  In that case we simply drop the fragment.
		if len(accumulatedWords) > 0 {
			wordSet := make(map[string]struct{}, len(accumulatedWords))
			for _, w := range accumulatedWords {
				wordSet[strings.ToLower(w)] = struct{}{}
			}

			dupCnt := 0
			for _, w := range words {
				if _, ok := wordSet[strings.ToLower(w)]; ok {
					dupCnt++
				}
			}
			if dupCnt*3 >= len(words)*2 { // > ~66% duplicates -> skip fragment
				continue
			}
		}

		for _, w := range words {
			accumulatedWords = append(accumulatedWords, w)

			if isSentenceTerminator(w) {
				sentenceCount++
				if sentenceCount >= maxSentencesPerSegment {
					flushChunk(&result, &accumulatedWords, &sentenceCount, &currentTime, segment.EndTime, seenChunks, &prevChunkTail)
				}
			}
		}
	}

	// Flush whatever is left.
	if len(accumulatedWords) > 0 {
		flushChunk(&result, &accumulatedWords, &sentenceCount, &currentTime, segments[len(segments)-1].EndTime, seenChunks, &prevChunkTail)
	}

	return result
}

package vtt

import (
	"bufio"
	"fmt"
	"os"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
)

type Segment struct {
	StartTime time.Duration
	EndTime   time.Duration
	Text      string
}

type Clip struct {
	StartTime        time.Duration
	EndTime          time.Duration
	Duration         time.Duration
	Text             string
	FirstSegmentText string // Just the first VTT segment for previews
	ClipNum          int
}

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

func ParseFile(vttPath string) ([]Segment, error) {
	file, err := os.Open(vttPath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var segments []Segment
	scanner := bufio.NewScanner(file)

	// Regex to match timestamp lines like "00:00:00.160 --> 00:00:02.350"
	timeRegex := regexp.MustCompile(`(\d{2}:\d{2}:\d{2}\.\d{3})\s+-->\s+(\d{2}:\d{2}:\d{2}\.\d{3})`)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		if timeRegex.MatchString(line) {
			matches := timeRegex.FindStringSubmatch(line)
			if len(matches) >= 3 {
				startTime, err1 := ParseTime(matches[1])
				endTime, err2 := ParseTime(matches[2])

				if err1 == nil && err2 == nil {
					// Read the next line for text content
					var textLines []string
					for scanner.Scan() {
						textLine := strings.TrimSpace(scanner.Text())
						if textLine == "" {
							break
						}
						// Clean up VTT formatting tags
						cleanText := regexp.MustCompile(`<[^>]*>`).ReplaceAllString(textLine, "")
						cleanText = regexp.MustCompile(`<\d{2}:\d{2}:\d{2}\.\d{3}>.*?</c>`).ReplaceAllString(cleanText, "")
						if cleanText != "" {
							textLines = append(textLines, cleanText)
						}
					}

					if len(textLines) > 0 {
						segments = append(segments, Segment{
							StartTime: startTime,
							EndTime:   endTime,
							Text:      strings.Join(textLines, " "),
						})
					}
				}
			}
		}
	}

	return segments, scanner.Err()
}

func SegmentIntoClips(segments []Segment, minDuration, maxDuration time.Duration) []Clip {
	var clips []Clip
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

		clips = append(clips, Clip{
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

// ParseAndDisplayCleanText parses a VTT file and displays cleaned, readable text
func ParseAndDisplayCleanText(vttPath string) error {
	segments, err := ParseFile(vttPath)
	if err != nil {
		return fmt.Errorf("failed to parse VTT file: %v", err)
	}

	if len(segments) == 0 {
		fmt.Printf("No segments found in VTT file\n")
		return nil
	}

	fmt.Printf("=== VTT TEXT DISPLAY: %s ===\n\n", vttPath)
	fmt.Printf("Found %d segments\n\n", len(segments))

    // Break into cleaner chunks – default to a maximum of two sentences so the
    // resulting captions are bite-sized and easier to work with.
    cleanedSegments := removeOverlappingTextImproved(segments, 1)
    cleanedSegments = postProcessSegments(cleanedSegments)
	
	fmt.Printf("=== ORIGINAL VTT (choppy) ===\n")
	for i, segment := range segments {
		if i >= 10 { // Show first 10 segments as sample
			fmt.Printf("... (and %d more segments)\n\n", len(segments)-10)
			break
		}
		fmt.Printf("[%v] %s\n", formatDuration(segment.StartTime), segment.Text)
	}
	
	fmt.Printf("=== CLEANED TEXT (with timestamps) ===\n")
	displayCleanedText(cleanedSegments)
	
	// Generate suggested vtt-clips command
	generateSuggestedClipsCommand(vttPath, cleanedSegments)
	
	return nil
}

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
    var accumulatedWords []string       // words for the currently building chunk
    var sentenceCount int               // how many sentence terminators we've met in the current chunk
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

// generateSuggestedClipsCommand analyzes segments and suggests a vtt-clips command
func generateSuggestedClipsCommand(vttPath string, segments []Segment) {
	if len(segments) == 0 {
		return
	}
	
	// Score segments based on length and content quality
	type ScoredSegment struct {
		Segment Segment
		Score   float64
		Duration int
	}
	
	var scored []ScoredSegment
	for _, seg := range segments {
		duration := int(seg.EndTime.Seconds()) - int(seg.StartTime.Seconds())
		if duration < 2 { // Skip very short segments
			continue
		}
		
		text := strings.TrimSpace(seg.Text)
		words := strings.Fields(text)
		
		// Base score on duration and word count
		score := float64(duration) * 0.5 + float64(len(words)) * 0.3
		
		// Bonus for complete sentences
		if strings.ContainsAny(text, ".!?") {
			score += 2.0
		}
		
		// Bonus for interesting content (questions, emotional words)
		if strings.Contains(strings.ToLower(text), "?") {
			score += 1.5
		}
		if containsInterestingWords(text) {
			score += 1.0
		}
		
		scored = append(scored, ScoredSegment{
			Segment: seg,
			Score: score,
			Duration: duration,
		})
	}
	
	// Sort by score descending
	sort.Slice(scored, func(i, j int) bool {
		return scored[i].Score > scored[j].Score
	})
	
	// Select segments for approximately 2 minutes (120 seconds)
	var selected []ScoredSegment
	totalDuration := 0
	targetDuration := 120
	
	for _, seg := range scored {
		if totalDuration + seg.Duration <= targetDuration {
			selected = append(selected, seg)
			totalDuration += seg.Duration
		}
		if totalDuration >= int(float64(targetDuration) * 0.8) { // Stop when we're close to target
			break
		}
	}
	
	// Sort selected segments by start time
	sort.Slice(selected, func(i, j int) bool {
		return selected[i].Segment.StartTime < selected[j].Segment.StartTime
	})
	
	if len(selected) == 0 {
		return
	}
	
	// Build the command
	fmt.Printf("=== SUGGESTED CLIPS COMMAND ===\n")
	fmt.Printf("For a ~%d second video, try:\n\n", totalDuration)
	
	clipPairs := make([]string, len(selected))
	for i, seg := range selected {
		startMin := int(seg.Segment.StartTime.Minutes())
		startSec := int(seg.Segment.StartTime.Seconds()) % 60
		clipPairs[i] = fmt.Sprintf("%02d:%02d_%d", startMin, startSec, seg.Duration)
	}
	
	fmt.Printf("./cutlass vtt-clips %s %s\n\n", vttPath, strings.Join(clipPairs, ","))
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
package vtt

import (
	"bufio"
	"cutlass/fcp"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
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

type SilenceGap struct {
	Start    time.Duration
	End      time.Duration
	Duration time.Duration
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

	// Score clips based on multiple quality factors
	type ScoredClip struct {
		StartTime time.Duration
		EndTime   time.Duration
		Text      string
		Score     float64
		Duration  int
	}

	var scored []ScoredClip
	// Select clips for approximately 2 minutes (120 seconds)
	// Distribute clips across the entire video timeline
	var selected []ScoredClip
	totalDuration := 0
	targetDuration := 120

	if len(scored) == 0 {
		return
	}

	// Find the total video duration to create time buckets
	lastSegment := segments[len(segments)-1]
	videoDuration := lastSegment.EndTime.Seconds()

	// Create time buckets to ensure distribution across the video
	numBuckets := 6 // Divide video into 6 sections for good distribution
	bucketSize := videoDuration / float64(numBuckets)

	// Group clips by time buckets
	buckets := make([][]ScoredClip, numBuckets)
	for _, clip := range scored {
		bucketIndex := int(clip.StartTime.Seconds() / bucketSize)
		if bucketIndex >= numBuckets {
			bucketIndex = numBuckets - 1
		}
		buckets[bucketIndex] = append(buckets[bucketIndex], clip)
	}

	// Sort each bucket by score
	for i := range buckets {
		sort.Slice(buckets[i], func(j, k int) bool {
			return buckets[i][j].Score > buckets[i][k].Score
		})
	}

	// Select best clips from each bucket in round-robin fashion
	for round := 0; round < 5 && totalDuration < targetDuration; round++ {
		for bucketIdx := 0; bucketIdx < numBuckets && totalDuration < targetDuration; bucketIdx++ {
			bucket := buckets[bucketIdx]
			if round < len(bucket) {
				clip := bucket[round]
				if totalDuration+clip.Duration <= targetDuration {
					selected = append(selected, clip)
					totalDuration += clip.Duration
				}
			}
		}
	}

	// Sort selected clips by start time
	sort.Slice(selected, func(i, j int) bool {
		return selected[i].StartTime < selected[j].StartTime
	})

	if len(selected) == 0 {
		return
	}

	// Build the command
	fmt.Printf("=== SUGGESTED CLIPS COMMAND ===\n")
	fmt.Printf("For a ~%d second video, try:\n\n", totalDuration)

	clipPairs := make([]string, len(selected))
	for i, clip := range selected {
		startMin := int(clip.StartTime.Minutes())
		startSec := int(clip.StartTime.Seconds()) % 60
		clipPairs[i] = fmt.Sprintf("%02d:%02d_%d", startMin, startSec, clip.Duration)
	}

	fmt.Printf("./cutlass vtt-clips %s %s\n\n", vttPath, strings.Join(clipPairs, ","))
}

// createSmartClips merges segments into complete thoughts with natural boundaries
func createSmartClips(segments []Segment) []Segment {
	if len(segments) == 0 {
		return segments
	}

	var smartClips []Segment
	i := 0

	for i < len(segments) {
		currentClip := segments[i]
		currentText := strings.TrimSpace(currentClip.Text)

		// Look ahead to merge incomplete thoughts
		j := i + 1
		for j < len(segments) {
			nextSegment := segments[j]
			nextText := strings.TrimSpace(nextSegment.Text)

			// Check if we should merge with next segment
			shouldMerge := false

			// Merge if current ends without punctuation and next is continuation
			if !endsWithCompletePunctuation(currentText) {
				shouldMerge = true
			}

			// Merge if current text is very short (likely incomplete)
			if len(strings.Fields(currentText)) < 4 {
				shouldMerge = true
			}

			// Merge if next starts with lowercase (continuation)
			if len(nextText) > 0 && nextText[0] >= 'a' && nextText[0] <= 'z' {
				shouldMerge = true
			}

			// Don't merge if the combined clip would be too long
			combinedDuration := nextSegment.EndTime - currentClip.StartTime
			if combinedDuration > 25*time.Second {
				break
			}

			// Don't merge if we've hit a clear topic change
			if isTopicChange(currentText, nextText) {
				break
			}

			if shouldMerge {
				// Merge the segments
				currentClip.EndTime = nextSegment.EndTime
				currentClip.Text = currentText + " " + nextText
				currentText = currentClip.Text
				j++
			} else {
				break
			}
		}

		// Add timing padding for natural speech boundaries
		currentClip = addNaturalPadding(currentClip)

		smartClips = append(smartClips, currentClip)
		i = j
	}

	return smartClips
}

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

// findAudioFile looks for corresponding audio/video file for the VTT
func findAudioFile(vttPath string) string {
	// Remove .vtt extension and try common video/audio extensions
	baseName := strings.TrimSuffix(vttPath, ".vtt")
	baseName = strings.TrimSuffix(baseName, ".en") // Remove language suffix if present

	extensions := []string{".mov", ".mp4", ".m4a", ".wav", ".mp3", ".mkv", ".avi"}

	for _, ext := range extensions {
		candidateFile := baseName + ext
		if _, err := os.Stat(candidateFile); err == nil {
			return candidateFile
		}
	}

	return ""
}

// detectSilenceGaps uses FFmpeg to analyze audio and find natural speech pauses
func detectSilenceGaps(audioFile string) []SilenceGap {
	// Use FFmpeg's silencedetect filter to find gaps
	cmd := exec.Command("ffmpeg", "-i", audioFile, "-af",
		"silencedetect=noise=-30dB:duration=0.3", "-f", "null", "-")

	output, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Printf("Warning: Could not analyze audio waveform: %v\n", err)
		return nil
	}

	return parseSilenceOutput(string(output))
}

// parseSilenceOutput extracts silence timestamps from FFmpeg output
func parseSilenceOutput(output string) []SilenceGap {
	var gaps []SilenceGap
	lines := strings.Split(output, "\n")

	silenceStartRegex := regexp.MustCompile(`silence_start: ([0-9.]+)`)
	silenceEndRegex := regexp.MustCompile(`silence_end: ([0-9.]+)`)

	var currentStart *time.Duration

	for _, line := range lines {
		if matches := silenceStartRegex.FindStringSubmatch(line); len(matches) > 1 {
			if seconds, err := strconv.ParseFloat(matches[1], 64); err == nil {
				start := time.Duration(seconds * float64(time.Second))
				currentStart = &start
			}
		} else if matches := silenceEndRegex.FindStringSubmatch(line); len(matches) > 1 && currentStart != nil {
			if seconds, err := strconv.ParseFloat(matches[1], 64); err == nil {
				end := time.Duration(seconds * float64(time.Second))
				duration := end - *currentStart

				// Only include meaningful pauses (300ms or longer)
				if duration >= 300*time.Millisecond {
					gaps = append(gaps, SilenceGap{
						Start:    *currentStart,
						End:      end,
						Duration: duration,
					})
				}
				currentStart = nil
			}
		}
	}

	return gaps
}

// createSmartClipsWithAudio creates clips using both text analysis and audio boundaries
func createSmartClipsWithAudio(segments []Segment, silenceGaps []SilenceGap) []Segment {
	if len(silenceGaps) == 0 {
		// Fallback to text-only analysis
		return createSmartClips(segments)
	}

	var smartClips []Segment
	i := 0

	for i < len(segments) {
		currentClip := segments[i]
		currentText := strings.TrimSpace(currentClip.Text)

		// Look ahead to merge incomplete thoughts
		j := i + 1
		for j < len(segments) {
			nextSegment := segments[j]
			nextText := strings.TrimSpace(nextSegment.Text)

			// Check if we should merge with next segment
			shouldMerge := false

			// Standard text-based merging logic
			if !endsWithCompletePunctuation(currentText) {
				shouldMerge = true
			}
			if len(strings.Fields(currentText)) < 4 {
				shouldMerge = true
			}
			if len(nextText) > 0 && nextText[0] >= 'a' && nextText[0] <= 'z' {
				shouldMerge = true
			}

			// Audio-based boundary detection
			if shouldMerge {
				// Check if there's a natural pause between segments
				gapBetween := nextSegment.StartTime - currentClip.EndTime
				if gapBetween > 800*time.Millisecond {
					// Long gap suggests natural boundary
					shouldMerge = false
				} else {
					// Look for silence gaps near the boundary
					boundaryTime := nextSegment.StartTime
					for _, gap := range silenceGaps {
						// If there's a silence gap within 1 second of the boundary
						if gap.Start <= boundaryTime+time.Second && gap.End >= boundaryTime-time.Second {
							if gap.Duration >= 400*time.Millisecond {
								shouldMerge = false
								break
							}
						}
					}
				}
			}

			// Don't merge if combined clip would be too long
			combinedDuration := nextSegment.EndTime - currentClip.StartTime
			if combinedDuration > 20*time.Second {
				break
			}

			// Don't merge if we've hit a topic change
			if isTopicChange(currentText, nextText) {
				break
			}

			if shouldMerge {
				currentClip.EndTime = nextSegment.EndTime
				currentClip.Text = currentText + " " + nextText
				currentText = currentClip.Text
				j++
			} else {
				break
			}
		}

		// Refine clip boundaries using audio analysis
		currentClip = refineClipBoundariesWithAudio(currentClip, silenceGaps)

		smartClips = append(smartClips, currentClip)
		i = j
	}

	return smartClips
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

// TimecodeWithDuration represents a timecode with its duration
type TimecodeWithDuration struct {
	Start    time.Duration
	Duration time.Duration
}

// ParseTimecode parses MM:SS format timecode to time.Duration
func ParseTimecode(timecode string) (time.Duration, error) {
	parts := strings.Split(timecode, ":")
	if len(parts) != 2 {
		return 0, fmt.Errorf("timecode must be in MM:SS format")
	}

	minutes, err := strconv.Atoi(parts[0])
	if err != nil {
		return 0, fmt.Errorf("invalid minutes: %v", err)
	}

	seconds, err := strconv.Atoi(parts[1])
	if err != nil {
		return 0, fmt.Errorf("invalid seconds: %v", err)
	}

	if minutes < 0 || seconds < 0 || seconds >= 60 {
		return 0, fmt.Errorf("invalid time values")
	}

	return time.Duration(minutes)*time.Minute + time.Duration(seconds)*time.Second, nil
}

// ParseTimecodeWithDuration parses MM:SS_duration format to start time and duration
func ParseTimecodeWithDuration(timecode string) (TimecodeWithDuration, error) {
	var result TimecodeWithDuration

	parts := strings.Split(timecode, "_")
	if len(parts) != 2 {
		return result, fmt.Errorf("timecode must be in MM:SS_duration format")
	}

	// Parse start time
	startTime, err := ParseTimecode(parts[0])
	if err != nil {
		return result, fmt.Errorf("invalid start time: %v", err)
	}

	// Parse duration in seconds
	durationSeconds, err := strconv.Atoi(parts[1])
	if err != nil {
		return result, fmt.Errorf("invalid duration: %v", err)
	}

	if durationSeconds <= 0 {
		return result, fmt.Errorf("duration must be positive")
	}

	result.Start = startTime
	result.Duration = time.Duration(durationSeconds) * time.Second
	return result, nil
}

// GenerateVTTClips generates FCPXML from VTT file and timecodes
func GenerateVTTClips(vttFile, timecodesStr, outputFile string) error {
	// Parse VTT filename to extract base ID (e.g., "IBnNedMh4Pg" from "IBnNedMh4Pg.en.vtt")
	baseName := filepath.Base(vttFile)

	// Remove .en.vtt suffix
	var videoID string
	if strings.HasSuffix(baseName, ".en.vtt") {
		videoID = strings.TrimSuffix(baseName, ".en.vtt")
	} else if strings.HasSuffix(baseName, ".vtt") {
		videoID = strings.TrimSuffix(baseName, ".vtt")
	} else {
		return fmt.Errorf("VTT file must end with .vtt or .en.vtt")
	}

	// Find corresponding MOV file in same directory
	vttDir := filepath.Dir(vttFile)
	videoFile := filepath.Join(vttDir, videoID+".mov")

	if _, err := os.Stat(videoFile); os.IsNotExist(err) {
		return fmt.Errorf("corresponding video file not found: %s", videoFile)
	}

	// Parse timecodes (format: "01:21_6,02:20_3,03:34_9,05:07_18")
	timecodeStrs := strings.Split(timecodesStr, ",")
	if len(timecodeStrs) == 0 {
		return fmt.Errorf("no timecodes provided")
	}

	var timecodes []TimecodeWithDuration
	for _, tc := range timecodeStrs {
		tc = strings.TrimSpace(tc)
		timecodeData, err := ParseTimecodeWithDuration(tc)
		if err != nil {
			return fmt.Errorf("invalid timecode '%s': %v", tc, err)
		}
		timecodes = append(timecodes, timecodeData)
	}

	// Set default output file if not provided
	if outputFile == "" {
		outputFile = filepath.Join(vttDir, videoID+"_clips.fcpxml")
	}
	if !strings.HasSuffix(strings.ToLower(outputFile), ".fcpxml") {
		outputFile += ".fcpxml"
	}

	// Parse VTT file to get all segments
	segments, err := ParseFile(vttFile)
	if err != nil {
		return fmt.Errorf("failed to parse VTT file: %v", err)
	}

	// Create clips from timecodes with durations
	clips, err := CreateClipsFromTimecodesWithDuration(segments, timecodes)
	if err != nil {
		return fmt.Errorf("failed to create clips: %v", err)
	}

	// Need to import fcp package - for now return with message
	fmt.Printf("Generating FCPXML with %d clips from %s\n", len(clips), videoFile)
	err = fcp.GenerateClipFCPXML(clips, videoFile, outputFile)
	if err != nil {
		return fmt.Errorf("failed to generate FCPXML: %v", err)
	}

	fmt.Printf("Would generate %s with %d clips\n", outputFile, len(clips))
	return nil
}

// CreateClipsFromTimecodes creates clips starting at specified timecodes
func CreateClipsFromTimecodes(segments []Segment, timecodes []time.Duration) ([]Clip, error) {
	var clips []Clip

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

		clip := Clip{
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

// CreateClipsFromTimecodesWithDuration creates clips with specified start times and durations
func CreateClipsFromTimecodesWithDuration(segments []Segment, timecodes []TimecodeWithDuration) ([]Clip, error) {
	var clips []Clip

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

		clip := Clip{
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

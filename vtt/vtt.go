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

	cleanedSegments := removeOverlappingText(segments)
	
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
	
	return nil
}

// removeOverlappingText processes VTT segments to remove overlapping text
func removeOverlappingText(segments []Segment) []Segment {
	if len(segments) == 0 {
		return segments
	}
	
	var result []Segment
	var accumulatedWords []string
	currentTime := segments[0].StartTime
	
	for i, segment := range segments {
		text := strings.TrimSpace(segment.Text)
		if text == "" {
			continue
		}
		
		words := strings.Fields(text)
		if len(words) == 0 {
			continue
		}
		
		if i == 0 {
			// First segment - add all words
			accumulatedWords = append(accumulatedWords, words...)
		} else {
			// Find overlap with accumulated text
			overlapLen := findOverlapLength(accumulatedWords, words)
			
			// Only add the non-overlapping part
			if overlapLen < len(words) {
				newWords := words[overlapLen:]
				accumulatedWords = append(accumulatedWords, newWords...)
			}
		}
		
		// Check if we should create a sentence break
		lastWord := words[len(words)-1]
		if strings.HasSuffix(lastWord, ".") || strings.HasSuffix(lastWord, "!") || strings.HasSuffix(lastWord, "?") {
			// Create a segment for this sentence
			if len(accumulatedWords) > 0 {
				result = append(result, Segment{
					StartTime: currentTime,
					EndTime:   segment.EndTime,
					Text:      strings.Join(accumulatedWords, " "),
				})
				accumulatedWords = nil
				currentTime = segment.EndTime
			}
		}
	}
	
	// Add any remaining accumulated text as final segment
	if len(accumulatedWords) > 0 {
		result = append(result, Segment{
			StartTime: currentTime,
			EndTime:   segments[len(segments)-1].EndTime,
			Text:      strings.Join(accumulatedWords, " "),
		})
	}
	
	return result
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

// formatDuration formats a time.Duration as MM:SS
func formatDuration(d time.Duration) string {
	minutes := int(d.Minutes())
	seconds := int(d.Seconds()) % 60
	return fmt.Sprintf("%02d:%02d", minutes, seconds)
}
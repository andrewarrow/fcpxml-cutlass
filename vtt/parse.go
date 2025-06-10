package vtt

import (
	"bufio"
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"
)

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

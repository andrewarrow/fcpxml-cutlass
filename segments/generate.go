package segments

import (
	"cutlass/fcp"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// GenerateSegments generates FCPXML from video ID in ./data/ and timecodes
func GenerateSegments(videoID, timecodesStr, outputFile string) error {
	// Look for video file in ./data/ directory
	videoFile := filepath.Join("./data", videoID+".mov")
	
	if _, err := os.Stat(videoFile); os.IsNotExist(err) {
		return fmt.Errorf("video file not found: %s", videoFile)
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
		outputFile = filepath.Join("./data", videoID+"_segments.fcpxml")
	}
	if !strings.HasSuffix(strings.ToLower(outputFile), ".fcpxml") {
		outputFile += ".fcpxml"
	}

	// Create clips from timecodes (no VTT text needed)
	clips := CreateClipsFromTimecodes(timecodes)

	// Generate FCPXML with clips
	fmt.Printf("Generating FCPXML with %d clips from %s\n", len(clips), videoFile)
	err := fcp.GenerateClipFCPXML(clips, videoFile, outputFile)
	if err != nil {
		return fmt.Errorf("failed to generate FCPXML: %v", err)
	}

	fmt.Printf("Successfully generated %s with %d clips\n", outputFile, len(clips))
	return nil
}

// CreateClipsFromTimecodes creates FCP clips from timecode data without VTT text
func CreateClipsFromTimecodes(timecodes []TimecodeWithDuration) []fcp.Clip {
	var clips []fcp.Clip
	
	for i, tc := range timecodes {
		clip := fcp.Clip{
			StartTime:        tc.StartTime,
			EndTime:          tc.StartTime + tc.Duration,
			Duration:         tc.Duration,
			Text:             fmt.Sprintf("Segment %d", i+1), // Generic text since no VTT
			FirstSegmentText: fmt.Sprintf("Segment %d", i+1),
			ClipNum:          i + 1,
		}
		clips = append(clips, clip)
	}
	
	return clips
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

	result.StartTime = startTime
	result.Duration = time.Duration(durationSeconds) * time.Second
	return result, nil
}
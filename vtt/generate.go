package vtt

import (
	"cutlass/fcp"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

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

	// Generate FCPXML with clips
	fmt.Printf("Generating FCPXML with %d clips from %s\n", len(clips), videoFile)
	err = fcp.GenerateClipFCPXML(clips, videoFile, outputFile)
	if err != nil {
		return fmt.Errorf("failed to generate FCPXML: %v", err)
	}

	fmt.Printf("Successfully generated %s with %d clips\n", outputFile, len(clips))
	return nil
}

// GenerateSegments generates FCPXML from video ID in ./data/ and timecodes
func GenerateSegments(videoID, timecodesStr, outputFile string) error {
	// Look for video file in ./data/ directory
	videoFile := filepath.Join("./data", videoID+".mov")
	
	if _, err := os.Stat(videoFile); os.IsNotExist(err) {
		return fmt.Errorf("video file not found: %s", videoFile)
	}

	// Look for corresponding VTT file in ./data/ directory
	vttFile := filepath.Join("./data", videoID+".en.vtt")
	if _, err := os.Stat(vttFile); os.IsNotExist(err) {
		// Try without .en suffix
		vttFile = filepath.Join("./data", videoID+".vtt")
		if _, err := os.Stat(vttFile); os.IsNotExist(err) {
			return fmt.Errorf("VTT file not found: %s.en.vtt or %s.vtt", 
				filepath.Join("./data", videoID), filepath.Join("./data", videoID))
		}
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

	// Generate FCPXML with clips
	fmt.Printf("Generating FCPXML with %d clips from %s\n", len(clips), videoFile)
	err = fcp.GenerateClipFCPXML(clips, videoFile, outputFile)
	if err != nil {
		return fmt.Errorf("failed to generate FCPXML: %v", err)
	}

	fmt.Printf("Successfully generated %s with %d clips\n", outputFile, len(clips))
	return nil
}

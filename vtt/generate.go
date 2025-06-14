package vtt

import (
	"cutlass/fcp"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// generateSuggestedClipsCommand is a placeholder for future clip analysis
func generateSuggestedClipsCommand(vttPath string, segments []Segment) {
	// TODO: Implement intelligent clip suggestion based on content analysis
	// For now, this function is disabled
	if len(segments) == 0 {
		return
	}
	
	fmt.Printf("=== CLIP SUGGESTION ===\n")
	fmt.Printf("VTT file: %s with %d segments\n", vttPath, len(segments))
	fmt.Printf("Manual clip selection is currently recommended.\n\n")
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

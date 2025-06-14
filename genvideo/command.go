package genvideo

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"cutlass/build2/api"
	"cutlass/build2/utils"
)

func HandleGenVideoCommand(args []string) {
	if len(args) < 1 {
		fmt.Println("Error: Please provide a .genvideo file path")
		return
	}

	genvideoFile := args[0]
	if err := processGenVideoFile(genvideoFile); err != nil {
		fmt.Printf("Error processing .genvideo file: %v\n", err)
	}
}

// GenVideoData represents the parsed .genvideo file content
type GenVideoData struct {
	AudioFile string
	Segments  []VideoSegment
}

// VideoSegment represents a segment with frames and text overlays
type VideoSegment struct {
	Frames []string
	Texts  []string
}

func processGenVideoFile(genvideoFile string) error {
	// Check if .genvideo file exists
	if _, err := os.Stat(genvideoFile); os.IsNotExist(err) {
		return fmt.Errorf(".genvideo file does not exist: %s", genvideoFile)
	}

	// Parse the .genvideo file
	genData, err := parseGenVideoFile(genvideoFile)
	if err != nil {
		return fmt.Errorf("failed to parse .genvideo file: %v", err)
	}

	// Generate output file name
	baseName := strings.TrimSuffix(filepath.Base(genvideoFile), ".genvideo")
	outputFile := filepath.Join(filepath.Dir(genvideoFile), baseName+".fcpxml")

	// Get audio duration
	audioDuration, err := utils.GetAudioDuration(genData.AudioFile)
	if err != nil {
		return fmt.Errorf("failed to get audio duration: %v", err)
	}

	fmt.Printf("Audio file: %s (duration: %s)\n", genData.AudioFile, audioDuration)
	fmt.Printf("Found %d video segments\n", len(genData.Segments))

	// Generate FCPXML using build2 API
	err = generateFCPXMLFromGenData(outputFile, genData, audioDuration)
	if err != nil {
		return fmt.Errorf("failed to generate FCPXML: %v", err)
	}

	fmt.Printf("Generated FCPXML: %s\n", outputFile)
	return nil
}

// parseGenVideoFile parses a .genvideo file and returns structured data
func parseGenVideoFile(filename string) (*GenVideoData, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	genData := &GenVideoData{
		Segments: make([]VideoSegment, 0),
	}

	lineNum := 0
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		lineNum++

		// Skip empty lines and comments
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// First line should be the audio file
		if genData.AudioFile == "" {
			if !strings.HasSuffix(strings.ToLower(line), ".wav") {
				return nil, fmt.Errorf("line %d: first line must be a .wav file, got: %s", lineNum, line)
			}
			genData.AudioFile = line
			continue
		}

		// Parse segment line: frames and text groups separated by commas
		parts := strings.Split(line, ",")
		segment := VideoSegment{
			Frames: make([]string, 0),
			Texts:  make([]string, 0),
		}

		for _, part := range parts {
			part = strings.TrimSpace(part)
			if part == "" {
				continue
			}

			// Check if it's a quoted text string
			if strings.HasPrefix(part, `"`) && strings.HasSuffix(part, `"`) {
				// Remove quotes and add to texts
				text := part[1 : len(part)-1]
				segment.Texts = append(segment.Texts, text)
			} else if strings.HasSuffix(strings.ToLower(part), ".jpg") || strings.HasSuffix(strings.ToLower(part), ".jpeg") || strings.HasSuffix(strings.ToLower(part), ".png") {
				// It's an image file
				segment.Frames = append(segment.Frames, part)
			} else {
				return nil, fmt.Errorf("line %d: unrecognized item '%s' - must be image file or quoted text", lineNum, part)
			}
		}

		if len(segment.Frames) > 0 || len(segment.Texts) > 0 {
			genData.Segments = append(genData.Segments, segment)
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	if genData.AudioFile == "" {
		return nil, fmt.Errorf("no audio file specified")
	}

	if len(genData.Segments) == 0 {
		return nil, fmt.Errorf("no video segments found")
	}

	return genData, nil
}

// validateGenVideoData validates that all referenced files exist
func validateGenVideoData(genData *GenVideoData) error {
	// Check audio file exists
	if _, err := os.Stat(genData.AudioFile); os.IsNotExist(err) {
		return fmt.Errorf("audio file does not exist: %s", genData.AudioFile)
	}

	// Check all image files exist
	for i, segment := range genData.Segments {
		for j, frame := range segment.Frames {
			if _, err := os.Stat(frame); os.IsNotExist(err) {
				return fmt.Errorf("segment %d, frame %d: image file does not exist: %s", i+1, j+1, frame)
			}
		}
	}

	return nil
}

// TextOverlay represents a text overlay with positioning
type TextOverlay struct {
	Text        string
	Lane        int
	Offset      string
	Duration    string
	YPosition   float64
}

// generateTextOverlays creates the staggered 3-lane text overlay system
func generateTextOverlays(texts []string, segmentOffset, segmentDuration string) []TextOverlay {
	overlays := make([]TextOverlay, 0)

	// Create 3 staggered text overlays per text group, cycling through lanes 3, 2, 1
	for i, text := range texts {
		lane := 3 - (i % 3) // Cycles: 3, 2, 1, 3, 2, 1, ...
		
		// Vertical positioning based on pattern from antiedit.fcpxml
		yPos := -19.7744
		if i%2 == 1 {
			yPos = -20.3409
		}

		overlay := TextOverlay{
			Text:      text,
			Lane:      lane,
			Offset:    segmentOffset,
			Duration:  segmentDuration,
			YPosition: yPos,
		}

		overlays = append(overlays, overlay)
	}

	return overlays
}

// calculateSegmentTimings distributes segments across the total audio duration
func calculateSegmentTimings(totalAudioDuration string, numSegments int) ([]string, []string, error) {
	// Parse total duration
	totalFrames, err := parseDurationToFrames(totalAudioDuration)
	if err != nil {
		return nil, nil, err
	}

	// Calculate frames per segment
	framesPerSegment := totalFrames / numSegments
	remainingFrames := totalFrames % numSegments

	offsets := make([]string, numSegments)
	durations := make([]string, numSegments)

	currentOffset := 0
	for i := 0; i < numSegments; i++ {
		// First 'remainingFrames' segments get one extra frame
		segmentFrames := framesPerSegment
		if i < remainingFrames {
			segmentFrames++
		}

		offsets[i] = fmt.Sprintf("%d/24000s", currentOffset)
		durations[i] = fmt.Sprintf("%d/24000s", segmentFrames)

		currentOffset += segmentFrames
	}

	return offsets, durations, nil
}

// parseDurationToFrames converts FCP duration to frame count
func parseDurationToFrames(duration string) (int, error) {
	if duration == "0s" {
		return 0, nil
	}

	if strings.HasSuffix(duration, "/24000s") {
		framesStr := strings.TrimSuffix(duration, "/24000s")
		return strconv.Atoi(framesStr)
	}

	return 0, fmt.Errorf("invalid duration format: %s", duration)
}

// generateUniqueTextStyleID creates unique text style IDs to avoid collisions
func generateUniqueTextStyleID(text string, segmentIndex, textIndex int) string {
	// Create a hash-based ID that's unique but deterministic
	hash := 0
	for _, c := range text {
		hash = hash*31 + int(c)
	}
	hash += segmentIndex*1000 + textIndex*100
	if hash < 0 {
		hash = -hash
	}

	// Generate 8-character alphanumeric ID
	id := "ts"
	for i := 0; i < 6; i++ {
		if i%2 == 0 {
			id += string(rune('A' + (hash>>(i*4))%26))
		} else {
			id += string(rune('0' + (hash>>(i*4))%10))
		}
	}
	return id
}

func generateFCPXMLFromGenData(outputFile string, genData *GenVideoData, audioDuration string) error {
	// Validate all files exist
	if err := validateGenVideoData(genData); err != nil {
		return err
	}

	// Create new project builder
	pb, err := api.NewProjectBuilder(outputFile)
	if err != nil {
		return err
	}

	// Calculate segment timings
	offsets, durations, err := calculateSegmentTimings(audioDuration, len(genData.Segments))
	if err != nil {
		return fmt.Errorf("failed to calculate segment timings: %v", err)
	}

	// Add the main audio track first (single audio file for entire duration)
	err = pb.AddAudioOnlySafe(genData.AudioFile, "0s")
	if err != nil {
		return fmt.Errorf("failed to add main audio track: %v", err)
	}

	// Process each segment - add frames sequentially starting after audio duration
	allTextOverlays := make([]TextOverlay, 0)
	
	// Start video clips after the audio track ends
	audioFrames, err := parseDurationToFrames(audioDuration)
	if err != nil {
		return fmt.Errorf("failed to parse audio duration: %v", err)
	}
	currentVideoOffset := fmt.Sprintf("%d/24000s", audioFrames)
	
	for segmentIndex, segment := range genData.Segments {
		segmentOffset := offsets[segmentIndex]
		segmentDuration := durations[segmentIndex]

		// Add all frames in this segment as video clips with proper positioning
		for _, frame := range segment.Frames {
			err = pb.AddVideoOnlySafe(frame, currentVideoOffset, segmentDuration)
			if err != nil {
				return fmt.Errorf("failed to add frame %s: %v", frame, err)
			}
			
			// Advance offset for next video clip
			segmentFrames, _ := parseDurationToFrames(segmentDuration)
			currentOffsetFrames, _ := parseDurationToFrames(currentVideoOffset)
			currentVideoOffset = fmt.Sprintf("%d/24000s", currentOffsetFrames + segmentFrames)
		}

		// Generate text overlays for this segment
		textOverlays := generateTextOverlays(segment.Texts, segmentOffset, segmentDuration)
		allTextOverlays = append(allTextOverlays, textOverlays...)
	}

	// TODO: Add text overlays - this will require extending the build2 API
	// For now, we'll add a note about text overlays
	if len(allTextOverlays) > 0 {
		fmt.Printf("Note: Generated %d text overlays (not yet implemented in build2 API)\n", len(allTextOverlays))
		for i, overlay := range allTextOverlays {
			fmt.Printf("  Text %d: '%s' on lane %d at %s\n", i+1, overlay.Text, overlay.Lane, overlay.Offset)
		}
	}

	// Save the project
	return pb.Save()
}
package genvideo

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"cutlass/build2/api"
	"cutlass/build2/utils"
)

func HandleGenVideoCommand(args []string) {
	if len(args) < 1 {
		fmt.Println("Error: Please provide a video ID")
		return
	}

	videoID := args[0]
	if err := processVideoID(videoID); err != nil {
		fmt.Printf("Error processing video ID: %v\n", err)
	}
}

func processVideoID(videoID string) error {
	// Define expected directories
	audioDir := fmt.Sprintf("./data/%s_audio", videoID)
	imageDir := fmt.Sprintf("./data/%s", videoID)
	outputFile := fmt.Sprintf("./data/%s.fcpxml", videoID)

	// Check if directories exist
	if _, err := os.Stat(audioDir); os.IsNotExist(err) {
		return fmt.Errorf("audio directory does not exist: %s", audioDir)
	}
	if _, err := os.Stat(imageDir); os.IsNotExist(err) {
		return fmt.Errorf("image directory does not exist: %s", imageDir)
	}

	// Get all WAV files and calculate total duration with caching
	wavFiles, audioDurations, audioDurationsFCP, totalDuration, err := getAudioFilesWithDurations(audioDir)
	if err != nil {
		return fmt.Errorf("failed to get audio files: %v", err)
	}

	if len(wavFiles) == 0 {
		return fmt.Errorf("no WAV files found in %s", audioDir)
	}

	// Get all JPG files
	jpgFiles, err := getImageFiles(imageDir)
	if err != nil {
		return fmt.Errorf("failed to get image files: %v", err)
	}

	if len(jpgFiles) == 0 {
		return fmt.Errorf("no JPG files found in %s", imageDir)
	}

	fmt.Printf("Found %d audio files (total duration: %.2fs)\n", len(wavFiles), totalDuration)
	fmt.Printf("Found %d image files\n", len(jpgFiles))

	// Generate FCPXML using build2 API
	err = generateFCPXML(outputFile, wavFiles, audioDurations, audioDurationsFCP, jpgFiles, totalDuration)
	if err != nil {
		return fmt.Errorf("failed to generate FCPXML: %v", err)
	}

	fmt.Printf("Generated FCPXML: %s\n", outputFile)
	return nil
}

func getAudioFilesWithDurations(audioDir string) ([]string, map[string]float64, map[string]string, float64, error) {
	files, err := filepath.Glob(filepath.Join(audioDir, "*.wav"))
	if err != nil {
		return nil, nil, nil, 0, err
	}

	// Sort files naturally
	sort.Strings(files)

	// Get all durations in a single ffprobe call
	fcpDurations, err := utils.GetBatchAudioDurations(audioDir)
	if err != nil {
		return nil, nil, nil, 0, fmt.Errorf("failed to get batch durations: %v", err)
	}

	// Keep durations in both FCP and seconds format
	audioDurations := make(map[string]float64)
	audioDurationsFCP := make(map[string]string)
	var totalDuration float64
	for _, file := range files {
		fileName := filepath.Base(file)
		fcpDuration, exists := fcpDurations[fileName]
		if !exists {
			return nil, nil, nil, 0, fmt.Errorf("duration not found for file: %s", fileName)
		}
		
		// Store the original FCP duration (no conversion)
		audioDurationsFCP[file] = fcpDuration
		
		// Convert to seconds only for total calculation
		duration, err := convertFCPDurationToSeconds(fcpDuration)
		if err != nil {
			return nil, nil, nil, 0, fmt.Errorf("failed to convert duration for %s: %v", file, err)
		}
		audioDurations[file] = duration
		totalDuration += duration
	}

	return files, audioDurations, audioDurationsFCP, totalDuration, nil
}

func getImageFiles(imageDir string) ([]string, error) {
	files, err := filepath.Glob(filepath.Join(imageDir, "*.jpg"))
	if err != nil {
		return nil, err
	}

	// Sort files naturally
	sort.Strings(files)
	return files, nil
}

func convertFCPDurationToSeconds(durationStr string) (float64, error) {
	// Parse the FCP duration format "frames/24000s"
	// Example: "48048/24000s" means 48048 frames at 24000 units per second
	parts := strings.Split(durationStr, "/")
	if len(parts) != 2 || !strings.HasSuffix(parts[1], "s") {
		return 0, fmt.Errorf("invalid duration format: %s", durationStr)
	}

	frames, err := strconv.ParseFloat(parts[0], 64)
	if err != nil {
		return 0, err
	}

	timebase := strings.TrimSuffix(parts[1], "s")
	timebaseFloat, err := strconv.ParseFloat(timebase, 64)
	if err != nil {
		return 0, err
	}

	// Convert to seconds
	return frames / timebaseFloat, nil
}

func convertSecondsToFCPDuration(seconds float64) string {
	// Convert to frame count using the sequence time base (1001/24000s frame duration)
	// This means 24000/1001 frames per second ≈ 23.976 fps
	framesPerSecond := 24000.0 / 1001.0
	frames := int(seconds * framesPerSecond)
	
	// Format as rational using the sequence time base
	return fmt.Sprintf("%d/24000s", frames*1001)
}

func getAudioDurationInSeconds(audioPath string) (float64, error) {
	// Use the existing duration utility
	durationStr, err := utils.GetAudioDuration(audioPath)
	if err != nil {
		return 0, err
	}

	return convertFCPDurationToSeconds(durationStr)
}

func generateFCPXML(outputFile string, wavFiles []string, audioDurations map[string]float64, audioDurationsFCP map[string]string, jpgFiles []string, totalDuration float64) error {
	// Create new project builder
	pb, err := api.NewProjectBuilder(outputFile)
	if err != nil {
		return err
	}

	// Calculate frame-aligned timing for image distribution
	framesPerSecond := 24000.0 / 1001.0
	totalFrames := int(totalDuration * framesPerSecond)
	framesPerImage := totalFrames / len(jpgFiles)
	remainingFrames := totalFrames % len(jpgFiles)
	
	// Distribute audio files across video elements
	audioIndex := 0
	var currentAudioOffset float64
	
	// Add all images as video clips with nested audio
	for i, jpgFile := range jpgFiles {
		// First 'remainingFrames' images get one extra frame for perfect distribution
		frames := framesPerImage
		if i < remainingFrames {
			frames++
		}
		
		// Convert frames to FCP duration format
		imageDurationFCP := fmt.Sprintf("%d/24000s", frames*1001)
		
		// Determine which audio file(s) to nest in this video element
		var audioFile string
		if audioIndex < len(wavFiles) {
			audioFile = wavFiles[audioIndex]
			audioIndex++
		}
		
		// Get actual audio duration in FCP format if audio file is provided
		// The audio should use its FULL duration, not the video duration
		var audioDurationFCP string
		if audioFile != "" {
			// Use the original FCP duration directly (no double conversion)
			audioDurationFCP = audioDurationsFCP[audioFile]
		}
		
		// Add video clip with nested audio
		// Video uses imageDurationFCP (distributed timing), audio uses its full duration
		err = pb.AddVideoWithNestedAudioWithDurationSafe(jpgFile, audioFile, "", imageDurationFCP, audioDurationFCP)
		if err != nil {
			return fmt.Errorf("failed to add video clip with audio %s: %v", jpgFile, err)
		}
		
		// Update audio offset for next clip
		if audioFile != "" {
			audioDuration := audioDurations[audioFile]
			currentAudioOffset += audioDuration
		}
	}
	
	// If there are remaining audio files, add them to the last few video elements
	// This handles cases where there are more audio files than video files
	for audioIndex < len(wavFiles) {
		wavFile := wavFiles[audioIndex]
		// For now, we'll need a different approach for extra audio files
		// This could be handled by extending video durations or creating additional video elements
		fmt.Printf("Warning: Extra audio file not nested: %s\n", wavFile)
		audioIndex++
	}

	// Save the project
	return pb.Save()
}
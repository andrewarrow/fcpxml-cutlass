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
	wavFiles, audioDurations, totalDuration, err := getAudioFilesWithDurations(audioDir)
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
	err = generateFCPXML(outputFile, wavFiles, audioDurations, jpgFiles, totalDuration)
	if err != nil {
		return fmt.Errorf("failed to generate FCPXML: %v", err)
	}

	fmt.Printf("Generated FCPXML: %s\n", outputFile)
	return nil
}

func getAudioFilesWithDurations(audioDir string) ([]string, map[string]float64, float64, error) {
	files, err := filepath.Glob(filepath.Join(audioDir, "*.wav"))
	if err != nil {
		return nil, nil, 0, err
	}

	// Sort files naturally
	sort.Strings(files)

	// Get all durations in a single ffprobe call
	fcpDurations, err := utils.GetBatchAudioDurations(audioDir)
	if err != nil {
		return nil, nil, 0, fmt.Errorf("failed to get batch durations: %v", err)
	}

	// Convert FCP durations to seconds and calculate total
	audioDurations := make(map[string]float64)
	var totalDuration float64
	for _, file := range files {
		fileName := filepath.Base(file)
		fcpDuration, exists := fcpDurations[fileName]
		if !exists {
			return nil, nil, 0, fmt.Errorf("duration not found for file: %s", fileName)
		}
		
		duration, err := convertFCPDurationToSeconds(fcpDuration)
		if err != nil {
			return nil, nil, 0, fmt.Errorf("failed to convert duration for %s: %v", file, err)
		}
		audioDurations[file] = duration
		totalDuration += duration
	}

	return files, audioDurations, totalDuration, nil
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

func getAudioDurationInSeconds(audioPath string) (float64, error) {
	// Use the existing duration utility
	durationStr, err := utils.GetAudioDuration(audioPath)
	if err != nil {
		return 0, err
	}

	return convertFCPDurationToSeconds(durationStr)
}

func generateFCPXML(outputFile string, wavFiles []string, audioDurations map[string]float64, jpgFiles []string, totalDuration float64) error {
	// Create new project builder
	pb, err := api.NewProjectBuilder(outputFile)
	if err != nil {
		return err
	}

	// Calculate timing for image distribution
	imageDuration := totalDuration / float64(len(jpgFiles))
	
	// Track current time position
	var currentTime float64
	imageIndex := 0

	// Add each audio file with corresponding images
	for _, wavFile := range wavFiles {
		// Get duration from cache
		audioDuration := audioDurations[wavFile]

		// Calculate how many images should be used for this audio segment
		imagesForThisSegment := int((audioDuration / totalDuration) * float64(len(jpgFiles)))
		if imagesForThisSegment < 1 {
			imagesForThisSegment = 1
		}

		// Make sure we don't exceed available images
		if imageIndex+imagesForThisSegment > len(jpgFiles) {
			imagesForThisSegment = len(jpgFiles) - imageIndex
		}

		// Add clips for this audio segment
		for i := 0; i < imagesForThisSegment && imageIndex < len(jpgFiles); i++ {
			jpgFile := jpgFiles[imageIndex]
			
			// For the first clip in each segment, include the audio
			// For subsequent clips in the same segment, no audio (just image)
			if i == 0 {
				// First clip with audio
				err = pb.AddClipSafe(api.ClipConfig{
					VideoFile: jpgFile,
					AudioFile: wavFile,
					Text:      "",
				})
			} else {
				// Subsequent clips without audio (image only)
				err = pb.AddClipSafe(api.ClipConfig{
					VideoFile: jpgFile,
					AudioFile: "", // No audio for subsequent clips
					Text:      "",
				})
			}
			
			if err != nil {
				return fmt.Errorf("failed to add clip: %v", err)
			}

			imageIndex++
			currentTime += imageDuration
		}
	}

	// If there are remaining images, add them without audio
	for imageIndex < len(jpgFiles) {
		jpgFile := jpgFiles[imageIndex]
		
		err = pb.AddClipSafe(api.ClipConfig{
			VideoFile: jpgFile,
			AudioFile: "",
			Text:      "",
		})
		if err != nil {
			return fmt.Errorf("failed to add remaining clip: %v", err)
		}
		
		imageIndex++
	}

	// Save the project
	return pb.Save()
}
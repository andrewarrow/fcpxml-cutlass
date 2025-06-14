package keyframe

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

// ExtractKeyframes extracts all keyframes from a video file as JPEG images
func ExtractKeyframes(videoID string) error {
	// Construct input video path
	inputPath := filepath.Join("data", videoID+".mov")
	
	// Check if input video file exists
	if _, err := os.Stat(inputPath); os.IsNotExist(err) {
		return fmt.Errorf("video file does not exist: %s", inputPath)
	}
	
	// Create output directory
	outputDir := filepath.Join("data", videoID)
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %v", err)
	}
	
	// Construct output path pattern for keyframes
	outputPattern := filepath.Join(outputDir, "keyframe_%04d.jpg")
	
	// Run ffmpeg command to extract keyframes
	// -vf "select='eq(pict_type,I)'" selects only I-frames (keyframes)
	// -vsync vfr ensures variable frame rate to avoid duplicates
	cmd := exec.Command("ffmpeg", 
		"-i", inputPath,
		"-vf", "select='eq(pict_type,I)'",
		"-vsync", "vfr",
		"-q:v", "2", // High quality JPEG
		outputPattern)
	
	fmt.Printf("Extracting keyframes from %s to %s/\n", inputPath, outputDir)
	
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("ffmpeg command failed: %v\nOutput: %s", err, string(output))
	}
	
	fmt.Printf("Keyframes successfully extracted to: %s/\n", outputDir)
	return nil
}
package utils

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"strconv"
)

// getVideoDuration uses ffprobe to get video duration in FCP time format
func GetVideoDuration(videoPath string) (string, error) {
	cmd := exec.Command("ffprobe", "-v", "quiet", "-print_format", "json", "-show_format", videoPath)
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}
	
	var result struct {
		Format struct {
			Duration string `json:"duration"`
		} `json:"format"`
	}
	
	err = json.Unmarshal(output, &result)
	if err != nil {
		return "", err
	}
	
	// Parse duration as float seconds
	durationFloat, err := strconv.ParseFloat(result.Format.Duration, 64)
	if err != nil {
		return "", err
	}
	
	// Convert to frame count using the sequence time base (1001/24000s frame duration)
	// This means 24000/1001 frames per second ≈ 23.976 fps
	framesPerSecond := 24000.0 / 1001.0
	frames := int(durationFloat * framesPerSecond)
	
	// Format as rational using the sequence time base
	return fmt.Sprintf("%d/24000s", frames*1001), nil
}

// getAudioDuration uses ffprobe to get audio duration in FCP time format
func GetAudioDuration(audioPath string) (string, error) {
	cmd := exec.Command("ffprobe", "-v", "quiet", "-print_format", "json", "-show_format", audioPath)
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}
	
	var result struct {
		Format struct {
			Duration string `json:"duration"`
		} `json:"format"`
	}
	
	err = json.Unmarshal(output, &result)
	if err != nil {
		return "", err
	}
	
	// Parse duration as float seconds
	durationFloat, err := strconv.ParseFloat(result.Format.Duration, 64)
	if err != nil {
		return "", err
	}
	
	// Convert to frame count using the sequence time base (1001/24000s frame duration)
	// This means 24000/1001 frames per second ≈ 23.976 fps
	framesPerSecond := 24000.0 / 1001.0
	frames := int(durationFloat * framesPerSecond)
	
	// Format as rational using the sequence time base
	return fmt.Sprintf("%d/24000s", frames*1001), nil
}
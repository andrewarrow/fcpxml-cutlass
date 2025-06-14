package utils

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"strconv"
	"strings"
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

// GetBatchAudioDurations uses a single ffprobe command to get durations for all WAV files in a directory
func GetBatchAudioDurations(audioDir string) (map[string]string, error) {
	cmd := exec.Command("bash", "-c", fmt.Sprintf("cd '%s' && ffprobe -v quiet -show_entries format=duration -of csv=p=0 *.wav", audioDir))
	output, err := cmd.Output()
	if err != nil {
		return nil, err
	}
	
	// Get list of WAV files in the same order as ffprobe output
	cmd = exec.Command("bash", "-c", fmt.Sprintf("cd '%s' && ls *.wav", audioDir))
	filesOutput, err := cmd.Output()
	if err != nil {
		return nil, err
	}
	
	// Parse the outputs
	durations := strings.Split(strings.TrimSpace(string(output)), "\n")
	files := strings.Split(strings.TrimSpace(string(filesOutput)), "\n")
	
	if len(durations) != len(files) {
		return nil, fmt.Errorf("mismatch between number of files (%d) and durations (%d)", len(files), len(durations))
	}
	
	result := make(map[string]string)
	for i, file := range files {
		if i >= len(durations) {
			break
		}
		
		// Parse duration as float seconds
		durationFloat, err := strconv.ParseFloat(durations[i], 64)
		if err != nil {
			return nil, fmt.Errorf("failed to parse duration for %s: %v", file, err)
		}
		
		// Convert to frame count using the sequence time base (1001/24000s frame duration)
		// This means 24000/1001 frames per second ≈ 23.976 fps
		framesPerSecond := 24000.0 / 1001.0
		frames := int(durationFloat * framesPerSecond)
		
		// Format as rational using the sequence time base
		fcpDuration := fmt.Sprintf("%d/24000s", frames*1001)
		result[file] = fcpDuration
	}
	
	return result, nil
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
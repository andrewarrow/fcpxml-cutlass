package utils

import (
	"encoding/json"
	"fmt"
	"math"
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

// GetBatchAudioDurations uses parallel ffprobe calls to efficiently get durations for all WAV files
func GetBatchAudioDurations(audioDir string) (map[string]string, error) {
	// Use find + xargs to run ffprobe in parallel on all WAV files
	bashScript := fmt.Sprintf(`
cd '%s'
# Get filenames and durations in parallel
find . -name "*.wav" -print0 | xargs -0 -P 8 -I {} sh -c 'echo "{}" $(ffprobe -v quiet -show_entries format=duration -of csv=p=0 "{}")'
`, audioDir)
	
	cmd := exec.Command("bash", "-c", bashScript)
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("parallel ffprobe failed: %v", err)
	}
	
	result := make(map[string]string)
	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	
	for _, line := range lines {
		if line == "" {
			continue
		}
		
		parts := strings.Fields(line)
		if len(parts) != 2 {
			return nil, fmt.Errorf("unexpected output format: %s", line)
		}
		
		fileName := strings.TrimPrefix(parts[0], "./")
		durationStr := parts[1]
		
		// Parse duration as float seconds
		durationFloat, err := strconv.ParseFloat(durationStr, 64)
		if err != nil {
			return nil, fmt.Errorf("failed to parse duration for %s: %v", fileName, err)
		}
		
		// Convert to frame count using the sequence time base (1001/24000s frame duration)
		// This means 24000/1001 frames per second ≈ 23.976 fps
		framesPerSecond := 24000.0 / 1001.0
		frames := int(durationFloat * framesPerSecond)
		
		// Format as rational using the sequence time base
		fcpDuration := fmt.Sprintf("%d/24000s", frames*1001)
		result[fileName] = fcpDuration
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

// ConvertSecondsToFCPDuration converts seconds to FCPXML duration format with proper frame alignment
func ConvertSecondsToFCPDuration(seconds float64) string {
	// Convert to frame count using the sequence time base (1001/24000s frame duration)
	// This means 24000/1001 frames per second ≈ 23.976 fps
	framesPerSecond := 24000.0 / 1001.0
	exactFrames := seconds * framesPerSecond
	
	// Choose the frame count that gives the closest duration to the target
	floorFrames := int(math.Floor(exactFrames))
	ceilFrames := int(math.Ceil(exactFrames))
	
	floorDuration := float64(floorFrames) / framesPerSecond
	ceilDuration := float64(ceilFrames) / framesPerSecond
	
	var frames int
	if math.Abs(seconds-floorDuration) <= math.Abs(seconds-ceilDuration) {
		frames = floorFrames
	} else {
		frames = ceilFrames
	}
	
	// Format as rational using the sequence time base for frame boundary alignment
	return fmt.Sprintf("%d/24000s", frames*1001)
}
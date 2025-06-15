package vtt

import (
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// findAudioFile looks for corresponding audio/video file for the VTT
func findAudioFile(vttPath string) string {
	// Remove .vtt extension and try common video/audio extensions
	baseName := strings.TrimSuffix(vttPath, ".vtt")
	baseName = strings.TrimSuffix(baseName, ".en") // Remove language suffix if present

	extensions := []string{".mov", ".mp4", ".m4a", ".wav", ".mp3", ".mkv", ".avi"}

	for _, ext := range extensions {
		candidateFile := baseName + ext
		if _, err := os.Stat(candidateFile); err == nil {
			return candidateFile
		}
	}

	return ""
}

// detectSilenceGaps uses FFmpeg to analyze audio and find natural speech pauses
func detectSilenceGaps(audioFile string) []SilenceGap {
	// Use FFmpeg's silencedetect filter to find gaps
	cmd := exec.Command("ffmpeg", "-i", audioFile, "-af",
		"silencedetect=noise=-30dB:duration=0.3", "-f", "null", "-")

	output, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Printf("Warning: Could not analyze audio waveform: %v\n", err)
		return nil
	}

	return parseSilenceOutput(string(output))
}

// parseSilenceOutput extracts silence timestamps from FFmpeg output
func parseSilenceOutput(output string) []SilenceGap {
	var gaps []SilenceGap
	lines := strings.Split(output, "\n")

	silenceStartRegex := regexp.MustCompile(`silence_start: ([0-9.]+)`)
	silenceEndRegex := regexp.MustCompile(`silence_end: ([0-9.]+)`)

	var currentStart *time.Duration

	for _, line := range lines {
		if matches := silenceStartRegex.FindStringSubmatch(line); len(matches) > 1 {
			if seconds, err := strconv.ParseFloat(matches[1], 64); err == nil {
				start := time.Duration(seconds * float64(time.Second))
				currentStart = &start
			}
		} else if matches := silenceEndRegex.FindStringSubmatch(line); len(matches) > 1 && currentStart != nil {
			if seconds, err := strconv.ParseFloat(matches[1], 64); err == nil {
				end := time.Duration(seconds * float64(time.Second))
				duration := end - *currentStart

				// Only include meaningful pauses (300ms or longer)
				if duration >= 300*time.Millisecond {
					gaps = append(gaps, SilenceGap{
						Start:    *currentStart,
						End:      end,
						Duration: duration,
					})
				}
				currentStart = nil
			}
		}
	}

	return gaps
}

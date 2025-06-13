package build

import (
	"fmt"
	"strconv"
	"strings"
)

// calculateTimelineOffset parses existing spine content and calculates where the next clip should start
func calculateTimelineOffset(spineContent string) string {
	if strings.TrimSpace(spineContent) == "" {
		return "0s"
	}
	
	// Parse existing asset-clips to find the total timeline length
	totalDuration := calculateTotalDuration(spineContent)
	return totalDuration
}

// calculateTotalDuration parses spine content and calculates the total timeline duration
func calculateTotalDuration(spineContent string) string {
	if strings.TrimSpace(spineContent) == "" {
		return "0s"
	}
	
	// Find all duration values in both asset-clips and video elements
	totalFrames := 0
	
	// Use regex to find all duration attributes more precisely
	// This avoids double-counting when splitting by both tags
	lines := strings.Split(spineContent, "\n")
	for _, line := range lines {
		// Look for asset-clip, video, or ref-clip elements with duration
		if (strings.Contains(line, "asset-clip") || strings.Contains(line, "<video") || strings.Contains(line, "ref-clip")) && strings.Contains(line, "duration=") {
			// Extract duration value
			start := strings.Index(line, "duration=\"") + 10
			if start > 9 {
				end := strings.Index(line[start:], "\"")
				if end > 0 {
					durationStr := line[start : start+end]
					// Parse "frames/24000s" format
					if strings.HasSuffix(durationStr, "/24000s") {
						framesStr := strings.TrimSuffix(durationStr, "/24000s")
						if frames, err := strconv.Atoi(framesStr); err == nil {
							totalFrames += frames
						}
					}
				}
			}
		}
	}
	
	if totalFrames == 0 {
		return "0s"
	}
	
	return fmt.Sprintf("%d/24000s", totalFrames)
}
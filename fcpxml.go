package main

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
)

type VTTSegment struct {
	StartTime time.Duration
	EndTime   time.Duration
	Text      string
}

type Clip struct {
	StartTime time.Duration
	EndTime   time.Duration
	Duration  time.Duration
	Text      string
	ClipNum   int
}

func parseVTTTime(timeStr string) (time.Duration, error) {
	// Parse format like "00:00:02.350"
	parts := strings.Split(timeStr, ":")
	if len(parts) != 3 {
		return 0, fmt.Errorf("invalid time format: %s", timeStr)
	}
	
	hours, _ := strconv.Atoi(parts[0])
	minutes, _ := strconv.Atoi(parts[1])
	secondsParts := strings.Split(parts[2], ".")
	seconds, _ := strconv.Atoi(secondsParts[0])
	milliseconds := 0
	if len(secondsParts) > 1 {
		// Pad or truncate to 3 digits
		msStr := secondsParts[1]
		if len(msStr) > 3 {
			msStr = msStr[:3]
		} else {
			for len(msStr) < 3 {
				msStr += "0"
			}
		}
		milliseconds, _ = strconv.Atoi(msStr)
	}
	
	return time.Duration(hours)*time.Hour + 
		   time.Duration(minutes)*time.Minute + 
		   time.Duration(seconds)*time.Second + 
		   time.Duration(milliseconds)*time.Millisecond, nil
}

func parseVTTFile(vttPath string) ([]VTTSegment, error) {
	file, err := os.Open(vttPath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var segments []VTTSegment
	scanner := bufio.NewScanner(file)
	
	// Regex to match timestamp lines like "00:00:00.160 --> 00:00:02.350"
	timeRegex := regexp.MustCompile(`(\d{2}:\d{2}:\d{2}\.\d{3})\s+-->\s+(\d{2}:\d{2}:\d{2}\.\d{3})`)
	
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		
		if timeRegex.MatchString(line) {
			matches := timeRegex.FindStringSubmatch(line)
			if len(matches) >= 3 {
				startTime, err1 := parseVTTTime(matches[1])
				endTime, err2 := parseVTTTime(matches[2])
				
				if err1 == nil && err2 == nil {
					// Read the next line for text content
					var textLines []string
					for scanner.Scan() {
						textLine := strings.TrimSpace(scanner.Text())
						if textLine == "" {
							break
						}
						// Clean up VTT formatting tags
						cleanText := regexp.MustCompile(`<[^>]*>`).ReplaceAllString(textLine, "")
						cleanText = regexp.MustCompile(`<\d{2}:\d{2}:\d{2}\.\d{3}>.*?</c>`).ReplaceAllString(cleanText, "")
						if cleanText != "" {
							textLines = append(textLines, cleanText)
						}
					}
					
					if len(textLines) > 0 {
						segments = append(segments, VTTSegment{
							StartTime: startTime,
							EndTime:   endTime,
							Text:      strings.Join(textLines, " "),
						})
					}
				}
			}
		}
	}
	
	return segments, scanner.Err()
}

func segmentIntoClips(segments []VTTSegment, minDuration, maxDuration time.Duration) []Clip {
	var clips []Clip
	clipNum := 1
	
	// Sort segments by start time
	sort.Slice(segments, func(i, j int) bool {
		return segments[i].StartTime < segments[j].StartTime
	})
	
	i := 0
	for i < len(segments) {
		clipStart := segments[i].StartTime
		var clipTexts []string
		clipTexts = append(clipTexts, segments[i].Text)
		
		// Extend clip duration by adding consecutive segments
		j := i + 1
		for j < len(segments) {
			currentDuration := segments[j-1].EndTime - clipStart
			
			// If adding this segment would exceed max duration, stop
			if currentDuration >= maxDuration {
				break
			}
			
			// If we have minimum duration and there's a natural break, stop
			if currentDuration >= minDuration {
				// Look for sentence endings or pauses
				lastText := segments[j-1].Text
				if strings.HasSuffix(lastText, ".") || strings.HasSuffix(lastText, "!") || strings.HasSuffix(lastText, "?") {
					break
				}
			}
			
			clipTexts = append(clipTexts, segments[j].Text)
			j++
		}
		
		clipEnd := segments[j-1].EndTime
		clipDuration := clipEnd - clipStart
		
		// Ensure minimum duration
		if clipDuration < minDuration && j < len(segments) {
			clipEnd = clipStart + minDuration
		}
		
		clips = append(clips, Clip{
			StartTime: clipStart,
			EndTime:   clipEnd,
			Duration:  clipEnd - clipStart,
			Text:      strings.Join(clipTexts, " "),
			ClipNum:   clipNum,
		})
		
		clipNum++
		i = j
	}
	
	return clips
}

func formatDurationForFCPXML(d time.Duration) string {
	// Convert to frame-aligned format for 30fps video
	// 30000 frames per second with 1001/30000s frame duration
	totalFrames := int64(d.Seconds() * 30000 / 1001)
	// Ensure frame alignment
	return fmt.Sprintf("%d/30000s", totalFrames*1001)
}

func generateClipFCPXML(clips []Clip, videoPath, outputPath string) error {
	absVideoPath, err := filepath.Abs(videoPath)
	if err != nil {
		return err
	}
	
	videoName := filepath.Base(absVideoPath)
	nameWithoutExt := strings.TrimSuffix(videoName, filepath.Ext(videoName))
	
	// Calculate total duration
	var totalDuration time.Duration
	for _, clip := range clips {
		totalDuration += clip.Duration + 2*time.Second // Add 2s for title card
	}
	
	xml := fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE fcpxml>
<fcpxml version="1.11">
    <resources>
        <format id="r1" name="FFVideoFormat1080p30" frameDuration="1001/30000s" width="1920" height="1080" colorSpace="1-1-1 (Rec. 709)"/>
        <asset id="r2" name="%s" uid="%s" start="0s" hasVideo="1" format="r1" hasAudio="1" audioSources="1" audioChannels="2" duration="%s">
            <media-rep kind="original-media" sig="%s" src="file://%s"/>
        </asset>
    </resources>
    <library>
        <event name="Auto Generated Clips">
            <project name="%s Clips">
                <sequence format="r1" duration="%s" tcStart="0s" tcFormat="NDF" audioLayout="stereo" audioRate="48k">
                    <spine>`, 
		nameWithoutExt, absVideoPath, formatDurationForFCPXML(totalDuration), 
		absVideoPath, absVideoPath, nameWithoutExt, formatDurationForFCPXML(totalDuration))
	
	currentOffset := time.Duration(0)
	
	for _, clip := range clips {
		// Video clip
		xml += fmt.Sprintf(`
                        <asset-clip ref="r2" offset="%s" name="%s Clip %d" start="%s" duration="%s" format="r1" tcFormat="NDF">
                        </asset-clip>`, 
			formatDurationForFCPXML(currentOffset),
			nameWithoutExt, clip.ClipNum,
			formatDurationForFCPXML(clip.StartTime),
			formatDurationForFCPXML(clip.Duration))
		
		currentOffset += clip.Duration
		
		// Title card with inline definition
		xml += fmt.Sprintf(`
                        <gap name="Gap" offset="%s" duration="%s">
                            <title lane="1" offset="0s" name="Clip %d Title" duration="%s">
                                <param name="Position" key="9999/999166631/999166633/1/100/101" value="0 0"/>
                                <param name="Flat" key="9999/999166631/999166633/1/999166650/999166651" value="1"/>
                                <param name="Alignment" key="9999/999166631/999166633/2/354/999169573/401" value="1 (Center)"/>
                                <text>
                                    <text-style ref="ts%d">Clip %d</text-style>
                                </text>
                                <text-style-def id="ts%d">
                                    <text-style font="Helvetica" fontSize="72" fontFace="Bold" fontColor="1 1 1 1" alignment="center"/>
                                </text-style-def>
                            </title>
                        </gap>`, 
			formatDurationForFCPXML(currentOffset),
			formatDurationForFCPXML(2*time.Second),
			clip.ClipNum, 
			formatDurationForFCPXML(2*time.Second),
			clip.ClipNum, clip.ClipNum, clip.ClipNum)
		
		currentOffset += 2 * time.Second
	}
	
	xml += `
                    </spine>
                </sequence>
            </project>
        </event>
    </library>
</fcpxml>`
	
	return os.WriteFile(outputPath, []byte(xml), 0644)
}

func breakIntoLogicalParts(youtubeID string) error {
	vttPath := fmt.Sprintf("%s.en.vtt", youtubeID)
	videoPath := fmt.Sprintf("%s.mov", youtubeID)
	outputPath := fmt.Sprintf("%s_clips.fcpxml", youtubeID)
	
	// Check if files exist
	if _, err := os.Stat(vttPath); os.IsNotExist(err) {
		return fmt.Errorf("VTT file not found: %s", vttPath)
	}
	if _, err := os.Stat(videoPath); os.IsNotExist(err) {
		return fmt.Errorf("video file not found: %s", videoPath)
	}
	
	// Parse VTT file
	fmt.Printf("Parsing VTT file: %s\n", vttPath)
	segments, err := parseVTTFile(vttPath)
	if err != nil {
		return fmt.Errorf("error parsing VTT file: %v", err)
	}
	
	fmt.Printf("Found %d VTT segments\n", len(segments))
	
	// Segment into logical clips (6-18 seconds)
	minDuration := 6 * time.Second
	maxDuration := 18 * time.Second
	clips := segmentIntoClips(segments, minDuration, maxDuration)
	
	fmt.Printf("Generated %d clips\n", len(clips))
	for i, clip := range clips {
		fmt.Printf("Clip %d: %v - %v (%.1fs) - %s\n", 
			i+1, clip.StartTime, clip.EndTime, clip.Duration.Seconds(), 
			clip.Text[:min(50, len(clip.Text))])
	}
	
	// Generate FCPXML
	fmt.Printf("Generating FCPXML: %s\n", outputPath)
	err = generateClipFCPXML(clips, videoPath, outputPath)
	if err != nil {
		return fmt.Errorf("error generating FCPXML: %v", err)
	}
	
	fmt.Printf("Successfully generated %s with %d clips\n", outputPath, len(clips))
	return nil
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

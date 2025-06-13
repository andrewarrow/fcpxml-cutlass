package speech

import (
	"bufio"
	"crypto/md5"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"text/template"
)

// generateVideoUID generates a unique identifier for the video file based on its content
func generateVideoUID(videoPath string) (string, error) {
	file, err := os.Open(videoPath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	hash := md5.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", err
	}

	// Convert to uppercase hex string like FCP UIDs
	return fmt.Sprintf("%X", hash.Sum(nil)), nil
}

// isPNGImage checks if the file is a PNG image
func isPNGImage(filePath string) bool {
	ext := strings.ToLower(filepath.Ext(filePath))
	return ext == ".png"
}

// getMediaDuration gets the duration of a media file using ffprobe for video, or returns 20 seconds for PNG images
func getMediaDuration(mediaPath string) (string, string, error) {
	if isPNGImage(mediaPath) {
		// For PNG images, use a fixed 20-second duration
		duration := 20.0
		
		// Convert to FCP asset format (frames/44100s)
		assetFrames := int64(duration * 44100)
		assetDuration := fmt.Sprintf("%d/44100s", assetFrames)

		// Convert to FCP clip format (frames/600s) aligned to frame boundaries
		// Frame duration is 20/600s, so we need to round to multiples of 20
		clipFrames := int64(duration * 600)
		// Round to nearest frame boundary (multiple of 20)
		clipFrames = (clipFrames / 20) * 20
		clipDuration := fmt.Sprintf("%d/600s", clipFrames)

		return assetDuration, clipDuration, nil
	}

	// For video files, use ffprobe
	cmd := exec.Command("ffprobe", "-v", "quiet", "-print_format", "json", "-show_entries", "format=duration", mediaPath)
	output, err := cmd.Output()
	if err != nil {
		return "", "", fmt.Errorf("failed to run ffprobe: %v", err)
	}

	var result struct {
		Format struct {
			Duration string `json:"duration"`
		} `json:"format"`
	}

	if err := json.Unmarshal(output, &result); err != nil {
		return "", "", fmt.Errorf("failed to parse ffprobe output: %v", err)
	}

	duration, err := strconv.ParseFloat(result.Format.Duration, 64)
	if err != nil {
		return "", "", fmt.Errorf("failed to parse duration: %v", err)
	}

	// Convert to FCP asset format (frames/44100s)
	assetFrames := int64(duration * 44100)
	assetDuration := fmt.Sprintf("%d/44100s", assetFrames)

	// Convert to FCP clip format (frames/600s) aligned to frame boundaries
	// Frame duration is 20/600s, so we need to round to multiples of 20
	clipFrames := int64(duration * 600)
	// Round to nearest frame boundary (multiple of 20)
	clipFrames = (clipFrames / 20) * 20
	clipDuration := fmt.Sprintf("%d/600s", clipFrames)

	return assetDuration, clipDuration, nil
}

type TextElement struct {
	Text                string
	Index               int
	Offset              string
	Duration            string
	YPosition           int
	Lane                int
	ReverseStartTimeNano string
	ReverseEndTimeNano   string
}

type SpeechData struct {
	TextElements      []TextElement
	VideoPath         string
	VideoUID          string
	VideoDuration     string
	VideoClipDuration string
	ReverseStartTime  string
	ReverseEndTime    string
	IsStillImage      bool
}

func GenerateSpeechFCPXML(inputFile, outputFile, videoFile string) error {
	// Read the text file
	file, err := os.Open(inputFile)
	if err != nil {
		return fmt.Errorf("failed to open input file: %v", err)
	}
	defer file.Close()

	var lines []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line != "" {
			lines = append(lines, line)
		}
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("failed to read input file: %v", err)
	}

	if len(lines) == 0 {
		return fmt.Errorf("no text found in input file")
	}

	// Create text elements with staggered timing
	var textElements []TextElement
	baseOffsetFrames := 4900         // Starting offset in timebase units (4900/3000s)
	pauseDurationFrames := 6000      // 2 seconds = 6000/3000s between each text appearance
	timeBase := 3000                 // From format frameDuration="100/3000s"
	yPositionBase := 800             // Base Y position
	ySpacing := 300                  // Vertical spacing between text elements
	
	// Calculate reverse animation timing
	// Last text appears at: baseOffsetFrames + (len(lines)-1) * pauseDurationFrames
	lastTextOffsetFrames := baseOffsetFrames + ((len(lines) - 1) * pauseDurationFrames)
	pauseAfterLastText := 6000       // 2 seconds pause after last text appears
	reverseAnimationDuration := 4000 // 1.33 seconds for reverse animation
	
	reverseStartFrames := lastTextOffsetFrames + pauseAfterLastText
	reverseEndFrames := reverseStartFrames + reverseAnimationDuration
	
	reverseStartTime := fmt.Sprintf("%d/%ds", reverseStartFrames, timeBase)
	reverseEndTime := fmt.Sprintf("%d/%ds", reverseEndFrames, timeBase)
	
	// Convert to nanoseconds for text animation (matching the existing format)
	reverseStartNano := fmt.Sprintf("%d/1000000000s", (reverseStartFrames * 1000000000) / timeBase)
	reverseEndNano := fmt.Sprintf("%d/1000000000s", (reverseEndFrames * 1000000000) / timeBase)

	for i, line := range lines {
		offsetFrames := baseOffsetFrames + (i * pauseDurationFrames)
		yPos := yPositionBase - (i * ySpacing) // Stack text elements vertically
		lane := -(i + 1)                       // Assign negative lanes (-1, -2, -3, -4 for items)
		
		// Calculate duration so each title ends just before the slide-back animation
		// Duration = reverseStartFrames - offsetFrames
		durationFrames := reverseStartFrames - offsetFrames
		duration := fmt.Sprintf("%d/%d", durationFrames, timeBase)

		textElements = append(textElements, TextElement{
			Text:                 line,
			Index:                i + 1,
			Offset:               fmt.Sprintf("%d/%d", offsetFrames, timeBase),
			Duration:             duration,
			YPosition:            yPos,
			Lane:                 lane,
			ReverseStartTimeNano: reverseStartNano,
			ReverseEndTimeNano:   reverseEndNano,
		})
	}

	// Get absolute path for video file
	absVideoPath, err := filepath.Abs(videoFile)
	if err != nil {
		return fmt.Errorf("failed to get absolute path for video file: %v", err)
	}

	// Generate unique UID for the media file
	videoUID, err := generateVideoUID(videoFile)
	if err != nil {
		return fmt.Errorf("failed to generate media UID: %v", err)
	}

	// Get media duration
	videoDuration, videoClipDuration, err := getMediaDuration(videoFile)
	if err != nil {
		return fmt.Errorf("failed to get media duration: %v", err)
	}

	// Create the speech data
	speechData := SpeechData{
		TextElements:      textElements,
		VideoPath:         "file://" + absVideoPath,
		VideoUID:          videoUID,
		VideoDuration:     videoDuration,
		VideoClipDuration: videoClipDuration,
		ReverseStartTime:  reverseStartTime,
		ReverseEndTime:    reverseEndTime,
		IsStillImage:      isPNGImage(videoFile),
	}

	// Read the template
	templatePath := "templates/slide.fcpxml"
	tmplContent, err := os.ReadFile(templatePath)
	if err != nil {
		return fmt.Errorf("failed to read template file: %v", err)
	}

	// Parse and execute the template
	tmpl, err := template.New("speech").Parse(string(tmplContent))
	if err != nil {
		return fmt.Errorf("failed to parse template: %v", err)
	}

	// Ensure output directory exists
	outputDir := filepath.Dir(outputFile)
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %v", err)
	}

	// Create output file
	outFile, err := os.Create(outputFile)
	if err != nil {
		return fmt.Errorf("failed to create output file: %v", err)
	}
	defer outFile.Close()

	// Execute template
	if err := tmpl.Execute(outFile, speechData); err != nil {
		return fmt.Errorf("failed to execute template: %v", err)
	}

	return nil
}
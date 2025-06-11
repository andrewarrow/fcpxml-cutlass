package speech

import (
	"bufio"
	"crypto/md5"
	"fmt"
	"io"
	"os"
	"path/filepath"
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

type TextElement struct {
	Text      string
	Index     int
	Offset    string
	YPosition int
	Lane      int
}

type SpeechData struct {
	TextElements []TextElement
	VideoPath    string
	VideoUID     string
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

	for i, line := range lines {
		offsetFrames := baseOffsetFrames + (i * pauseDurationFrames)
		yPos := yPositionBase - (i * ySpacing) // Stack text elements vertically
		lane := len(lines) - i                 // Assign lanes in descending order (3, 2, 1 for 3 items)

		textElements = append(textElements, TextElement{
			Text:      line,
			Index:     i + 1,
			Offset:    fmt.Sprintf("%d/%d", offsetFrames, timeBase),
			YPosition: yPos,
			Lane:      lane,
		})
	}

	// Get absolute path for video file
	absVideoPath, err := filepath.Abs(videoFile)
	if err != nil {
		return fmt.Errorf("failed to get absolute path for video file: %v", err)
	}

	// Generate unique UID for the video file
	videoUID, err := generateVideoUID(videoFile)
	if err != nil {
		return fmt.Errorf("failed to generate video UID: %v", err)
	}

	// Create the speech data
	speechData := SpeechData{
		TextElements: textElements,
		VideoPath:    "file://" + absVideoPath,
		VideoUID:     videoUID,
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
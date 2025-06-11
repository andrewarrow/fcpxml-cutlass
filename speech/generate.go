package speech

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/template"
)

type TextElement struct {
	Text      string
	Index     int
	Offset    string
	YPosition int
	Lane      int
}

type SpeechData struct {
	TextElements []TextElement
}

func GenerateSpeechFCPXML(inputFile, outputFile string) error {
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
	ySpacing := 200                  // Vertical spacing between text elements

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

	// Create the speech data
	speechData := SpeechData{
		TextElements: textElements,
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
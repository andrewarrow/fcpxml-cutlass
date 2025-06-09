package fcp

import (
	"fmt"
	"io/ioutil"
	"math/rand"
	"path/filepath"
	"strings"
	"time"
)

func GenerateTop5FCPXML(templatePath string, videoIDs []string, name, outputPath string) error {
	// Read the template file
	content, err := ioutil.ReadFile(templatePath)
	if err != nil {
		return fmt.Errorf("failed to read template file: %v", err)
	}

	templateStr := string(content)

	// Replace the name with newlines between words
	nameWords := strings.Fields(name)
	
	// Replace both instances of "Amelia" and "Dimoldenber" with the new name
	result := templateStr
	if len(nameWords) >= 1 {
		result = strings.ReplaceAll(result, "Amelia", nameWords[0])
	}
	if len(nameWords) >= 2 {
		result = strings.ReplaceAll(result, "Dimoldenber", nameWords[1])
	} else if len(nameWords) == 1 {
		// If only one word, replace Dimoldenber with empty but keep structure
		result = strings.ReplaceAll(result, "Dimoldenber", "")
	}

	// Create numbers 1, 2, 3 by duplicating the existing NUMBER 4 and NUMBER 5 patterns
	result = expandToFullTop5(result, videoIDs)

	// Generate random clips from the downloaded videos
	if len(videoIDs) > 0 {
		result = replaceVideoClipsWithRandom(result, videoIDs)
	}

	// Write the result to the output file
	err = ioutil.WriteFile(outputPath, []byte(result), 0644)
	if err != nil {
		return fmt.Errorf("failed to write output file: %v", err)
	}

	return nil
}

func replaceVideoClipsWithRandom(fcpxmlContent string, videoIDs []string) string {
	// For now, this is a placeholder - we would need to:
	// 1. Parse the existing video clips in the FCPXML
	// 2. Generate random start times (within 30 seconds of total duration)
	// 3. Replace the video asset references with paths to ./data/{id}.mov
	// 4. Update the timing and duration to be 30 seconds
	
	// This is a simplified implementation that just updates the content
	// In a full implementation, we'd need to properly parse and modify the XML structure
	
	rand.Seed(time.Now().UnixNano())
	
	// Replace existing video asset with random video from our list
	if len(videoIDs) > 0 {
		randomID := videoIDs[rand.Intn(len(videoIDs))]
		dataPath := fmt.Sprintf("./data/%s.mov", randomID)
		
		// This is a basic replacement - in practice we'd need more sophisticated XML manipulation
		content := strings.ReplaceAll(fcpxmlContent, "ben.mov", filepath.Base(dataPath))
		return content
	}
	
	return fcpxmlContent
}

func expandToFullTop5(fcpxmlContent string, videoIDs []string) string {
	// This is a simplified approach - we'll find the NUMBER 5 section and duplicate it
	// to create NUMBER 1, 2, 3 before the existing NUMBER 4 and 5
	
	// Find the NUMBER 5 title section
	number5Start := strings.Index(fcpxmlContent, `<title ref="r8" lane="1" offset="3600s" name="NUMBER 5 - Basic 3D"`)
	if number5Start == -1 {
		return fcpxmlContent // Return unchanged if pattern not found
	}
	
	// Find the end of the NUMBER 5 section (look for the next major closing tag)
	searchStart := number5Start
	titleDepth := 0
	number5End := -1
	
	for i := searchStart; i < len(fcpxmlContent); i++ {
		if i+7 < len(fcpxmlContent) && fcpxmlContent[i:i+7] == "<title " {
			titleDepth++
		} else if i+8 < len(fcpxmlContent) && fcpxmlContent[i:i+8] == "</title>" {
			titleDepth--
			if titleDepth == 0 {
				number5End = i + 8
				break
			}
		}
	}
	
	if number5End == -1 {
		return fcpxmlContent // Return unchanged if end not found
	}
	
	// Extract the NUMBER 5 section
	number5Section := fcpxmlContent[number5Start:number5End]
	
	// Create sections for numbers 1, 2, 3 by duplicating and modifying the NUMBER 5 section
	var newSections strings.Builder
	
	for num := 1; num <= 3; num++ {
		numStr := fmt.Sprintf("%d", num)
		newSection := strings.ReplaceAll(number5Section, "NUMBER 5", fmt.Sprintf("NUMBER %s", numStr))
		newSection = strings.ReplaceAll(newSection, ">5</text-style>", fmt.Sprintf(">%s</text-style>", numStr))
		
		// Update the unique IDs and references to avoid conflicts
		newSection = strings.ReplaceAll(newSection, "ts8", fmt.Sprintf("ts%d_num", num))
		newSection = strings.ReplaceAll(newSection, "ts9", fmt.Sprintf("ts%d_digit", num))
		
		newSections.WriteString(newSection)
		newSections.WriteString("\n")
	}
	
	// Insert the new sections before the existing NUMBER 5 section
	result := fcpxmlContent[:number5Start] + newSections.String() + fcpxmlContent[number5Start:]
	
	return result
}
package fcp

import (
	"fmt"
	"io/ioutil"
	"strings"
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
		// Remove the extra "g" after the second name
		result = strings.ReplaceAll(result, nameWords[1]+"</text-style>\n                                <text-style ref=\"ts7\">g</text-style>", nameWords[1]+"</text-style>")
		result = strings.ReplaceAll(result, nameWords[1]+"</text-style>\n                                <text-style ref=\"ts16\">g</text-style>", nameWords[1]+"</text-style>")
	} else if len(nameWords) == 1 {
		// If only one word, replace Dimoldenber with empty but keep structure
		result = strings.ReplaceAll(result, "Dimoldenber", "")
		result = strings.ReplaceAll(result, "</text-style>\n                                <text-style ref=\"ts7\">g</text-style>", "</text-style>")
		result = strings.ReplaceAll(result, "</text-style>\n                                <text-style ref=\"ts16\">g</text-style>", "</text-style>")
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
	if len(videoIDs) == 0 {
		return fcpxmlContent
	}
	
	result := fcpxmlContent
	
	// The order in the FCPXML is now 5,4,3,2,1
	// We want to use videoIDs[0] for NUMBER 5, videoIDs[1] for NUMBER 4, etc.
	
	// Replace ben.mov with the first video ID (for NUMBER 5)
	if len(videoIDs) > 0 {
		result = strings.ReplaceAll(result, "ben.mov", fmt.Sprintf("%s.mov", videoIDs[0]))
	}
	
	// For a more complete implementation, we would:
	// 1. Create unique asset IDs for each video
	// 2. Update all the video references in each NUMBER section
	// 3. Ensure each number uses a different video
	
	// For now, this basic replacement will use the first video for all sections
	// A full implementation would require proper XML parsing and manipulation
	
	return result
}

func expandToFullTop5(fcpxmlContent string, videoIDs []string) string {
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
	
	// Now find the NUMBER 4 section to understand the full pattern
	number4Start := strings.Index(fcpxmlContent, `<title ref="r8" lane="1" offset="3600s" name="NUMBER 4 - Basic 3D"`)
	if number4Start == -1 {
		return fcpxmlContent // Return unchanged if pattern not found
	}
	
	// Find the end of the NUMBER 4 section
	searchStart = number4Start
	titleDepth = 0
	number4End := -1
	
	for i := searchStart; i < len(fcpxmlContent); i++ {
		if i+7 < len(fcpxmlContent) && fcpxmlContent[i:i+7] == "<title " {
			titleDepth++
		} else if i+8 < len(fcpxmlContent) && fcpxmlContent[i:i+8] == "</title>" {
			titleDepth--
			if titleDepth == 0 {
				number4End = i + 8
				break
			}
		}
	}
	
	if number4End == -1 {
		return fcpxmlContent // Return unchanged if end not found
	}
	
	number4Section := fcpxmlContent[number4Start:number4End]
	
	// Create sections for numbers 3, 2, 1 in the right order
	// We want final order: 5, 4, 3, 2, 1
	var allNumberSections strings.Builder
	
	// Start with NUMBER 5 (keep original)
	allNumberSections.WriteString(number5Section)
	allNumberSections.WriteString("\n")
	
	// Add NUMBER 4 (keep original) 
	allNumberSections.WriteString(number4Section)
	allNumberSections.WriteString("\n")
	
	// Add NUMBER 3, 2, 1 by duplicating NUMBER 5 section
	for num := 3; num >= 1; num-- {
		numStr := fmt.Sprintf("%d", num)
		newSection := strings.ReplaceAll(number5Section, "NUMBER 5", fmt.Sprintf("NUMBER %s", numStr))
		newSection = strings.ReplaceAll(newSection, ">5</text-style>", fmt.Sprintf(">%s</text-style>", numStr))
		
		// Update the unique IDs and references to avoid conflicts
		newSection = strings.ReplaceAll(newSection, "ts8", fmt.Sprintf("ts%d_num", num))
		newSection = strings.ReplaceAll(newSection, "ts9", fmt.Sprintf("ts%d_digit", num))
		
		allNumberSections.WriteString(newSection)
		allNumberSections.WriteString("\n")
	}
	
	// Replace both original sections with the new ordered sections
	result := fcpxmlContent[:number5Start] + allNumberSections.String() + fcpxmlContent[number4End:]
	
	return result
}
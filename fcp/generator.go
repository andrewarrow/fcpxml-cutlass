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
	
	// Replace the ben asset definition and all references
	// First, we need to create new asset definitions for each video ID
	
	// Find the original ben asset definition
	benAssetStart := strings.Index(result, `<asset id="r10" name="ben"`)
	if benAssetStart == -1 {
		return result // Return if ben asset not found
	}
	
	// Find the end of the ben asset
	benAssetEnd := strings.Index(result[benAssetStart:], "</asset>")
	if benAssetEnd == -1 {
		return result
	}
	benAssetEnd += benAssetStart + 8 // Include </asset>
	
	// Extract the original ben asset for use as template
	originalBenAsset := result[benAssetStart:benAssetEnd]
	
	// Create new asset definitions for each video ID (5,4,3,2,1 order)
	var newAssets strings.Builder
	
	for i := 0; i < 5 && i < len(videoIDs); i++ {
		videoID := videoIDs[i]
		assetID := fmt.Sprintf("r1%d", i) // r10, r11, r12, r13, r14
		
		// Create new asset by modifying the template
		newAsset := strings.ReplaceAll(originalBenAsset, `id="r10"`, fmt.Sprintf(`id="%s"`, assetID))
		newAsset = strings.ReplaceAll(newAsset, `name="ben"`, fmt.Sprintf(`name="%s"`, videoID))
		newAsset = strings.ReplaceAll(newAsset, "ben.mov", fmt.Sprintf("%s.mov", videoID))
		
		// Update duration to be longer (30 seconds = 30*320000/320000s for 30s at 25fps)
		// 30 seconds = 960000/32000s 
		newAsset = strings.ReplaceAll(newAsset, `duration="257945/12800s"`, `duration="960000/32000s"`)
		
		newAssets.WriteString(newAsset)
		newAssets.WriteString("\n        ")
	}
	
	// Replace the original ben asset with all new assets
	result = result[:benAssetStart] + newAssets.String() + result[benAssetEnd:]
	
	// Now update ALL asset-clip references from "ben" to proper video names
	// First, replace all remaining "ben" names with the first video ID
	if len(videoIDs) > 0 {
		result = strings.ReplaceAll(result, `name="ben"`, fmt.Sprintf(`name="%s"`, videoIDs[0]))
	}
	
	return result
}

func expandToFullTop5(fcpxmlContent string, videoIDs []string) string {
	// Find the NUMBER 5 title section
	number5Start := strings.Index(fcpxmlContent, `<title ref="r8" lane="1" offset="3600s" name="NUMBER 5 - Basic 3D"`)
	if number5Start == -1 {
		return fcpxmlContent // Return unchanged if pattern not found
	}
	
	// Find the end of the NUMBER 5 section including its asset-clip
	// Look for the pattern: </title>\n        <asset-clip ref="r10"...
	searchStart := number5Start
	titleDepth := 0
	titleEnd := -1
	
	for i := searchStart; i < len(fcpxmlContent); i++ {
		if i+7 < len(fcpxmlContent) && fcpxmlContent[i:i+7] == "<title " {
			titleDepth++
		} else if i+8 < len(fcpxmlContent) && fcpxmlContent[i:i+8] == "</title>" {
			titleDepth--
			if titleDepth == 0 {
				titleEnd = i + 8
				break
			}
		}
	}
	
	// Now find the asset-clip that follows this title
	number5End := titleEnd
	if titleEnd != -1 {
		// Look for the asset-clip that should follow
		assetClipStart := strings.Index(fcpxmlContent[titleEnd:], "<asset-clip ref=\"r10\"")
		if assetClipStart != -1 {
			assetClipStart += titleEnd
			assetClipEnd := strings.Index(fcpxmlContent[assetClipStart:], "/>")
			if assetClipEnd != -1 {
				number5End = assetClipStart + assetClipEnd + 2
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
	
	// Find the end of the NUMBER 4 section including its asset-clip
	searchStart = number4Start
	titleDepth = 0
	title4End := -1
	
	for i := searchStart; i < len(fcpxmlContent); i++ {
		if i+7 < len(fcpxmlContent) && fcpxmlContent[i:i+7] == "<title " {
			titleDepth++
		} else if i+8 < len(fcpxmlContent) && fcpxmlContent[i:i+8] == "</title>" {
			titleDepth--
			if titleDepth == 0 {
				title4End = i + 8
				break
			}
		}
	}
	
	// Find the asset-clip that follows NUMBER 4 title
	number4End := title4End
	if title4End != -1 {
		assetClipStart := strings.Index(fcpxmlContent[title4End:], "<asset-clip ref=\"r10\"")
		if assetClipStart != -1 {
			assetClipStart += title4End
			assetClipEnd := strings.Index(fcpxmlContent[assetClipStart:], "/>")
			if assetClipEnd != -1 {
				number4End = assetClipStart + assetClipEnd + 2
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
	
	// Add NUMBER 4 (modify to use second video)
	number4Modified := number4Section
	if len(videoIDs) > 1 {
		number4Modified = strings.ReplaceAll(number4Modified, `ref="r10"`, `ref="r11"`)
		number4Modified = strings.ReplaceAll(number4Modified, `name="ben"`, fmt.Sprintf(`name="%s"`, videoIDs[1]))
	}
	allNumberSections.WriteString(number4Modified)
	allNumberSections.WriteString("\n")
	
	// Add NUMBER 3, 2, 1 by duplicating NUMBER 5 section
	for num := 3; num >= 1; num-- {
		numStr := fmt.Sprintf("%d", num)
		newSection := strings.ReplaceAll(number5Section, "NUMBER 5", fmt.Sprintf("NUMBER %s", numStr))
		newSection = strings.ReplaceAll(newSection, ">5</text-style>", fmt.Sprintf(">%s</text-style>", numStr))
		
		// Update the unique IDs and references to avoid conflicts
		newSection = strings.ReplaceAll(newSection, "ts8", fmt.Sprintf("ts%d_num", num))
		newSection = strings.ReplaceAll(newSection, "ts9", fmt.Sprintf("ts%d_digit", num))
		
		// Update asset-clip references to use different videos
		// NUMBER 5 -> r10 (videoIDs[0])
		// NUMBER 4 -> r11 (videoIDs[1]) 
		// NUMBER 3 -> r12 (videoIDs[2])
		// NUMBER 2 -> r13 (videoIDs[3])
		// NUMBER 1 -> r14 (videoIDs[4])
		assetIndex := 5 - num // Convert number to index: 3->2, 2->3, 1->4
		if assetIndex < len(videoIDs) {
			newAssetRef := fmt.Sprintf("r1%d", assetIndex)
			newSection = strings.ReplaceAll(newSection, `ref="r10"`, fmt.Sprintf(`ref="%s"`, newAssetRef))
			
			// Update the asset-clip name if it exists in this section
			videoName := videoIDs[assetIndex]
			newSection = strings.ReplaceAll(newSection, fmt.Sprintf(`name="%s"`, videoIDs[0]), fmt.Sprintf(`name="%s"`, videoName))
		}
		
		allNumberSections.WriteString(newSection)
		allNumberSections.WriteString("\n")
	}
	
	// Replace both original sections with the new ordered sections
	result := fcpxmlContent[:number5Start] + allNumberSections.String() + fcpxmlContent[number4End:]
	
	return result
}
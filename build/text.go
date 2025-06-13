package build

import (
	"fmt"
	"cutlass/fcp"
)

// ensureTextEffect ensures the Text effect is available in resources and returns its ID
func ensureTextEffect(fcpxml *fcp.FCPXML) string {
	// Check if Text effect already exists
	for _, effect := range fcpxml.Resources.Effects {
		if effect.Name == "Text" {
			return effect.ID // Return existing ID
		}
	}
	
	// Calculate next available ID considering all resources
	totalResources := len(fcpxml.Resources.Assets) + len(fcpxml.Resources.Formats) + len(fcpxml.Resources.Effects) + len(fcpxml.Resources.Media)
	effectID := fmt.Sprintf("r%d", totalResources+1)
	
	// Add Text effect if it doesn't exist
	textEffect := fcp.Effect{
		ID:   effectID,
		Name: "Text",
		UID:  ".../Titles.localized/Basic Text.localized/Text.localized/Text.moti",
	}
	fcpxml.Resources.Effects = append(fcpxml.Resources.Effects, textEffect)
	return effectID
}

// createTextTitle creates a Title struct for text overlay
func createTextTitle(text, duration, baseName, textEffectID string) fcp.Title {
	// Generate unique text style ID using the text content and baseName
	textStyleID := generateTextStyleID(text, baseName)
	
	return fcp.Title{
		Ref:      textEffectID, // Reference to Text effect
		Lane:     "1",  // Lane 1 (above the video)
		Offset:   "0s",
		Name:     baseName + " - Text",
		Duration: duration,
		Start:    "86486400/24000s",
		Params: []fcp.Param{
			{Name: "Layout Method", Key: "9999/10003/13260/3296672360/2/314", Value: "1 (Paragraph)"},
			{Name: "Left Margin", Key: "9999/10003/13260/3296672360/2/323", Value: "-1730"},
			{Name: "Right Margin", Key: "9999/10003/13260/3296672360/2/324", Value: "1730"},
			{Name: "Top Margin", Key: "9999/10003/13260/3296672360/2/325", Value: "960"},
			{Name: "Bottom Margin", Key: "9999/10003/13260/3296672360/2/326", Value: "-960"},
			{Name: "Alignment", Key: "9999/10003/13260/3296672360/2/354/3296667315/401", Value: "1 (Center)"},
			{Name: "Line Spacing", Key: "9999/10003/13260/3296672360/2/354/3296667315/404", Value: "-19"},
			{Name: "Auto-Shrink", Key: "9999/10003/13260/3296672360/2/370", Value: "3 (To All Margins)"},
			{Name: "Alignment", Key: "9999/10003/13260/3296672360/2/373", Value: "0 (Left) 0 (Top)"},
			{Name: "Opacity", Key: "9999/10003/13260/3296672360/4/3296673134/1000/1044", Value: "0"},
			{Name: "Speed", Key: "9999/10003/13260/3296672360/4/3296673134/201/208", Value: "6 (Custom)"},
			{
				Name: "Custom Speed", 
				Key: "9999/10003/13260/3296672360/4/3296673134/201/209",
				KeyframeAnimation: &fcp.KeyframeAnimation{
					Keyframes: []fcp.Keyframe{
						{Time: "-469658744/1000000000s", Value: "0"},
						{Time: "12328542033/1000000000s", Value: "1"},
					},
				},
			},
			{Name: "Apply Speed", Key: "9999/10003/13260/3296672360/4/3296673134/201/211", Value: "2 (Per Object)"},
		},
		Text: &fcp.TitleText{
			TextStyle: fcp.TextStyleRef{
				Ref:  textStyleID,
				Text: text,
			},
		},
		TextStyleDef: &fcp.TextStyleDef{
			ID: textStyleID,
			TextStyle: fcp.TextStyle{
				Font:        "Helvetica Neue",
				FontSize:    "196",
				FontColor:   "1 1 1 1",
				Bold:        "1",
				Alignment:   "center",
				LineSpacing: "-19",
			},
		},
	}
}

// generateTextStyleID creates a unique text style ID based on content and baseName
// CRITICAL: This ensures text-style-def IDs are unique across the entire FCPXML document.
// Never hardcode text style IDs like "ts1" as this causes DTD validation failures
// when multiple text overlays are added to the same project.
func generateTextStyleID(text, baseName string) string {
	// Use the existing generateUID function to create a hash-based ID
	fullText := "text_" + baseName + "_" + text
	uid := generateUID(fullText)
	// Return a shorter, more readable ID using the first 8 characters
	return "ts" + uid[0:8]
}
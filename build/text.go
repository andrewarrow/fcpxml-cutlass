package build

import (
	"cutlass/fcp"
)

// ensureTextEffect ensures the Text effect is available in resources
func ensureTextEffect(fcpxml *fcp.FCPXML) {
	// Check if Text effect already exists
	for _, effect := range fcpxml.Resources.Effects {
		if effect.Name == "Text" {
			return // Already exists
		}
	}
	
	// Add Text effect if it doesn't exist
	textEffect := fcp.Effect{
		ID:   "r6",
		Name: "Text",
		UID:  ".../Titles.localized/Basic Text.localized/Text.localized/Text.moti",
	}
	fcpxml.Resources.Effects = append(fcpxml.Resources.Effects, textEffect)
}

// createTextTitle creates a Title struct for text overlay
func createTextTitle(text, duration, baseName string) fcp.Title {
	return fcp.Title{
		Ref:      "r6", // Reference to Text effect
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
				Ref:  "ts1",
				Text: text,
			},
		},
		TextStyleDef: &fcp.TextStyleDef{
			ID: "ts1",
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
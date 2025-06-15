package fcp

import (
	"encoding/xml"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

func FormatDurationForFCPXML(d time.Duration) string {
	// Convert to frame-aligned format for 30fps video
	// 30000 frames per second with 1001/30000s frame duration
	totalFrames := int64(d.Seconds() * 30000 / 1001)
	// Ensure frame alignment
	return fmt.Sprintf("%d/30000s", totalFrames*1001)
}

// FormatRationalTime converts FCPXML rational time format (e.g. "8300/3000s") to readable seconds (e.g. "2.77s")
func FormatRationalTime(rationalTime string) string {
	if rationalTime == "" {
		return ""
	}
	
	// Remove trailing 's' if present
	time := strings.TrimSuffix(rationalTime, "s")
	
	// Check if it contains a fraction
	if strings.Contains(time, "/") {
		parts := strings.Split(time, "/")
		if len(parts) == 2 {
			numerator, err1 := strconv.ParseFloat(parts[0], 64)
			denominator, err2 := strconv.ParseFloat(parts[1], 64)
			if err1 == nil && err2 == nil && denominator != 0 {
				result := numerator / denominator
				return fmt.Sprintf("%.2fs", result)
			}
		}
	}
	
	// If not a fraction, try to parse as a simple number
	if value, err := strconv.ParseFloat(time, 64); err == nil {
		return fmt.Sprintf("%.2fs", value)
	}
	
	// Return original if parsing fails
	return rationalTime
}

func GenerateStandard(inputFile, outputFile string) error {
	inputName := filepath.Base(inputFile)
	inputExt := strings.ToLower(filepath.Ext(inputFile))
	nameWithoutExt := strings.TrimSuffix(inputName, inputExt)

	fcpxml := FCPXML{
		Version: "1.11",
		Resources: Resources{
			Formats: []Format{
				{
					ID:            "r1",
					Name:          "FFVideoFormat1080p30",
					FrameDuration: "1001/30000s",
					Width:         "1920",
					Height:        "1080",
					ColorSpace:    "1-1-1 (Rec. 709)",
				},
			},
			Assets: []Asset{
				{
					ID:           "r2",
					Name:         nameWithoutExt,
					UID:          inputFile,
					Start:        "0s",
					HasVideo:     "1",
					Format:       "r1",
					HasAudio:     "1",
					AudioSources: "1",
					AudioChannels: "2",
					Duration:     "3600s",
					MediaRep: MediaRep{
						Kind: "original-media",
						Sig:  inputFile,
						Src:  "file://" + inputFile,
					},
				},
			},
		},
		Library: Library{
			Events: []Event{
				{
					Name: "Converted Media",
					Projects: []Project{
						{
							Name: nameWithoutExt,
							Sequences: []Sequence{
								{
									Format:      "r1",
									Duration:    "3600s",
									TCStart:     "0s",
									TCFormat:    "NDF",
									AudioLayout: "stereo",
									AudioRate:   "48k",
									Spine: Spine{
										Content: "",  // Will be populated below using structs
									},
								},
							},
						},
					},
				},
			},
		},
	}

	// Create the asset-clip using structs
	assetClip := AssetClip{
		Ref:       "r2",
		Offset:    "0s",
		Name:      nameWithoutExt,
		Duration:  "3600s",
		TCFormat:  "NDF",
		AudioRole: "dialogue",
	}
	
	// Marshal the asset-clip to XML and set it as spine content
	spineXML, err := xml.MarshalIndent(assetClip, "                        ", "    ")
	if err != nil {
		return err
	}
	
	// Set the spine content with proper indentation
	fcpxml.Library.Events[0].Projects[0].Sequences[0].Spine.Content = "\n                        " + string(spineXML) + "\n                    "

	output, err := xml.MarshalIndent(fcpxml, "", "    ")
	if err != nil {
		return err
	}

	xmlContent := xml.Header + "<!DOCTYPE fcpxml>\n" + string(output)
	return os.WriteFile(outputFile, []byte(xmlContent), 0644)
}

func escapeXMLText(text string) string {
	text = strings.ReplaceAll(text, "&", "&amp;")
	text = strings.ReplaceAll(text, "<", "&lt;")
	text = strings.ReplaceAll(text, ">", "&gt;")
	text = strings.ReplaceAll(text, "\"", "&quot;")
	text = strings.ReplaceAll(text, "'", "&#39;")
	return text
}

func ParseFCPXML(filePath string) (*FCPXML, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %v", err)
	}

	var fcpxml FCPXML
	err = xml.Unmarshal(data, &fcpxml)
	if err != nil {
		return nil, fmt.Errorf("failed to parse XML: %v", err)
	}

	return &fcpxml, nil
}

func DisplayFCPXML(fcpxml *FCPXML) {
	fmt.Printf("=== FCPXML File Analysis ===\n")
	fmt.Printf("Version: %s\n\n", fcpxml.Version)

	fmt.Printf("=== Resources ===\n")
	fmt.Printf("Formats: %d\n", len(fcpxml.Resources.Formats))
	for i, format := range fcpxml.Resources.Formats {
		fmt.Printf("  Format %d: %s (%s)\n", i+1, format.Name, format.ID)
		fmt.Printf("    Resolution: %sx%s\n", format.Width, format.Height)
		fmt.Printf("    Frame Duration: %s\n", format.FrameDuration)
		fmt.Printf("    Color Space: %s\n", format.ColorSpace)
	}
	fmt.Printf("\n")

	fmt.Printf("Assets: %d\n", len(fcpxml.Resources.Assets))
	for i, asset := range fcpxml.Resources.Assets {
		fmt.Printf("  Asset %d: %s (%s)\n", i+1, asset.Name, asset.ID)
		fmt.Printf("    Duration: %s\n", asset.Duration)
		fmt.Printf("    Video: %s, Audio: %s\n", asset.HasVideo, asset.HasAudio)
		if asset.HasAudio == "1" {
			fmt.Printf("    Audio Channels: %s\n", asset.AudioChannels)
		}
		fmt.Printf("    Source: %s\n", asset.MediaRep.Src)
	}
	fmt.Printf("\n")

	fmt.Printf("Effects: %d\n", len(fcpxml.Resources.Effects))
	for i, effect := range fcpxml.Resources.Effects {
		fmt.Printf("  Effect %d: %s (%s)\n", i+1, effect.Name, effect.ID)
	}
	fmt.Printf("\n")

	fmt.Printf("=== Library Structure ===\n")
	fmt.Printf("Events: %d\n", len(fcpxml.Library.Events))
	for i, event := range fcpxml.Library.Events {
		fmt.Printf("  Event %d: %s\n", i+1, event.Name)
		fmt.Printf("    Projects: %d\n", len(event.Projects))
		for j, project := range event.Projects {
			fmt.Printf("      Project %d: %s\n", j+1, project.Name)
			fmt.Printf("        Sequences: %d\n", len(project.Sequences))
			for k, sequence := range project.Sequences {
				fmt.Printf("          Sequence %d:\n", k+1)
				fmt.Printf("            Duration: %s\n", sequence.Duration)
				fmt.Printf("            Format: %s\n", sequence.Format)
				fmt.Printf("            Timecode Start: %s\n", sequence.TCStart)
				fmt.Printf("            Audio Layout: %s\n", sequence.AudioLayout)
				fmt.Printf("            Audio Rate: %s\n", sequence.AudioRate)
				
				spineContent := strings.TrimSpace(sequence.Spine.Content)
				if spineContent != "" {
					fmt.Printf("            Timeline Elements:\n")
					parseSpineContent(spineContent, "              ")
				}
			}
		}
	}
}

func parseSpineContent(content, indent string) {
	// Wrap content in a root element to make it valid XML
	wrappedContent := "<spine>" + content + "</spine>"
	
	var spineData struct {
		Videos     []Video     `xml:"video"`
		Titles     []Title     `xml:"title"`
		AssetClips []AssetClip `xml:"asset-clip"`
		Gaps       []Gap       `xml:"gap"`
	}
	
	err := xml.Unmarshal([]byte(wrappedContent), &spineData)
	if err != nil {
		fmt.Printf("%sError parsing spine content: %v\n", indent, err)
		return
	}
	
	// Display asset clips (main video/audio clips)
	for i, clip := range spineData.AssetClips {
		fmt.Printf("%sAsset Clip %d: %s\n", indent, i+1, clip.Name)
		fmt.Printf("%s  Reference: %s\n", indent, clip.Ref)
		fmt.Printf("%s  Offset: %s\n", indent, clip.Offset)
		fmt.Printf("%s  Duration: %s\n", indent, clip.Duration)
		if clip.Start != "" {
			fmt.Printf("%s  Start: %s\n", indent, clip.Start)
		}
	}
	
	// Display video elements (shapes, generators, etc.)
	for i, video := range spineData.Videos {
		displayVideoElement(video, i+1, indent, 0)
	}
	
	// Display title elements
	for i, title := range spineData.Titles {
		fmt.Printf("%sTitle %d: %s\n", indent, i+1, title.Name)
		fmt.Printf("%s  Reference: %s\n", indent, title.Ref)
		fmt.Printf("%s  Offset: %s\n", indent, title.Offset)
		fmt.Printf("%s  Duration: %s\n", indent, title.Duration)
		if title.Lane != "" {
			fmt.Printf("%s  Lane: %s\n", indent, title.Lane)
		}
	}
	
	// Display gaps
	for i, gap := range spineData.Gaps {
		fmt.Printf("%sGap %d: %s\n", indent, i+1, gap.Name)
		fmt.Printf("%s  Offset: %s\n", indent, gap.Offset)
		fmt.Printf("%s  Duration: %s\n", indent, gap.Duration)
	}
}

func displayVideoElement(video Video, index int, baseIndent string, level int) {
	indent := baseIndent + strings.Repeat("  ", level)
	
	if level == 0 {
		fmt.Printf("%sVideo Element %d: %s\n", indent, index, video.Name)
	} else {
		fmt.Printf("%sNested Video (Lane %s): %s\n", indent, video.Lane, video.Name)
	}
	
	fmt.Printf("%s  Reference: %s\n", indent, video.Ref)
	fmt.Printf("%s  Offset: %s\n", indent, video.Offset)
	fmt.Printf("%s  Duration: %s\n", indent, video.Duration)
	if video.Lane != "" && level == 0 {
		fmt.Printf("%s  Lane: %s\n", indent, video.Lane)
	}
	if video.Start != "" {
		fmt.Printf("%s  Start: %s\n", indent, video.Start)
	}
	
	// Show key parameters
	keyParams := []string{"Shape", "Fill Color", "Center", "Outline"}
	for _, param := range video.Params {
		for _, key := range keyParams {
			if strings.Contains(param.Name, key) {
				fmt.Printf("%s  %s: %s\n", indent, param.Name, param.Value)
				break
			}
		}
	}
	
	// Show transform info
	if video.AdjustTransform != nil {
		if video.AdjustTransform.Position != "" {
			fmt.Printf("%s  Position: %s\n", indent, video.AdjustTransform.Position)
		}
		if video.AdjustTransform.Scale != "" {
			fmt.Printf("%s  Scale: %s\n", indent, video.AdjustTransform.Scale)
		}
	}
	
	// Display nested elements
	for i, nestedVideo := range video.NestedVideos {
		displayVideoElement(nestedVideo, i+1, baseIndent, level+1)
	}
	
	for _, nestedTitle := range video.NestedTitles {
		fmt.Printf("%sNested Title (Lane %s): %s\n", indent+"  ", nestedTitle.Lane, nestedTitle.Name)
		fmt.Printf("%s  Reference: %s\n", indent+"  ", nestedTitle.Ref)
		fmt.Printf("%s  Offset: %s\n", indent+"  ", nestedTitle.Offset)
		fmt.Printf("%s  Duration: %s\n", indent+"  ", nestedTitle.Duration)
	}
}

func DisplayFCPXMLWithOptions(fcpxml *FCPXML, options ParseOptions) {
	fmt.Printf("=== FCPXML Analysis: %s ===\n", getTierDescription(options.Tier))
	fmt.Printf("Version: %s\n\n", fcpxml.Version)

	// Always show basic structure for tier 1+
	displayTier1Structure(fcpxml, options)

	// Show story elements for tier 2+
	if options.Tier >= 2 || options.ShowElements {
		displayTier2Elements(fcpxml, options)
	}

	// Show detailed parameters and animations for tier 3+
	if options.Tier >= 3 || options.ShowParams || options.ShowAnimation {
		displayTier3Details(fcpxml, options)
	}
}

func getTierDescription(tier int) string {
	switch tier {
	case 1:
		return "Tier 1 - Foundation Elements (Core DNA)"
	case 2:
		return "Tier 2 - Foundation + Story Elements"
	case 3:
		return "Tier 3 - Complete Technical Analysis"
	default:
		return "All Elements"
	}
}

func displayTier1Structure(fcpxml *FCPXML, options ParseOptions) {
	fmt.Printf("🎬 === TIER 1: THE FOUNDATION ===\n")
	fmt.Printf("The DNA present in every video project:\n\n")

	// fcpxml root element
	fmt.Printf("📄 FCPXML Root Element\n")
	fmt.Printf("   Project's birth certificate: version %s\n", fcpxml.Version)
	fmt.Printf("   Contains: resources + library structure\n\n")

	// Resources overview
	fmt.Printf("📦 Resources Section (Digital Warehouse)\n")
	totalAssets := len(fcpxml.Resources.Assets)
	totalFormats := len(fcpxml.Resources.Formats)
	totalEffects := len(fcpxml.Resources.Effects)
	
	fmt.Printf("   📁 %d Video/Audio Assets\n", totalAssets)
	fmt.Printf("   🎞️  %d Format Specifications\n", totalFormats)
	fmt.Printf("   ✨ %d Effects/Generators\n", totalEffects)

	if options.ShowResources || options.Tier >= 2 {
		displayResourceDetails(fcpxml, options)
	}

	// Library structure
	fmt.Printf("\n🎭 Sequence Element (Movie Timeline)\n")
	eventCount := len(fcpxml.Library.Events)
	fmt.Printf("   📂 %d Event(s) containing projects\n", eventCount)
	
	for i, event := range fcpxml.Library.Events {
		fmt.Printf("      Event %d: %s\n", i+1, event.Name)
		for j, project := range event.Projects {
			fmt.Printf("         📁 Project %d: %s\n", j+1, project.Name)
			for k, sequence := range project.Sequences {
				fmt.Printf("            🎬 Sequence %d: %s (%s) duration, %s layout\n", 
					k+1, sequence.Duration, FormatRationalTime(sequence.Duration), sequence.AudioLayout)
				if options.ShowStructure {
					fmt.Printf("               Frame Rate: %s\n", sequence.Format)
					fmt.Printf("               Timecode: %s (%s)\n", sequence.TCStart, sequence.TCFormat)
					fmt.Printf("               Audio Rate: %s\n", sequence.AudioRate)
				}
			}
		}
	}
	fmt.Printf("\n")
}

func displayResourceDetails(fcpxml *FCPXML, options ParseOptions) {
	if len(fcpxml.Resources.Formats) > 0 {
		fmt.Printf("\n   🎞️  Format Specifications:\n")
		for i, format := range fcpxml.Resources.Formats {
			fmt.Printf("      Format %d (%s): %sx%s @ %s\n", 
				i+1, format.ID, format.Width, format.Height, format.FrameDuration)
			if options.ShowStructure {
				fmt.Printf("         Name: %s\n", format.Name)
				fmt.Printf("         Color Space: %s\n", format.ColorSpace)
			}
		}
	}

	if len(fcpxml.Resources.Assets) > 0 {
		fmt.Printf("\n   📁 Media Assets:\n")
		for i, asset := range fcpxml.Resources.Assets {
			fmt.Printf("      Asset %d (%s): %s\n", i+1, asset.ID, asset.Name)
			fmt.Printf("         Duration: %s (%s)\n", asset.Duration, FormatRationalTime(asset.Duration))
			
			mediaType := []string{}
			if asset.HasVideo == "1" {
				mediaType = append(mediaType, "Video")
			}
			if asset.HasAudio == "1" {
				mediaType = append(mediaType, fmt.Sprintf("Audio (%s ch)", asset.AudioChannels))
			}
			fmt.Printf("         Type: %s\n", strings.Join(mediaType, " + "))
			
			if options.ShowStructure {
				fmt.Printf("         Source: %s\n", asset.MediaRep.Src)
				fmt.Printf("         UID: %s\n", asset.UID)
			}
		}
	}

	if len(fcpxml.Resources.Effects) > 0 {
		fmt.Printf("\n   ✨ Effects & Generators:\n")
		for i, effect := range fcpxml.Resources.Effects {
			fmt.Printf("      Effect %d (%s): %s\n", i+1, effect.ID, effect.Name)
			if options.ShowStructure && effect.UID != "" {
				fmt.Printf("         UID: %s\n", effect.UID)
			}
		}
	}
}

func displayTier2Elements(fcpxml *FCPXML, options ParseOptions) {
	fmt.Printf("🎪 === TIER 2: STORY ELEMENTS ===\n")
	fmt.Printf("Moving beyond basic editing to engaging content:\n\n")

	for _, event := range fcpxml.Library.Events {
		for _, project := range event.Projects {
			for _, sequence := range project.Sequences {
				spineContent := strings.TrimSpace(sequence.Spine.Content)
				if spineContent != "" {
					fmt.Printf("📽️  Timeline Elements in '%s':\n", project.Name)
					parseSpineContentTiered(spineContent, "   ", options)
				}
			}
		}
	}
	fmt.Printf("\n")
}

func displayTier3Details(fcpxml *FCPXML, options ParseOptions) {
	fmt.Printf("🪄 === TIER 3: TECHNICAL MAGIC ===\n")
	fmt.Printf("Where complexity creates cinematic beauty:\n\n")
	
	// This would show detailed parameter analysis, keyframe data, etc.
	// For now, we'll indicate what would be shown here
	fmt.Printf("🔧 Parameter Hierarchies:\n")
	fmt.Printf("   Nested layers of control for precise positioning and animation\n\n")
	
	fmt.Printf("⚡ Keyframe Animations:\n")
	fmt.Printf("   Frame-perfect timing using rational numbers (e.g., 1001/30000s)\n\n")
	
	fmt.Printf("🎯 Lane Systems:\n")
	fmt.Printf("   Vertical stacking: Lane 0 (main), Lane 1+ (above), Lane -1- (below)\n\n")
	
	if options.ShowAnimation {
		fmt.Printf("🎭 Animation Details:\n")
		displayAnimationDetails(fcpxml, "   ")
	}
	
	if options.ShowParams {
		fmt.Printf("⚙️  Parameter Details:\n")
		fmt.Printf("   [Complete parameter hierarchy would appear here]\n\n")
	}
}

func parseSpineContentTiered(content, indent string, options ParseOptions) {
	// Wrap content in a root element to make it valid XML
	wrappedContent := "<spine>" + content + "</spine>"
	
	var spineData struct {
		Videos     []Video     `xml:"video"`
		Titles     []Title     `xml:"title"`
		AssetClips []AssetClip `xml:"asset-clip"`
		Gaps       []Gap       `xml:"gap"`
	}
	
	err := xml.Unmarshal([]byte(wrappedContent), &spineData)
	if err != nil {
		fmt.Printf("%sError parsing spine content: %v\n", indent, err)
		return
	}
	
	// Display asset clips (main video/audio clips)
	for i, clip := range spineData.AssetClips {
		fmt.Printf("%s🎬 Asset Clip %d: %s\n", indent, i+1, clip.Name)
		fmt.Printf("%s   📎 Reference: %s\n", indent, clip.Ref)
		fmt.Printf("%s   ⏰ Timeline: offset %s (%s), duration %s (%s)\n", indent, clip.Offset, FormatRationalTime(clip.Offset), clip.Duration, FormatRationalTime(clip.Duration))
		if clip.Start != "" {
			fmt.Printf("%s   🎯 Source start: %s\n", indent, clip.Start)
		}
		if clip.AudioRole != "" {
			fmt.Printf("%s   🔊 Audio role: %s\n", indent, clip.AudioRole)
		}
	}
	
	// Display video elements (shapes, generators, etc.)
	for i, video := range spineData.Videos {
		displayVideoElementTiered(video, i+1, indent, 0, options)
	}
	
	// Display title elements
	for i, title := range spineData.Titles {
		fmt.Printf("%s✍️  Title %d: %s\n", indent, i+1, title.Name)
		fmt.Printf("%s   📎 Reference: %s\n", indent, title.Ref)
		fmt.Printf("%s   ⏰ Timeline: offset %s (%s), duration %s (%s)\n", indent, title.Offset, FormatRationalTime(title.Offset), title.Duration, FormatRationalTime(title.Duration))
		if title.Lane != "" {
			fmt.Printf("%s   🎚️  Lane: %s\n", indent, title.Lane)
		}
		if title.Start != "" {
			fmt.Printf("%s   🎯 Source start: %s\n", indent, title.Start)
		}
		
		if options.ShowParams && len(title.Params) > 0 {
			fmt.Printf("%s   ⚙️  Parameters: %d total\n", indent, len(title.Params))
			for j, param := range title.Params {
				if j < 3 { // Show first 3 params to avoid overwhelming
					fmt.Printf("%s      • %s: %s\n", indent, param.Name, param.Value)
				} else if j == 3 {
					fmt.Printf("%s      • ... and %d more\n", indent, len(title.Params)-3)
					break
				}
			}
		}
	}
	
	// Display gaps
	for i, gap := range spineData.Gaps {
		fmt.Printf("%s⏸️  Gap %d: %s\n", indent, i+1, gap.Name)
		fmt.Printf("%s   ⏰ Timeline: offset %s (%s), duration %s (%s)\n", indent, gap.Offset, FormatRationalTime(gap.Offset), gap.Duration, FormatRationalTime(gap.Duration))
	}
}

func displayVideoElementTiered(video Video, index int, baseIndent string, level int, options ParseOptions) {
	indent := baseIndent + strings.Repeat("  ", level)
	
	if level == 0 {
		fmt.Printf("%s🎨 Video Element %d: %s\n", indent, index, video.Name)
	} else {
		fmt.Printf("%s  🎨 Nested Video (Lane %s): %s\n", indent, video.Lane, video.Name)
	}
	
	fmt.Printf("%s   📎 Reference: %s\n", indent, video.Ref)
	fmt.Printf("%s   ⏰ Timeline: offset %s (%s), duration %s (%s)\n", indent, video.Offset, FormatRationalTime(video.Offset), video.Duration, FormatRationalTime(video.Duration))
	if video.Lane != "" && level == 0 {
		fmt.Printf("%s   🎚️  Lane: %s\n", indent, video.Lane)
	}
	if video.Start != "" {
		fmt.Printf("%s   🎯 Source start: %s\n", indent, video.Start)
	}
	
	// Show transform info (tier 2+)
	if options.Tier >= 2 || options.ShowParams {
		if video.AdjustTransform != nil {
			if video.AdjustTransform.Position != "" {
				fmt.Printf("%s   📍 Position: %s\n", indent, video.AdjustTransform.Position)
			}
			if video.AdjustTransform.Scale != "" {
				fmt.Printf("%s   📏 Scale: %s\n", indent, video.AdjustTransform.Scale)
			}
		}
	}
	
	// Show detailed parameters (tier 3+)
	if (options.Tier >= 3 || options.ShowParams) && len(video.Params) > 0 {
		fmt.Printf("%s   ⚙️  Parameters: %d total\n", indent, len(video.Params))
		for j, param := range video.Params {
			if j < 3 { // Show first 3 params to avoid overwhelming
				fmt.Printf("%s      • %s: %s\n", indent, param.Name, param.Value)
				if options.ShowAnimation && param.KeyframeAnimation != nil {
					fmt.Printf("%s        🎭 Animated (%d keyframes)\n", indent, len(param.KeyframeAnimation.Keyframes))
				}
			} else if j == 3 {
				fmt.Printf("%s      • ... and %d more\n", indent, len(video.Params)-3)
				break
			}
		}
	}
	
	// Display nested elements
	for i, nestedVideo := range video.NestedVideos {
		displayVideoElementTiered(nestedVideo, i+1, baseIndent, level+1, options)
	}
	
	for _, nestedTitle := range video.NestedTitles {
		fmt.Printf("%s  ✍️  Nested Title (Lane %s): %s\n", indent, nestedTitle.Lane, nestedTitle.Name)
		fmt.Printf("%s     📎 Reference: %s\n", indent, nestedTitle.Ref)
		fmt.Printf("%s     ⏰ Timeline: offset %s (%s), duration %s (%s)\n", indent, nestedTitle.Offset, FormatRationalTime(nestedTitle.Offset), nestedTitle.Duration, FormatRationalTime(nestedTitle.Duration))
	}
}

// displayAnimationDetails analyzes and displays keyframe animations throughout the project
func displayAnimationDetails(fcpxml *FCPXML, indent string) {
	animationCount := 0
	
	// Scan through all events, projects, and sequences
	for _, event := range fcpxml.Library.Events {
		for _, project := range event.Projects {
			for _, sequence := range project.Sequences {
				spineContent := strings.TrimSpace(sequence.Spine.Content)
				if spineContent != "" {
					animationCount += analyzeSpineAnimations(spineContent, indent, project.Name)
				}
			}
		}
	}
	
	if animationCount == 0 {
		fmt.Printf("%sNo keyframe animations found in this project\n\n", indent)
	} else {
		fmt.Printf("%sTotal animated parameters: %d\n\n", indent, animationCount)
	}
}

// analyzeSpineAnimations parses spine content and analyzes animations
func analyzeSpineAnimations(content, indent, projectName string) int {
	// Wrap content in a root element to make it valid XML
	wrappedContent := "<spine>" + content + "</spine>"
	
	var spineData struct {
		Videos     []Video     `xml:"video"`
		Titles     []Title     `xml:"title"`
		AssetClips []AssetClip `xml:"asset-clip"`
		Gaps       []Gap       `xml:"gap"`
	}
	
	err := xml.Unmarshal([]byte(wrappedContent), &spineData)
	if err != nil {
		fmt.Printf("%sError parsing spine content for animations: %v\n", indent, err)
		return 0
	}
	
	animationCount := 0
	
	// Analyze asset clips (main video/audio clips with potential transforms and titles)
	for i, clip := range spineData.AssetClips {
		count := analyzeAssetClipAnimations(clip, i+1, indent, projectName)
		animationCount += count
	}
	
	// Analyze video elements
	for i, video := range spineData.Videos {
		count := analyzeVideoAnimations(video, i+1, indent, projectName, 0)
		animationCount += count
	}
	
	// Analyze title elements
	for i, title := range spineData.Titles {
		count := analyzeTitleAnimations(title, i+1, indent, projectName)
		animationCount += count
	}
	
	// Analyze gaps (which can contain generator clips and titles)
	for _, gap := range spineData.Gaps {
		for i, title := range gap.Titles {
			count := analyzeTitleAnimations(title, i+1, indent, projectName)
			animationCount += count
		}
	}
	
	return animationCount
}

// analyzeVideoAnimations analyzes animations in video elements
func analyzeVideoAnimations(video Video, index int, baseIndent, projectName string, level int) int {
	indent := baseIndent + strings.Repeat("  ", level)
	animationCount := 0
	
	// Check parameters for animations
	for _, param := range video.Params {
		if param.KeyframeAnimation != nil && len(param.KeyframeAnimation.Keyframes) > 0 {
			if animationCount == 0 {
				if level == 0 {
					fmt.Printf("%s🎨 Video Element %d (\"%s\") in project \"%s\":\n", indent, index, video.Name, projectName)
				} else {
					fmt.Printf("%s🎨 Nested Video (\"%s\", Lane %s):\n", indent, video.Name, video.Lane)
				}
			}
			
			fmt.Printf("%s   🎭 Parameter \"%s\" animated:\n", indent, param.Name)
			fmt.Printf("%s      Keyframes: %d total\n", indent, len(param.KeyframeAnimation.Keyframes))
			
			// Show first few keyframes for detail
			maxShow := 3
			if len(param.KeyframeAnimation.Keyframes) < maxShow {
				maxShow = len(param.KeyframeAnimation.Keyframes)
			}
			
			for i := 0; i < maxShow; i++ {
				kf := param.KeyframeAnimation.Keyframes[i]
				fmt.Printf("%s      • Frame %d: %s (%s) → %s\n", indent, i+1, kf.Time, FormatRationalTime(kf.Time), kf.Value)
			}
			
			if len(param.KeyframeAnimation.Keyframes) > maxShow {
				fmt.Printf("%s      • ... and %d more frames\n", indent, len(param.KeyframeAnimation.Keyframes)-maxShow)
			}
			
			animationCount++
		}
		
		// Recursively check nested parameters
		headerPrinted := (animationCount > 0)
		nestedCount := analyzeNestedParameterAnimations(param.NestedParams, indent, video.Name, projectName, "Video", &headerPrinted)
		animationCount += nestedCount
	}
	
	// Analyze nested elements
	for i, nestedVideo := range video.NestedVideos {
		count := analyzeVideoAnimations(nestedVideo, i+1, baseIndent, projectName, level+1)
		animationCount += count
	}
	
	for _, nestedTitle := range video.NestedTitles {
		count := analyzeTitleAnimations(nestedTitle, 1, baseIndent, projectName)
		animationCount += count
	}
	
	return animationCount
}

// analyzeTitleAnimations analyzes animations in title elements
func analyzeTitleAnimations(title Title, index int, indent, projectName string) int {
	animationCount := 0
	headerPrinted := false
	
	// Check parameters for animations
	for _, param := range title.Params {
		if param.KeyframeAnimation != nil && len(param.KeyframeAnimation.Keyframes) > 0 {
			if !headerPrinted {
				fmt.Printf("%s✍️  Title Element %d (\"%s\") in project \"%s\":\n", indent, index, title.Name, projectName)
				if title.Lane != "" {
					fmt.Printf("%s   Lane: %s\n", indent, title.Lane)
				}
				headerPrinted = true
			}
			
			fmt.Printf("%s   🎭 Parameter \"%s\" animated:\n", indent, param.Name)
			fmt.Printf("%s      Keyframes: %d total\n", indent, len(param.KeyframeAnimation.Keyframes))
			
			// Show first few keyframes for detail
			maxShow := 3
			if len(param.KeyframeAnimation.Keyframes) < maxShow {
				maxShow = len(param.KeyframeAnimation.Keyframes)
			}
			
			for i := 0; i < maxShow; i++ {
				kf := param.KeyframeAnimation.Keyframes[i]
				fmt.Printf("%s      • Frame %d: %s (%s) → %s\n", indent, i+1, kf.Time, FormatRationalTime(kf.Time), kf.Value)
			}
			
			if len(param.KeyframeAnimation.Keyframes) > maxShow {
				fmt.Printf("%s      • ... and %d more frames\n", indent, len(param.KeyframeAnimation.Keyframes)-maxShow)
			}
			
			animationCount++
		}
		
		// Recursively check nested parameters
		nestedCount := analyzeNestedParameterAnimations(param.NestedParams, indent, title.Name, projectName, "Title", &headerPrinted)
		animationCount += nestedCount
		if nestedCount > 0 && !headerPrinted {
			fmt.Printf("%s✍️  Title Element %d (\"%s\") in project \"%s\":\n", indent, index, title.Name, projectName)
			if title.Lane != "" {
				fmt.Printf("%s   Lane: %s\n", indent, title.Lane)
			}
			headerPrinted = true
		}
	}
	
	return animationCount
}

// analyzeNestedParameterAnimations recursively analyzes nested parameters for animations
func analyzeNestedParameterAnimations(params []Param, indent, elementName, projectName, elementType string, headerPrinted *bool) int {
	animationCount := 0
	
	for _, param := range params {
		if param.KeyframeAnimation != nil && len(param.KeyframeAnimation.Keyframes) > 0 {
			if !*headerPrinted {
				fmt.Printf("%s🎯 %s Element (\"%s\") in project \"%s\":\n", indent, elementType, elementName, projectName)
				*headerPrinted = true
			}
			
			// Show the parameter path
			paramPath := param.Name
			if param.Key != "" {
				paramPath += fmt.Sprintf(" (key: %s)", param.Key)
			}
			
			fmt.Printf("%s   🎭 Parameter \"%s\" animated:\n", indent, paramPath)
			fmt.Printf("%s      Keyframes: %d total\n", indent, len(param.KeyframeAnimation.Keyframes))
			
			// Show first few keyframes for detail
			maxShow := 3
			if len(param.KeyframeAnimation.Keyframes) < maxShow {
				maxShow = len(param.KeyframeAnimation.Keyframes)
			}
			
			for i := 0; i < maxShow; i++ {
				kf := param.KeyframeAnimation.Keyframes[i]
				fmt.Printf("%s      • Frame %d: %s (%s) → %s\n", indent, i+1, kf.Time, FormatRationalTime(kf.Time), kf.Value)
			}
			
			if len(param.KeyframeAnimation.Keyframes) > maxShow {
				fmt.Printf("%s      • ... and %d more frames\n", indent, len(param.KeyframeAnimation.Keyframes)-maxShow)
			}
			
			animationCount++
		}
		
		// Recursively check further nested parameters
		animationCount += analyzeNestedParameterAnimations(param.NestedParams, indent, elementName, projectName, elementType, headerPrinted)
	}
	
	return animationCount
}

// analyzeAssetClipAnimations analyzes animations in asset clip elements
func analyzeAssetClipAnimations(clip AssetClip, index int, indent, projectName string) int {
	animationCount := 0
	headerPrinted := false
	
	// Check adjust-transform for animations
	if clip.AdjustTransform != nil {
		for _, param := range clip.AdjustTransform.Params {
			if param.KeyframeAnimation != nil && len(param.KeyframeAnimation.Keyframes) > 0 {
				if !headerPrinted {
					fmt.Printf("%s🎬 Asset Clip %d (\"%s\") in project \"%s\":\n", indent, index, clip.Name, projectName)
					headerPrinted = true
				}
				
				fmt.Printf("%s   🎭 Transform Parameter \"%s\" animated:\n", indent, param.Name)
				fmt.Printf("%s      Keyframes: %d total\n", indent, len(param.KeyframeAnimation.Keyframes))
				
				// Show first few keyframes for detail
				maxShow := 3
				if len(param.KeyframeAnimation.Keyframes) < maxShow {
					maxShow = len(param.KeyframeAnimation.Keyframes)
				}
				
				for i := 0; i < maxShow; i++ {
					kf := param.KeyframeAnimation.Keyframes[i]
					fmt.Printf("%s      • Frame %d: %s (%s) → %s\n", indent, i+1, kf.Time, FormatRationalTime(kf.Time), kf.Value)
				}
				
				if len(param.KeyframeAnimation.Keyframes) > maxShow {
					fmt.Printf("%s      • ... and %d more frames\n", indent, len(param.KeyframeAnimation.Keyframes)-maxShow)
				}
				
				animationCount++
			}
			
			// Recursively check nested parameters in transform
			nestedCount := analyzeNestedParameterAnimations(param.NestedParams, indent, clip.Name, projectName, "Asset Clip", &headerPrinted)
			animationCount += nestedCount
		}
	}
	
	// Check nested titles for animations
	for i, title := range clip.Titles {
		count := analyzeTitleAnimations(title, i+1, indent, projectName)
		animationCount += count
	}
	
	// Check nested videos for animations
	for i, video := range clip.Videos {
		count := analyzeVideoAnimations(video, i+1, indent, projectName, 0)
		animationCount += count
	}
	
	return animationCount
}
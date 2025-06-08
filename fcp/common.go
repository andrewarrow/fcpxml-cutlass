package fcp

import (
	"encoding/xml"
	"fmt"
	"os"
	"path/filepath"
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
										Content: `<asset-clip ref="r2" offset="0s" name="` + nameWithoutExt + `" duration="3600s" tcFormat="NDF" audioRole="dialogue"/>`,
									},
								},
							},
						},
					},
				},
			},
		},
	}

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
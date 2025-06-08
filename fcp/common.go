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
					fmt.Printf("            Spine Content:\n")
					lines := strings.Split(spineContent, "\n")
					for _, line := range lines {
						if strings.TrimSpace(line) != "" {
							fmt.Printf("              %s\n", strings.TrimSpace(line))
						}
					}
				}
			}
		}
	}
}
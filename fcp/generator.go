package fcp

import (
	"encoding/xml"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"cutalyst/vtt"
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

func BuildClipFCPXML(clips []vtt.Clip, videoPath string) (FCPXML, error) {
	absVideoPath, err := filepath.Abs(videoPath)
	if err != nil {
		return FCPXML{}, err
	}

	videoName := filepath.Base(absVideoPath)
	nameWithoutExt := strings.TrimSuffix(videoName, filepath.Ext(videoName))

	// Calculate total duration - textblock, clip, textblock, clip pattern
	var totalDuration time.Duration
	for _, clip := range clips {
		totalDuration += clip.Duration + 10*time.Second // Add 10s for textblock
	}
	totalDuration += 10 * time.Second // Add final textblock

	var spineContent strings.Builder
	currentOffset := time.Duration(0)

	for i, clip := range clips {
		// Textblock gap before each clip - show just the first segment of what's coming next
		clipText := clip.FirstSegmentText

		escapedText := strings.ReplaceAll(clipText, "&", "&amp;")
		escapedText = strings.ReplaceAll(escapedText, "<", "&lt;")
		escapedText = strings.ReplaceAll(escapedText, ">", "&gt;")
		escapedText = strings.ReplaceAll(escapedText, "\"", "&quot;")
		escapedText = strings.ReplaceAll(escapedText, "'", "&#39;")

		spineContent.WriteString(fmt.Sprintf(`
			<gap name="Gap" offset="%s" duration="%s">
				<title ref="r2" lane="1" offset="%s" name="Graphic Text Block" start="%s" duration="%s">
					<text>
						<text-style ref="ts%d">%s</text-style>
					</text>
					<text-style-def id="ts%d">
						<text-style font="Helvetica Neue" fontSize="176.8" fontColor="1 1 1 1"/>
					</text-style-def>
				</title>
			</gap>`,
			FormatDurationForFCPXML(currentOffset),
			FormatDurationForFCPXML(10*time.Second),
			FormatDurationForFCPXML(360*time.Millisecond),
			FormatDurationForFCPXML(360*time.Millisecond),
			FormatDurationForFCPXML(10*time.Second-133*time.Millisecond),
			i+1, escapedText, i+1))

		currentOffset += 10 * time.Second

		// Video clip
		spineContent.WriteString(fmt.Sprintf(`
			<asset-clip ref="r3" offset="%s" name="%s Clip %d" start="%s" duration="%s" tcFormat="NDF" audioRole="dialogue"/>`,
			FormatDurationForFCPXML(currentOffset),
			nameWithoutExt, clip.ClipNum,
			FormatDurationForFCPXML(clip.StartTime),
			FormatDurationForFCPXML(clip.Duration)))

		currentOffset += clip.Duration
	}

	return FCPXML{
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
					ID:            "r3",
					Name:          nameWithoutExt,
					UID:           absVideoPath,
					Start:         "0s",
					HasVideo:      "1",
					Format:        "r1",
					HasAudio:      "1",
					AudioSources:  "1",
					AudioChannels: "2",
					Duration:      FormatDurationForFCPXML(totalDuration),
					MediaRep: MediaRep{
						Kind: "original-media",
						Sig:  absVideoPath,
						Src:  "file://" + absVideoPath,
					},
				},
			},
			Effects: []Effect{
				{
					ID:   "r2",
					Name: "Graphic Text Block",
					UID:  ".../Titles.localized/Basic Text.localized/Graphic Text Block.localized/Graphic Text Block.moti",
				},
			},
		},
		Library: Library{
			Events: []Event{
				{
					Name: "Auto Generated Clips",
					Projects: []Project{
						{
							Name: nameWithoutExt + " Clips",
							Sequences: []Sequence{
								{
									Format:      "r1",
									Duration:    FormatDurationForFCPXML(totalDuration),
									TCStart:     "0s",
									TCFormat:    "NDF",
									AudioLayout: "stereo",
									AudioRate:   "48k",
									Spine: Spine{
										Content: spineContent.String(),
									},
								},
							},
						},
					},
				},
			},
		},
	}, nil
}

func GenerateClipFCPXML(clips []vtt.Clip, videoPath, outputPath string) error {
	fcpxml, err := BuildClipFCPXML(clips, videoPath)
	if err != nil {
		return err
	}

	output, err := xml.MarshalIndent(fcpxml, "", "    ")
	if err != nil {
		return err
	}

	xmlContent := xml.Header + "<!DOCTYPE fcpxml>\n" + string(output)
	return os.WriteFile(outputPath, []byte(xmlContent), 0644)
}

func GenerateWikipediaTableFCPXML(data interface{}, outputPath string) error {
	// Import the wikipedia package here locally to avoid circular imports
	tables, ok := data.([]interface{})
	if !ok {
		return fmt.Errorf("invalid table data format")
	}

	totalDuration := 6 * time.Minute // 6 minutes total
	
	var spineContent strings.Builder
	currentOffset := time.Duration(0)
	
	if len(tables) == 0 {
		return fmt.Errorf("no tables found in Wikipedia data")
	}
	
	// Get the first table
	firstTable := tables[0]
	tableMap, ok := firstTable.(map[string]interface{})
	if !ok {
		return fmt.Errorf("invalid table format")
	}
	
	headers, _ := tableMap["Headers"].([]string)
	rows, _ := tableMap["Rows"].([]interface{})
	
	totalRows := len(rows)
	if totalRows == 0 {
		return fmt.Errorf("no rows found in table")
	}
	
	// Calculate timing - reveal rows slowly over 6 minutes
	rowDuration := totalDuration / time.Duration(totalRows)
	if rowDuration < 2*time.Second {
		rowDuration = 2 * time.Second // Minimum 2 seconds per row
	}
	
	// Create header text first
	if len(headers) > 0 {
		headerText := strings.Join(headers, " | ")
		escapedText := escapeXMLText(headerText)
		
		spineContent.WriteString(fmt.Sprintf(`
			<gap name="Header Gap" offset="%s" duration="%s">
				<text lane="1" offset="%s" name="Table Header" duration="%s">
					<text-style font="Helvetica Neue Bold" fontSize="144" fontColor="1 1 1 1">%s</text-style>
				</text>
			</gap>`,
			FormatDurationForFCPXML(currentOffset),
			FormatDurationForFCPXML(rowDuration),
			FormatDurationForFCPXML(100*time.Millisecond),
			FormatDurationForFCPXML(rowDuration-200*time.Millisecond),
			escapedText))
		
		currentOffset += rowDuration
	}
	
	// Add each row as a text block
	for i, row := range rows {
		rowMap, ok := row.(map[string]interface{})
		if !ok {
			continue
		}
		
		cells, ok := rowMap["Cells"].([]string)
		if !ok {
			continue
		}
		
		if len(cells) == 0 {
			continue
		}
		
		// Create row text - join first few cells
		var displayCells []string
		maxCells := 4 // Show max 4 cells to avoid overcrowding
		for j, cell := range cells {
			if j >= maxCells {
				break
			}
			if strings.TrimSpace(cell) != "" {
				displayCells = append(displayCells, strings.TrimSpace(cell))
			}
		}
		
		if len(displayCells) == 0 {
			continue
		}
		
		rowText := strings.Join(displayCells, " | ")
		if len(rowText) > 100 { // Truncate if too long
			rowText = rowText[:97] + "..."
		}
		
		escapedText := escapeXMLText(rowText)
		
		spineContent.WriteString(fmt.Sprintf(`
			<gap name="Row Gap" offset="%s" duration="%s">
				<text lane="1" offset="%s" name="Table Row %d" duration="%s">
					<text-style font="Helvetica Neue" fontSize="120" fontColor="1 1 1 1">%s</text-style>
				</text>
			</gap>`,
			FormatDurationForFCPXML(currentOffset),
			FormatDurationForFCPXML(rowDuration),
			FormatDurationForFCPXML(100*time.Millisecond),
			i+1,
			FormatDurationForFCPXML(rowDuration-200*time.Millisecond),
			escapedText))
		
		currentOffset += rowDuration
	}
	
	// Create the FCPXML structure
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
		},
		Library: Library{
			Events: []Event{
				{
					Name: "Wikipedia Table",
					Projects: []Project{
						{
							Name: "Wikipedia Table Reveal",
							Sequences: []Sequence{
								{
									Format:      "r1",
									Duration:    FormatDurationForFCPXML(currentOffset),
									TCStart:     "0s",
									TCFormat:    "NDF",
									AudioLayout: "stereo",
									AudioRate:   "48k",
									Spine: Spine{
										Content: spineContent.String(),
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
	return os.WriteFile(outputPath, []byte(xmlContent), 0644)
}

func escapeXMLText(text string) string {
	text = strings.ReplaceAll(text, "&", "&amp;")
	text = strings.ReplaceAll(text, "<", "&lt;")
	text = strings.ReplaceAll(text, ">", "&gt;")
	text = strings.ReplaceAll(text, "\"", "&quot;")
	text = strings.ReplaceAll(text, "'", "&#39;")
	return text
}
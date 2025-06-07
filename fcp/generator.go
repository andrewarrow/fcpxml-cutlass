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

func GenerateEnhancedWikipediaTableFCPXML(data interface{}, outputPath string) error {
	tables, ok := data.([]interface{})
	if !ok {
		return fmt.Errorf("invalid table data format")
	}

	if len(tables) == 0 {
		return fmt.Errorf("no tables found in Wikipedia data")
	}
	
	// Get the first table (largest one)
	firstTable := tables[0]
	tableMap, ok := firstTable.(map[string]interface{})
	if !ok {
		return fmt.Errorf("invalid table format")
	}
	
	headers, _ := tableMap["Headers"].([]string)
	rows, _ := tableMap["Rows"].([]interface{})
	
	// Find the 1986 column index
	year1986Index := -1
	for i, header := range headers {
		if strings.Contains(header, "1986") {
			year1986Index = i
			break
		}
	}
	
	if year1986Index == -1 {
		return fmt.Errorf("1986 column not found in table headers")
	}
	
	// Create a visually appealing table focused on 1986 first
	var spineContent strings.Builder
	currentOffset := time.Duration(0)
	
	// Add generators for different colors
	generators := []Generator{
		{ID: "gen1", Name: "Solid", UID: ".../Generators.localized/Solids.localized/Solid.localized/Solid.moti"},
		{ID: "gen2", Name: "Custom", UID: ".../Generators.localized/Solids.localized/Custom.localized/Custom.moti"},
	}
	
	// Title card: "Andre Agassi - 1986 Tournament Results"
	titleDuration := 3 * time.Second
	spineContent.WriteString(fmt.Sprintf(`
		<gap name="Title Gap" offset="%s" duration="%s">
			<title ref="r2" lane="1" offset="%s" name="Main Title" start="%s" duration="%s">
				<text>
					<text-style ref="ts1">Andre Agassi - 1986 Tournament Results</text-style>
				</text>
				<text-style-def id="ts1">
					<text-style font="SF Pro Display" fontSize="72" fontFace="Bold" fontColor="1 1 1 1" alignment="center"/>
				</text-style-def>
			</title>
		</gap>`,
		FormatDurationForFCPXML(currentOffset),
		FormatDurationForFCPXML(titleDuration),
		FormatDurationForFCPXML(200*time.Millisecond),
		FormatDurationForFCPXML(200*time.Millisecond),
		FormatDurationForFCPXML(titleDuration-400*time.Millisecond)))
	
	currentOffset += titleDuration
	
	// Extract tournament results for 1986
	var tournamentResults []TournamentResult
	fmt.Printf("Extracting 1986 results (column index %d)...\n", year1986Index)
	
	for rowIndex, row := range rows {
		rowMap, ok := row.(map[string]interface{})
		if !ok {
			fmt.Printf("Row %d: not a map\n", rowIndex)
			continue
		}
		
		cells, ok := rowMap["Cells"].([]interface{})
		if !ok {
			fmt.Printf("Row %d: no cells\n", rowIndex)
			continue
		}
		
		fmt.Printf("Row %d: %d cells\n", rowIndex, len(cells))
		
		if len(cells) <= year1986Index {
			fmt.Printf("Row %d: not enough cells for 1986 column\n", rowIndex)
			continue
		}
		
		// Extract tournament name (first cell) and 1986 result
		var tournamentName, result1986 string
		var resultStyle map[string]string
		
		if len(cells) > 0 {
			fmt.Printf("Row %d: First cell type: %T\n", rowIndex, cells[0])
			if firstCell, ok := cells[0].(map[string]interface{}); ok {
				if content, ok := firstCell["Content"].(string); ok {
					tournamentName = content
					fmt.Printf("Row %d: Tournament name: %s\n", rowIndex, tournamentName)
				}
			}
		}
		
		if len(cells) > year1986Index {
			fmt.Printf("Row %d: 1986 cell type: %T\n", rowIndex, cells[year1986Index])
			if cell1986, ok := cells[year1986Index].(map[string]interface{}); ok {
				if content, ok := cell1986["Content"].(string); ok {
					result1986 = content
					fmt.Printf("Row %d: 1986 result: %s\n", rowIndex, result1986)
				}
				if style, ok := cell1986["Style"].(map[string]string); ok {
					resultStyle = style
					fmt.Printf("Row %d: 1986 style: %v\n", rowIndex, resultStyle)
				}
			}
		}
		
		if tournamentName != "" && result1986 != "" && !strings.Contains(tournamentName, "colspan") {
			fmt.Printf("Adding tournament result: %s -> %s\n", tournamentName, result1986)
			tournamentResults = append(tournamentResults, TournamentResult{
				Tournament: tournamentName,
				Result:     result1986,
				Style:      resultStyle,
			})
		}
	}
	
	fmt.Printf("Found %d tournament results for 1986\n", len(tournamentResults))
	
	// Display each tournament result with animation
	resultDuration := 4 * time.Second
	for i, result := range tournamentResults {
		// Background color based on result
		bgColor := getBackgroundColor(result.Result, result.Style)
		
		// Background shape
		spineContent.WriteString(fmt.Sprintf(`
		<gap name="BG Gap %d" offset="%s" duration="%s">
			<generator-clip ref="gen1" lane="-1" offset="%s" name="Background %d" duration="%s">
				<param name="Background Color" key="9999/999166631/999166633/1/100/101" value="%s"/>
			</generator-clip>
		</gap>`,
			i+1,
			FormatDurationForFCPXML(currentOffset),
			FormatDurationForFCPXML(resultDuration),
			FormatDurationForFCPXML(100*time.Millisecond),
			i+1,
			FormatDurationForFCPXML(resultDuration-200*time.Millisecond),
			bgColor))
		
		// Tournament name
		spineContent.WriteString(fmt.Sprintf(`
		<gap name="Tournament Gap %d" offset="%s" duration="%s">
			<title ref="r2" lane="1" offset="%s" name="Tournament %d" start="%s" duration="%s">
				<text>
					<text-style ref="ts%d">%s</text-style>
				</text>
				<text-style-def id="ts%d">
					<text-style font="SF Pro Display" fontSize="48" fontFace="Medium" fontColor="0.1 0.1 0.1 1" alignment="left"/>
				</text-style-def>
			</title>
		</gap>`,
			i+1,
			FormatDurationForFCPXML(currentOffset),
			FormatDurationForFCPXML(resultDuration),
			FormatDurationForFCPXML(300*time.Millisecond),
			i+1,
			FormatDurationForFCPXML(300*time.Millisecond),
			FormatDurationForFCPXML(resultDuration-600*time.Millisecond),
			i+2, escapeXMLText(result.Tournament), i+2))
		
		// Result
		spineContent.WriteString(fmt.Sprintf(`
		<gap name="Result Gap %d" offset="%s" duration="%s">
			<title ref="r2" lane="2" offset="%s" name="Result %d" start="%s" duration="%s">
				<text>
					<text-style ref="ts%d">%s</text-style>
				</text>
				<text-style-def id="ts%d">
					<text-style font="SF Pro Display" fontSize="56" fontFace="Bold" fontColor="0.1 0.1 0.1 1" alignment="right"/>
				</text-style-def>
			</title>
		</gap>`,
			i+1,
			FormatDurationForFCPXML(currentOffset),
			FormatDurationForFCPXML(resultDuration),
			FormatDurationForFCPXML(800*time.Millisecond),
			i+1,
			FormatDurationForFCPXML(800*time.Millisecond),
			FormatDurationForFCPXML(resultDuration-1600*time.Millisecond),
			i+100, escapeXMLText(result.Result), i+100))
		
		currentOffset += resultDuration
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
			Effects: []Effect{
				{
					ID:   "r2",
					Name: "Graphic Text Block",
					UID:  ".../Titles.localized/Basic Text.localized/Graphic Text Block.localized/Graphic Text Block.moti",
				},
			},
			Generators: generators,
		},
		Library: Library{
			Events: []Event{
				{
					Name: "Andre Agassi Tennis",
					Projects: []Project{
						{
							Name: "1986 Tournament Timeline",
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

type TournamentResult struct {
	Tournament string
	Result     string
	Style      map[string]string
}

func getBackgroundColor(result string, style map[string]string) string {
	// Check style first
	if style != nil {
		if bg, ok := style["background"]; ok {
			switch bg {
			case "lime":
				return "0.2 0.8 0.2 1" // Green for wins
			case "yellow":
				return "1 1 0.2 1" // Yellow for semifinals
			case "thistle":
				return "0.8 0.6 0.8 1" // Purple for finals
			case "#afeeee":
				return "0.7 0.9 0.9 1" // Light blue for rounds
			case "#ffebcd":
				return "1 0.9 0.8 1" // Light orange for quarterfinals
			}
		}
		if bgColor, ok := style["background-color"]; ok {
			switch bgColor {
			case "lime":
				return "0.2 0.8 0.2 1"
			case "yellow":
				return "1 1 0.2 1"
			case "thistle":
				return "0.8 0.6 0.8 1"
			}
		}
	}
	
	// Fallback based on result content
	switch result {
	case "W", "'''W'''":
		return "0.2 0.8 0.2 1" // Green for wins
	case "F":
		return "0.8 0.6 0.8 1" // Purple for finals
	case "SF":
		return "1 1 0.2 1" // Yellow for semifinals
	case "QF":
		return "1 0.9 0.8 1" // Light orange for quarterfinals
	case "1R", "2R", "3R", "4R":
		return "0.7 0.9 0.9 1" // Light blue for rounds
	case "A":
		return "0.9 0.9 0.9 1" // Light gray for absent
	case "DNQ":
		return "0.8 0.8 0.8 1" // Gray for did not qualify
	default:
		return "0.95 0.95 0.95 1" // Very light gray default
	}
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
				<title ref="r2" lane="1" offset="%s" name="Table Header" start="%s" duration="%s">
					<text>
						<text-style ref="ts1">%s</text-style>
					</text>
					<text-style-def id="ts1">
						<text-style font="Helvetica Neue" fontSize="176.8" fontColor="1 1 1 1"/>
					</text-style-def>
				</title>
			</gap>`,
			FormatDurationForFCPXML(currentOffset),
			FormatDurationForFCPXML(rowDuration),
			FormatDurationForFCPXML(100*time.Millisecond),
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
				<title ref="r2" lane="1" offset="%s" name="Table Row %d" start="%s" duration="%s">
					<text>
						<text-style ref="ts%d">%s</text-style>
					</text>
					<text-style-def id="ts%d">
						<text-style font="Helvetica Neue" fontSize="176.8" fontColor="1 1 1 1"/>
					</text-style-def>
				</title>
			</gap>`,
			FormatDurationForFCPXML(currentOffset),
			FormatDurationForFCPXML(rowDuration),
			FormatDurationForFCPXML(100*time.Millisecond),
			i+1,
			FormatDurationForFCPXML(100*time.Millisecond),
			FormatDurationForFCPXML(rowDuration-200*time.Millisecond),
			i+2, escapedText, i+2))
		
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
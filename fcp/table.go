package fcp

import (
	"encoding/xml"
	"fmt"
	"os"
	"strings"
	"time"
)

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func getCenter(params []Param) string {
	for _, param := range params {
		if param.Name == "Center" {
			return param.Value
		}
	}
	return "unknown"
}

func buildSpineContent(elements []interface{}) string {
	var content strings.Builder
	for _, elem := range elements {
		switch e := elem.(type) {
		case Video:
			xml, _ := xml.MarshalIndent(e, "                        ", "    ")
			content.WriteString("\n                        ")
			content.Write(xml)
		case Title:
			xml, _ := xml.MarshalIndent(e, "                        ", "    ")
			content.WriteString("\n                        ")
			content.Write(xml)
		}
	}
	content.WriteString("\n                    ")
	return content.String()
}

type TableResult struct {
	Column1 string
	Column2 string
	Style   map[string]string
}

func GenerateTableGridFCPXML(tableData *TableData, outputPath string) error {
	fmt.Printf("DEBUG: GenerateTableGridFCPXML called with outputPath: %s\n", outputPath)
	
	// Use default data if tableData is nil
	if tableData == nil {
		fmt.Printf("DEBUG: tableData is nil, using default data\n")
		tableData = &TableData{
			Headers: []string{"Column 1", "Column 2"},
			Rows: []TableRow{
				{Cells: []TableCell{{Content: "Sample Item", Style: map[string]string{"background": "#f0f0f0"}}, {Content: "Sample Value"}}},
				{Cells: []TableCell{{Content: "Another Item"}, {Content: "Another Value"}}},
			},
		}
	} else {
		fmt.Printf("DEBUG: tableData provided with %d headers and %d rows\n", len(tableData.Headers), len(tableData.Rows))
		fmt.Printf("DEBUG: Headers: %v\n", tableData.Headers)
	}

	// Convert TableData to the format expected by the rest of the function
	var tableResults []TableResult
	for _, row := range tableData.Rows {
		if len(row.Cells) >= 2 {
			tableResults = append(tableResults, TableResult{
				Column1: row.Cells[0].Content,
				Column2: row.Cells[1].Content,
				Style:   row.Cells[1].Style,
			})
		} else if len(row.Cells) == 1 {
			tableResults = append(tableResults, TableResult{
				Column1: row.Cells[0].Content,
				Column2: "",
				Style:   row.Cells[0].Style,
			})
		}
	}

	if len(tableResults) == 0 {
		tableResults = []TableResult{
			{Column1: "Sample Item", Column2: "Sample Value", Style: map[string]string{"background": "#f0f0f0"}},
			{Column1: "Another Item", Column2: "Another Value", Style: map[string]string{}},
		}
	}

	totalDuration := 15 * time.Second
	currentOffset := time.Duration(0)

	numRows := len(tableResults) + 1
	numCols := 2
	
	fmt.Printf("DEBUG: numRows=%d (data rows + 1 header), numCols=%d\n", numRows, numCols)
	fmt.Printf("DEBUG: totalDuration=%v, currentOffset=%v\n", totalDuration, currentOffset)

	var horizontalPositions []float64
	startY := 0.500  // Center the lines around middle of screen like in table.fcpxml
	rowSpacing := 0.080  // Smaller spacing to keep lines visible
	fmt.Printf("DEBUG: Calculating horizontal positions with startY=%.3f, rowSpacing=%.3f\n", startY, rowSpacing)
	for i := 0; i <= min(3, numRows); i++ {  // Only create a few key horizontal lines like the reference
		pos := startY + rowSpacing*float64(i-1)  // Center around startY
		horizontalPositions = append(horizontalPositions, pos)
		fmt.Printf("DEBUG: horizontalPositions[%d] = %.3f\n", i, pos)
	}

	verticalPositions := []float64{0.100, 0.500, 0.900}
	fmt.Printf("DEBUG: verticalPositions: %v\n", verticalPositions)

	primaryHeader := "Column 1"
	secondaryHeader := "Column 2"
	if len(tableData.Headers) > 0 {
		primaryHeader = tableData.Headers[0]
	}
	if len(tableData.Headers) > 1 {
		secondaryHeader = tableData.Headers[1]
	}

	var spineElements []interface{}

	// Create proper table grid with many lines
	fmt.Printf("DEBUG: Creating full table grid structure\n")
	fmt.Printf("DEBUG: Table has %d headers, so need %d vertical lines\n", len(tableData.Headers), len(tableData.Headers)+1)
	fmt.Printf("DEBUG: Table has %d data rows + 1 header = %d total rows, so need %d horizontal lines\n", 
		len(tableData.Rows), numRows, numRows+1)
	
	// Calculate positions for proper table grid
	tableTop := 0.2     // Start table near top of screen  
	tableBottom := 0.8  // End table near bottom of screen
	tableLeft := 0.05   // Start table near left edge
	tableRight := 0.95  // End table near right edge
	
	// Calculate horizontal line positions (rows)
	var tableHorizontalPositions []float64
	for i := 0; i <= numRows; i++ {
		yPos := tableTop + (tableBottom-tableTop)*float64(i)/float64(numRows)
		tableHorizontalPositions = append(tableHorizontalPositions, yPos)
		fmt.Printf("DEBUG: Horizontal line %d at Y=%.3f\n", i, yPos)
	}
	
	// Calculate vertical line positions (columns) - limit to first 10 columns for visibility
	maxCols := min(10, len(tableData.Headers))
	var tableVerticalPositions []float64
	for j := 0; j <= maxCols; j++ {
		xPos := tableLeft + (tableRight-tableLeft)*float64(j)/float64(maxCols)
		tableVerticalPositions = append(tableVerticalPositions, xPos)
		fmt.Printf("DEBUG: Vertical line %d at X=%.3f\n", j, xPos)
	}
	
	// Create stacked horizontal lines using nested structure like table.fcpxml
	fmt.Printf("DEBUG: Creating stacked horizontal lines using nested structure\n")
	fmt.Printf("DEBUG: Will create main video with %d nested lines inside\n", len(tableHorizontalPositions))
	
	// Create nested videos for all horizontal lines
	var nestedHorizontalVideos []Video
	for i, yPos := range tableHorizontalPositions {
		if i == 0 {
			continue // Skip first one as it will be the main video
		}
		
		// Calculate position offset relative to main line
		mainYPos := tableHorizontalPositions[0] // First line position
		offsetY := (yPos - mainYPos) * 1000 // Scale up the offset for visibility
		
		nestedVideo := Video{
			Ref:      "r2",
			Lane:     "1",
			Offset:   FormatDurationForFCPXML(currentOffset + time.Duration(i)*time.Second),
			Name:     fmt.Sprintf("H-Line %d Nested", i),
			Start:    FormatDurationForFCPXML(currentOffset),
			Duration: FormatDurationForFCPXML(totalDuration - time.Duration(i)*time.Second),
			Params: []Param{
				{Name: "Drop Shadow Opacity", Key: "9999/988455508/1/208/211", Value: "0.7426"},
				{Name: "Feather", Key: "9999/988455508/988455699/2/353/102", Value: "3"},
				{Name: "Fill Color", Key: "9999/988455508/988455699/2/353/113/111", Value: "1.0817 -0.0799793 -0.145856"},
				{Name: "Falloff", Key: "9999/988455508/988455699/2/353/158", Value: "-2"},
				{Name: "Shape", Key: "9999/988461322/100/988461395/2/100", Value: "4 (Rectangle)"},
				{Name: "Outline", Key: "9999/988461322/100/988464485/2/100", Value: "0"},
				{Name: "Outline Width", Key: "9999/988461322/100/988467855/2/100", Value: "0.338788"},
				{Name: "Corners", Key: "9999/988461322/100/988469428/2/100", Value: "1 (Square)"},
				{Name: "Center", Key: "9999/988469355/988469353/3/988469357/1", Value: fmt.Sprintf("0.5 %.2f", mainYPos)},
			},
			AdjustTransform: &AdjustTransform{Position: fmt.Sprintf("0 %.1f", offsetY), Scale: "1 0.0394"},
		}
		
		fmt.Printf("DEBUG: Created nested H-Line %d - Position offset: 0 %.1f, Center: 0.5 %.2f\n", i, offsetY, mainYPos)
		nestedHorizontalVideos = append(nestedHorizontalVideos, nestedVideo)
	}
	
	// Create main horizontal video with all nested lines
	mainYPos := tableHorizontalPositions[0]
	mainHorizontalVideo := Video{
		Ref:      "r2",
		Offset:   FormatDurationForFCPXML(currentOffset),
		Name:     "H-Lines Main Container",
		Start:    FormatDurationForFCPXML(currentOffset),
		Duration: FormatDurationForFCPXML(totalDuration),
		Params: []Param{
			{Name: "Drop Shadow Opacity", Key: "9999/988455508/1/208/211", Value: "0.7426"},
			{Name: "Feather", Key: "9999/988455508/988455699/2/353/102", Value: "3"},
			{Name: "Fill Color", Key: "9999/988455508/988455699/2/353/113/111", Value: "1.0817 -0.0799793 -0.145856"},
			{Name: "Falloff", Key: "9999/988455508/988455699/2/353/158", Value: "-2"},
			{Name: "Shape", Key: "9999/988461322/100/988461395/2/100", Value: "4 (Rectangle)"},
			{Name: "Outline", Key: "9999/988461322/100/988464485/2/100", Value: "0"},
			{Name: "Outline Width", Key: "9999/988461322/100/988467855/2/100", Value: "0.338788"},
			{Name: "Corners", Key: "9999/988461322/100/988469428/2/100", Value: "1 (Square)"},
			{Name: "Center", Key: "9999/988469355/988469353/3/988469357/1", Value: fmt.Sprintf("0.5 %.2f", mainYPos)},
		},
		AdjustTransform: &AdjustTransform{Scale: "1 0.0395"},
		NestedVideos:    nestedHorizontalVideos,
	}
	
	fmt.Printf("DEBUG: Created main H-Lines container with %d nested lines\n", len(nestedHorizontalVideos))
	spineElements = append(spineElements, mainHorizontalVideo)

	// Create stacked vertical lines using nested structure
	fmt.Printf("DEBUG: Creating stacked vertical lines using nested structure\n")
	fmt.Printf("DEBUG: Will create main video with %d nested vertical lines inside\n", len(tableVerticalPositions))
	
	// Create nested videos for all vertical lines
	var nestedVerticalVideos []Video
	for j, xPos := range tableVerticalPositions {
		if j == 0 {
			continue // Skip first one as it will be the main video
		}
		
		// Calculate position offset relative to main line
		mainXPos := tableVerticalPositions[0] // First line position
		offsetX := (xPos - mainXPos) * 1000 // Scale up the offset for visibility
		
		nestedVideo := Video{
			Ref:      "r2",
			Lane:     "2",
			Offset:   FormatDurationForFCPXML(currentOffset + time.Duration(j)*time.Second),
			Name:     fmt.Sprintf("V-Line %d Nested", j),
			Start:    FormatDurationForFCPXML(currentOffset),
			Duration: FormatDurationForFCPXML(totalDuration - time.Duration(j)*time.Second),
			Params: []Param{
				{Name: "Drop Shadow Opacity", Key: "9999/988455508/1/208/211", Value: "0.7426"},
				{Name: "Feather", Key: "9999/988455508/988455699/2/353/102", Value: "3"},
				{Name: "Fill Color", Key: "9999/988455508/988455699/2/353/113/111", Value: "1.0817 -0.0799793 -0.145856"},
				{Name: "Falloff", Key: "9999/988455508/988455699/2/353/158", Value: "-2"},
				{Name: "Shape", Key: "9999/988461322/100/988461395/2/100", Value: "4 (Rectangle)"},
				{Name: "Outline", Key: "9999/988461322/100/988464485/2/100", Value: "0"},
				{Name: "Outline Width", Key: "9999/988461322/100/988467855/2/100", Value: "0.338788"},
				{Name: "Corners", Key: "9999/988461322/100/988469428/2/100", Value: "1 (Square)"},
				{Name: "Center", Key: "9999/988469355/988469353/3/988469357/1", Value: fmt.Sprintf("%.2f 0.5", mainXPos)},
			},
			AdjustTransform: &AdjustTransform{Position: fmt.Sprintf("%.1f 0", offsetX), Scale: "0.0394 1"},
		}
		
		fmt.Printf("DEBUG: Created nested V-Line %d - Position offset: %.1f 0, Center: %.2f 0.5\n", j, offsetX, mainXPos)
		nestedVerticalVideos = append(nestedVerticalVideos, nestedVideo)
	}
	
	// Create main vertical video with all nested lines
	mainXPos := tableVerticalPositions[0]
	mainVerticalVideo := Video{
		Ref:      "r2",
		Lane:     "1",
		Offset:   FormatDurationForFCPXML(currentOffset),
		Name:     "V-Lines Main Container",
		Start:    FormatDurationForFCPXML(currentOffset),
		Duration: FormatDurationForFCPXML(totalDuration),
		Params: []Param{
			{Name: "Drop Shadow Opacity", Key: "9999/988455508/1/208/211", Value: "0.7426"},
			{Name: "Feather", Key: "9999/988455508/988455699/2/353/102", Value: "3"},
			{Name: "Fill Color", Key: "9999/988455508/988455699/2/353/113/111", Value: "1.0817 -0.0799793 -0.145856"},
			{Name: "Falloff", Key: "9999/988455508/988455699/2/353/158", Value: "-2"},
			{Name: "Shape", Key: "9999/988461322/100/988461395/2/100", Value: "4 (Rectangle)"},
			{Name: "Outline", Key: "9999/988461322/100/988464485/2/100", Value: "0"},
			{Name: "Outline Width", Key: "9999/988461322/100/988467855/2/100", Value: "0.338788"},
			{Name: "Corners", Key: "9999/988461322/100/988469428/2/100", Value: "1 (Square)"},
			{Name: "Center", Key: "9999/988469355/988469353/3/988469357/1", Value: fmt.Sprintf("%.2f 0.5", mainXPos)},
		},
		AdjustTransform: &AdjustTransform{Scale: "0.0395 1"},
		NestedVideos:    nestedVerticalVideos,
	}
	
	fmt.Printf("DEBUG: Created main V-Lines container with %d nested lines\n", len(nestedVerticalVideos))
	spineElements = append(spineElements, mainVerticalVideo)

	// Header - left column
	spineElements = append(spineElements, Title{
		Ref:      "r3",
		Lane:     "2",
		Offset:   FormatDurationForFCPXML(currentOffset),
		Name:     fmt.Sprintf("Header %s", escapeXMLText(primaryHeader)),
		Start:    FormatDurationForFCPXML(currentOffset),
		Duration: FormatDurationForFCPXML(totalDuration),
		Params: []Param{
			{Name: "Position", Key: "9999/10003/13260/3296672360/1/100/101", Value: "-200.0 -259.2"},
			{Name: "Layout Method", Key: "9999/10003/13260/3296672360/2/314", Value: "1 (Paragraph)"},
			{Name: "Alignment", Key: "9999/10003/13260/3296672360/2/354/3296667315/401", Value: "1 (Center)"},
		},
		Text: TitleText{
			TextStyle: TextStyleRef{
				Ref:  "ts1",
				Text: escapeXMLText(primaryHeader),
			},
		},
		TextStyleDef: TextStyleDef{
			ID: "ts1",
			TextStyle: TextStyle{
				Font:      "SF Pro Display",
				FontSize:  "36",
				FontFace:  "Bold",
				FontColor: "0.1 0.1 0.1 1",
				Alignment: "center",
			},
		},
	})

	// Header - right column
	spineElements = append(spineElements, Title{
		Ref:      "r3",
		Lane:     "3",
		Offset:   FormatDurationForFCPXML(currentOffset),
		Name:     fmt.Sprintf("Header %s", escapeXMLText(secondaryHeader)),
		Start:    FormatDurationForFCPXML(currentOffset),
		Duration: FormatDurationForFCPXML(totalDuration),
		Params: []Param{
			{Name: "Position", Key: "9999/10003/13260/3296672360/1/100/101", Value: "200.0 -259.2"},
			{Name: "Layout Method", Key: "9999/10003/13260/3296672360/2/314", Value: "1 (Paragraph)"},
			{Name: "Alignment", Key: "9999/10003/13260/3296672360/2/354/3296667315/401", Value: "1 (Center)"},
		},
		Text: TitleText{
			TextStyle: TextStyleRef{
				Ref:  "ts2",
				Text: escapeXMLText(secondaryHeader),
			},
		},
		TextStyleDef: TextStyleDef{
			ID: "ts2",
			TextStyle: TextStyle{
				Font:      "SF Pro Display",
				FontSize:  "36",
				FontFace:  "Bold",
				FontColor: "0.1 0.1 0.1 1",
				Alignment: "center",
			},
		},
	})

	var textYPositions []float64
	textStartPx := -129.6
	textSpacingPx := 129.6
	for i := 0; i < len(tableResults); i++ {
		textYPositions = append(textYPositions, textStartPx+float64(i)*textSpacingPx)
	}

	var cellCenterPositions []float64
	for i := 0; i < len(tableResults); i++ {
		centerY := startY + rowSpacing*float64(i+1) + rowSpacing/2
		cellCenterPositions = append(cellCenterPositions, centerY)
	}

	// Add table data with gradual reveal
	for i, result := range tableResults {
		revealTime := currentOffset + time.Duration(i+1)*2*time.Second
		cellDuration := totalDuration - time.Duration(i+1)*2*time.Second
		
		// Ensure duration is never negative
		if cellDuration <= 0 {
			cellDuration = time.Second // Minimum 1 second duration
		}

		col1EscAttr := escapeXMLText(result.Column1)
		col2EscAttr := escapeXMLText(result.Column2)

		bgColor := getBackgroundColor(result.Column2, result.Style)

		// Background shape for result cell
		spineElements = append(spineElements, Video{
			Ref:      "r2",
			Lane:     "4",
			Offset:   FormatDurationForFCPXML(revealTime),
			Name:     fmt.Sprintf("BG %s", col1EscAttr),
			Start:    FormatDurationForFCPXML(revealTime),
			Duration: FormatDurationForFCPXML(cellDuration),
			Params: []Param{
				{Name: "Shape", Key: "9999/988461322/100/988461395/2/100", Value: "4 (Rectangle)"},
				{Name: "Fill Color", Key: "9999/988455508/988455699/2/353/113/111", Value: bgColor},
				{Name: "Outline", Key: "9999/988461322/100/988464485/2/100", Value: "0"},
				{Name: "Center", Key: "9999/988469355/988469353/3/988469357/1", Value: fmt.Sprintf("0.700 %.3f", cellCenterPositions[i])},
			},
			AdjustTransform: &AdjustTransform{Scale: "0.380 0.096"},
		})

		// Column 1 text
		spineElements = append(spineElements, Title{
			Ref:      "r3",
			Lane:     "5",
			Offset:   FormatDurationForFCPXML(revealTime),
			Name:     col1EscAttr,
			Start:    FormatDurationForFCPXML(revealTime),
			Duration: FormatDurationForFCPXML(cellDuration),
			Params: []Param{
				{Name: "Position", Key: "9999/10003/13260/3296672360/1/100/101", Value: fmt.Sprintf("-200.0 %.1f", textYPositions[i])},
				{Name: "Layout Method", Key: "9999/10003/13260/3296672360/2/314", Value: "1 (Paragraph)"},
				{Name: "Alignment", Key: "9999/10003/13260/3296672360/2/354/3296667315/401", Value: "1 (Center)"},
			},
			Text: TitleText{
				TextStyle: TextStyleRef{
					Ref:  fmt.Sprintf("ts%d", i+10),
					Text: escapeXMLText(result.Column1),
				},
			},
			TextStyleDef: TextStyleDef{
				ID: fmt.Sprintf("ts%d", i+10),
				TextStyle: TextStyle{
					Font:      "SF Pro Display",
					FontSize:  "28",
					FontFace:  "Medium",
					FontColor: "0.1 0.1 0.1 1",
					Alignment: "center",
				},
			},
		})

		// Column 2 text
		spineElements = append(spineElements, Title{
			Ref:      "r3",
			Lane:     "6",
			Offset:   FormatDurationForFCPXML(revealTime),
			Name:     fmt.Sprintf("Result %s", col2EscAttr),
			Start:    FormatDurationForFCPXML(revealTime),
			Duration: FormatDurationForFCPXML(cellDuration),
			Params: []Param{
				{Name: "Position", Key: "9999/10003/13260/3296672360/1/100/101", Value: fmt.Sprintf("200.0 %.1f", textYPositions[i])},
				{Name: "Layout Method", Key: "9999/10003/13260/3296672360/2/314", Value: "1 (Paragraph)"},
				{Name: "Alignment", Key: "9999/10003/13260/3296672360/2/354/3296667315/401", Value: "1 (Center)"},
			},
			Text: TitleText{
				TextStyle: TextStyleRef{
					Ref:  fmt.Sprintf("ts%d", i+50),
					Text: escapeXMLText(result.Column2),
				},
			},
			TextStyleDef: TextStyleDef{
				ID: fmt.Sprintf("ts%d", i+50),
				TextStyle: TextStyle{
					Font:      "SF Pro Display",
					FontSize:  "32",
					FontFace:  "Bold",
					FontColor: "0.1 0.1 0.1 1",
					Alignment: "center",
				},
			},
		})
	}

	fmt.Printf("DEBUG: SUMMARY - Total spine elements created: %d\n", len(spineElements))
	
	// Count different types of elements
	hLineCount := 0
	vLineCount := 0
	titleCount := 0
	bgCount := 0
	
	for i, elem := range spineElements {
		switch e := elem.(type) {
		case Video:
			if strings.HasPrefix(e.Name, "H-Line") {
				hLineCount++
				fmt.Printf("DEBUG: Element %d: HORIZONTAL LINE - Name: %s, Center: %s\n", i, e.Name, getCenter(e.Params))
			} else if strings.HasPrefix(e.Name, "V-Line") {
				vLineCount++
				fmt.Printf("DEBUG: Element %d: VERTICAL LINE - Name: %s, Center: %s\n", i, e.Name, getCenter(e.Params))
			} else if strings.HasPrefix(e.Name, "BG") {
				bgCount++
				fmt.Printf("DEBUG: Element %d: BACKGROUND - Name: %s, Lane: %s\n", i, e.Name, e.Lane)
			} else {
				fmt.Printf("DEBUG: Element %d: VIDEO - Name: %s, Lane: %s\n", i, e.Name, e.Lane)
			}
		case Title:
			titleCount++
			fmt.Printf("DEBUG: Element %d: TITLE - Name: %s, Lane: %s\n", i, e.Name, e.Lane)
		}
	}
	
	fmt.Printf("DEBUG: ELEMENT COUNTS - H-Lines: %d, V-Lines: %d, Titles: %d, Backgrounds: %d\n", 
		hLineCount, vLineCount, titleCount, bgCount)
	
	spineContent := buildSpineContent(spineElements)
	fmt.Printf("DEBUG: Spine content generated, length: %d characters\n", len(spineContent))

	fcpxml := FCPXML{
		Version: "1.13",
		Resources: Resources{
			Formats: []Format{
				{
					ID:            "r1",
					Name:          "FFVideoFormat1080p2997",
					FrameDuration: "1001/30000s",
					Width:         "1920",
					Height:        "1080",
					ColorSpace:    "1-1-1 (Rec. 709)",
				},
			},
			Effects: []Effect{
				{
					ID:   "r2",
					Name: "Shapes",
					UID:  ".../Generators.localized/Elements.localized/Shapes.localized/Shapes.motn",
				},
				{
					ID:   "r3",
					Name: "Text",
					UID:  ".../Titles.localized/Basic Text.localized/Text.localized/Text.moti",
				},
			},
		},
		Library: Library{
			Events: []Event{
				{
					Name: "Table View",
					Projects: []Project{
						{
							Name: "Data Table",
							Sequences: []Sequence{
								{
									Format:      "r1",
									Duration:    FormatDurationForFCPXML(totalDuration),
									TCStart:     "0s",
									TCFormat:    "NDF",
									AudioLayout: "stereo",
									AudioRate:   "48k",
									Spine: Spine{
										Content: spineContent,
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


func getBackgroundColor(content string, style map[string]string) string {
	if style != nil {
		if bg, ok := style["background"]; ok {
			switch bg {
			case "lime":
				return "0.2 0.8 0.2"
			case "yellow":
				return "1 1 0.2"
			case "thistle":
				return "0.8 0.6 0.8"
			case "#afeeee":
				return "0.7 0.9 0.9"
			case "#ffebcd":
				return "1 0.9 0.8"
			case "lightblue":
				return "0.7 0.9 1.0"
			case "lightgreen":
				return "0.7 1.0 0.7"
			case "lightgray":
				return "0.9 0.9 0.9"
			}
		}
		if bgColor, ok := style["background-color"]; ok {
			switch bgColor {
			case "lime":
				return "0.2 0.8 0.2"
			case "yellow":
				return "1 1 0.2"
			case "thistle":
				return "0.8 0.6 0.8"
			case "lightblue":
				return "0.7 0.9 1.0"
			case "lightgreen":
				return "0.7 1.0 0.7"
			}
		}
	}

	return "0.95 0.95 0.95"
}

func GenerateWikipediaTableFCPXML(tableData *TableData, outputPath string) error {
	return GenerateTableGridFCPXML(tableData, outputPath)
}
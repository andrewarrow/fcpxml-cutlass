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

type Position struct {
	X, Y float64
}

func calculateCellTextPositions(horizontalOffsets, verticalOffsets []float64) [][]Position {
	var positions [][]Position
	
	// Calculate the center position of each cell formed by the grid lines
	for row := 0; row < len(horizontalOffsets)-1; row++ {
		var rowPositions []Position
		for col := 0; col < len(verticalOffsets)-1; col++ {
			// Calculate center X position between two vertical lines
			centerX := (verticalOffsets[col] + verticalOffsets[col+1]) / 2
			// Calculate center Y position between two horizontal lines
			centerY := (horizontalOffsets[row] + horizontalOffsets[row+1]) / 2
			
			rowPositions = append(rowPositions, Position{
				X: centerX,
				Y: centerY,
			})
		}
		positions = append(positions, rowPositions)
	}
	
	return positions
}

func GenerateTableGridFCPXML(tableData *TableData, outputPath string) error {
	fmt.Printf("DEBUG: GenerateTableGridFCPXML called with outputPath: %s\n", outputPath)
	
	// Use default tennis data if tableData is nil - Andre Agassi tournament results by year
	if tableData == nil {
		fmt.Printf("DEBUG: tableData is nil, using default tennis data\n")
		tableData = &TableData{
			Headers: []string{"Tournament", "1986"},
			Rows: []TableRow{
				{Cells: []TableCell{{Content: "Australian Open"}, {Content: "NH"}}},
				{Cells: []TableCell{{Content: "French Open"}, {Content: "A"}}},
				{Cells: []TableCell{{Content: "Wimbledon"}, {Content: "A"}}},
				{Cells: []TableCell{{Content: "US Open"}, {Content: "1R"}}},
			},
		}
	} else {
		fmt.Printf("DEBUG: tableData provided with %d headers and %d rows\n", len(tableData.Headers), len(tableData.Rows))
		fmt.Printf("DEBUG: Headers: %v\n", tableData.Headers)
	}

	// Total duration: 190 seconds (19 years * 10 seconds each)
	totalDuration := 190 * time.Second
	
	// Calculate grid dimensions with FCP layer limits in mind
	// FCP has a practical limit of ~50-60 nested elements before performance issues
	// Each row+col creates multiple elements (lines + text), so limit conservatively
	const maxFCPRows = 5    // Maximum 5 data rows + 1 header = 6 total rows
	const maxFCPCols = 4    // Maximum 4 columns to stay within layer limits
	
	maxRows := min(maxFCPRows, len(tableData.Rows))     // Limit rows for FCP
	maxCols := min(maxFCPCols, len(tableData.Headers))  // Limit columns for FCP
	totalRows := maxRows + 1  // Add 1 for header row
	
	// Calculate and warn about element counts
	totalHorizontalLines := totalRows + 1  // rows + 1 for borders
	totalVerticalLines := maxCols + 1      // cols + 1 for borders  
	totalLines := totalHorizontalLines + totalVerticalLines
	totalTextElements := (maxCols) + (maxRows * maxCols) // headers + data cells
	totalElements := totalLines + totalTextElements
	
	fmt.Printf("DEBUG: Creating %dx%d table (including header)\n", totalRows, maxCols)
	fmt.Printf("DEBUG: Element count - Lines: %d, Text: %d, Total: %d\n", totalLines, totalTextElements, totalElements)
	
	if len(tableData.Rows) > maxFCPRows {
		fmt.Printf("DEBUG: Limited rows from %d to %d for FCP compatibility\n", len(tableData.Rows), maxFCPRows)
	}
	if len(tableData.Headers) > maxFCPCols {
		fmt.Printf("DEBUG: Limited columns from %d to %d for FCP compatibility\n", len(tableData.Headers), maxFCPCols)
	}
	
	// Create more lines for proper table grid
	// Generate horizontal lines: top border, header separator, row separators, bottom border
	horizontalPositionOffsets := make([]float64, totalRows+1)
	startY := -100.0
	endY := 100.0
	stepY := (endY - startY) / float64(totalRows)
	for i := 0; i <= totalRows; i++ {
		horizontalPositionOffsets[i] = startY + float64(i)*stepY
	}
	
	// Generate vertical lines: left border, column separators, right border
	verticalPositionOffsets := make([]float64, maxCols+1)
	startX := -150.0
	endX := 150.0
	stepX := (endX - startX) / float64(maxCols)
	for i := 0; i <= maxCols; i++ {
		verticalPositionOffsets[i] = startX + float64(i)*stepX
	}
	
	// Lines are already generated with exact count needed
	
	fmt.Printf("DEBUG: Using %d horizontal lines and %d vertical lines\n", 
		len(horizontalPositionOffsets), len(verticalPositionOffsets))
	
	// Create all nested elements for the main video
	var nestedVideos []Video
	var nestedTitles []Title
	laneCounter := 1
	
	// Add all horizontal lines as nested videos
	for i, yOffset := range horizontalPositionOffsets {
		horizontalLine := Video{
			Ref:      "r2",
			Lane:     fmt.Sprintf("%d", laneCounter),
			Offset:   "0s",
			Name:     fmt.Sprintf("Horizontal Line %d", i+1),
			Start:    "0s",
			Duration: FormatDurationForFCPXML(totalDuration),
			Params: []Param{
				{Name: "Shape", Key: "9999/988461322/100/988461395/2/100", Value: "4 (Rectangle)"},
				{Name: "Fill Color", Key: "9999/988455508/988455699/2/353/113/111", Value: "1 0 0"},
				{Name: "Outline", Key: "9999/988461322/100/988464485/2/100", Value: "0"},
				{Name: "Corners", Key: "9999/988461322/100/988469428/2/100", Value: "1 (Square)"},
			},
			AdjustTransform: &AdjustTransform{Position: fmt.Sprintf("0 %.1f", yOffset), Scale: "30 0.05"},
		}
		nestedVideos = append(nestedVideos, horizontalLine)
		laneCounter++
	}
	
	// Add all vertical lines as nested videos
	for j, xOffset := range verticalPositionOffsets {
		verticalLine := Video{
			Ref:      "r2",
			Lane:     fmt.Sprintf("%d", laneCounter),
			Offset:   "0s",
			Name:     fmt.Sprintf("Vertical Line %d", j+1),
			Start:    "0s",
			Duration: FormatDurationForFCPXML(totalDuration),
			Params: []Param{
				{Name: "Shape", Key: "9999/988461322/100/988461395/2/100", Value: "4 (Rectangle)"},
				{Name: "Fill Color", Key: "9999/988455508/988455699/2/353/113/111", Value: "1 0 0"},
				{Name: "Outline", Key: "9999/988461322/100/988464485/2/100", Value: "0"},
				{Name: "Corners", Key: "9999/988461322/100/988469428/2/100", Value: "1 (Square)"},
			},
			AdjustTransform: &AdjustTransform{Position: fmt.Sprintf("%.1f 0", xOffset), Scale: "0.0081 30"},
		}
		nestedVideos = append(nestedVideos, verticalLine)
		laneCounter++
	}
	
	// Calculate cell positions for text placement
	// Position text in the center of each cell based on grid lines
	cellTextPositions := calculateCellTextPositions(horizontalPositionOffsets, verticalPositionOffsets)
	
	// Tennis tournament data by year (1986-2004)
	yearsData := map[int][]string{
		1986: {"NH", "A", "A", "1R"},
		1987: {"A", "2R", "1R", "1R"},
		1988: {"A", "SF", "A", "SF"},
		1989: {"A", "3R", "A", "SF"},
		1990: {"A", "F", "A", "F"},
		1991: {"A", "F", "QF", "1R"},
		1992: {"A", "SF", "W", "QF"},
		1993: {"A", "A", "QF", "1R"},
		1994: {"A", "2R", "4R", "W"},
		1995: {"W", "QF", "SF", "F"},
		1996: {"SF", "2R", "1R", "SF"},
		1997: {"A", "A", "A", "4R"},
		1998: {"A", "1R", "2R", "4R"},
		1999: {"4R", "W", "F", "W"},
		2000: {"A", "2R", "SF", "2R"},
		2001: {"W", "QF", "SF", "QF"},
		2002: {"A", "QF", "2R", "F"},
		2003: {"W", "QF", "4R", "SF"},
		2004: {"A", "1R", "A", "QF"},
	}

	// Add static Tournament column header (always visible)
	tournamentHeaderTitle := Title{
		Ref:      "r3",
		Lane:     fmt.Sprintf("%d", laneCounter),
		Offset:   "0s",
		Name:     "Tournament Header",
		Start:    "0s",
		Duration: FormatDurationForFCPXML(totalDuration),
		Params: []Param{
			{Name: "Position", Key: "9999/10003/13260/3296672360/1/100/101", Value: fmt.Sprintf("%.0f %.0f", cellTextPositions[0][0].X*10, cellTextPositions[0][0].Y*10)},
		},
		Text: &TitleText{
			TextStyle: TextStyleRef{
				Ref:  "tournament-header-style",
				Text: "Tournament",
			},
		},
		TextStyleDef: &TextStyleDef{
			ID: "tournament-header-style",
			TextStyle: TextStyle{
				Font:        "Helvetica Neue",
				FontSize:    "150",
				FontColor:   "1 1 1 1",
				Bold:        "1",
				Alignment:   "center",
				LineSpacing: "1.08",
			},
		},
	}
	nestedTitles = append(nestedTitles, tournamentHeaderTitle)
	laneCounter++

	// Add year headers (one for each 10-second segment)
	for i, year := range []int{1986, 1987, 1988, 1989, 1990, 1991, 1992, 1993, 1994, 1995, 1996, 1997, 1998, 1999, 2000, 2001, 2002, 2003, 2004} {
		yearOffset := fmt.Sprintf("%ds", i*10)
		yearDuration := "10s"
		yearHeaderTitle := Title{
			Ref:      "r3",
			Lane:     fmt.Sprintf("%d", laneCounter),
			Offset:   yearOffset,
			Name:     fmt.Sprintf("Year Header %d", year),
			Start:    "0s",
			Duration: yearDuration,
			Params: []Param{
				{Name: "Position", Key: "9999/10003/13260/3296672360/1/100/101", Value: fmt.Sprintf("%.0f %.0f", cellTextPositions[0][1].X*10, cellTextPositions[0][1].Y*10)},
			},
			Text: &TitleText{
				TextStyle: TextStyleRef{
					Ref:  fmt.Sprintf("year-header-style-%d", year),
					Text: fmt.Sprintf("%d", year),
				},
			},
			TextStyleDef: &TextStyleDef{
				ID: fmt.Sprintf("year-header-style-%d", year),
				TextStyle: TextStyle{
					Font:        "Helvetica Neue",
					FontSize:    "150",
					FontColor:   "0.5 0.8 1 1",
					Bold:        "1",
					Alignment:   "center",
					LineSpacing: "1.08",
				},
			},
		}
		nestedTitles = append(nestedTitles, yearHeaderTitle)
	}
	laneCounter++
	
	// Add static tournament names (always visible)
	tournaments := []string{"Australian Open", "French Open", "Wimbledon", "US Open"}
	for row, tournament := range tournaments {
		if row+1 < len(cellTextPositions) {
			tournamentStyleID := fmt.Sprintf("tournament-style-%d", row+1)
			tournamentTitle := Title{
				Ref:      "r3",
				Lane:     fmt.Sprintf("%d", laneCounter),
				Offset:   "0s",
				Name:     fmt.Sprintf("Tournament %d", row+1),
				Start:    "0s",
				Duration: FormatDurationForFCPXML(totalDuration),
				Params: []Param{
					{Name: "Position", Key: "9999/10003/13260/3296672360/1/100/101", Value: fmt.Sprintf("%.0f %.0f", cellTextPositions[row+1][0].X*10, cellTextPositions[row+1][0].Y*10)},
				},
				Text: &TitleText{
					TextStyle: TextStyleRef{
						Ref:  tournamentStyleID,
						Text: tournament,
					},
				},
				TextStyleDef: &TextStyleDef{
					ID: tournamentStyleID,
					TextStyle: TextStyle{
						Font:        "Helvetica Neue",
						FontSize:    "120",
						FontColor:   "0.9 0.9 0.9 1",
						Alignment:   "center",
						LineSpacing: "1.08",
					},
				},
			}
			nestedTitles = append(nestedTitles, tournamentTitle)
			laneCounter++
		}
	}

	// Add dynamic results data for each year (appearing for 10 seconds each)
	for i, year := range []int{1986, 1987, 1988, 1989, 1990, 1991, 1992, 1993, 1994, 1995, 1996, 1997, 1998, 1999, 2000, 2001, 2002, 2003, 2004} {
		yearOffset := fmt.Sprintf("%ds", i*10)
		yearDuration := "10s"
		
		if results, exists := yearsData[year]; exists {
			for row, result := range results {
				if row+1 < len(cellTextPositions) && len(cellTextPositions[row+1]) > 1 {
					cellStyleID := fmt.Sprintf("result-style-%d-%d", year, row+1)
					resultTitle := Title{
						Ref:      "r3",
						Lane:     fmt.Sprintf("%d", laneCounter),
						Offset:   yearOffset,
						Name:     fmt.Sprintf("Result %d-%d", year, row+1),
						Start:    "0s",
						Duration: yearDuration,
						Params: []Param{
							{Name: "Position", Key: "9999/10003/13260/3296672360/1/100/101", Value: fmt.Sprintf("%.0f %.0f", cellTextPositions[row+1][1].X*10, cellTextPositions[row+1][1].Y*10)},
						},
						Text: &TitleText{
							TextStyle: TextStyleRef{
								Ref:  cellStyleID,
								Text: result,
							},
						},
						TextStyleDef: &TextStyleDef{
							ID: cellStyleID,
							TextStyle: TextStyle{
								Font:        "Helvetica Neue",
								FontSize:    "120",
								FontColor:   getResultColor(result),
								Bold:        getBoldForResult(result),
								Alignment:   "center",
								LineSpacing: "1.08",
							},
						},
					}
					nestedTitles = append(nestedTitles, resultTitle)
					laneCounter++
				}
			}
		}
	}
	
	// Create the main spine video with all lines and text nested inside
	mainVideo := Video{
		Ref:      "r2",
		Offset:   "0s",
		Name:     "Table Grid Base",
		Start:    "0s",
		Duration: FormatDurationForFCPXML(totalDuration),
		Params: []Param{
			{Name: "Drop Shadow Opacity", Key: "9999/988455508/1/208/211", Value: "0"},
			{Name: "Feather", Key: "9999/988455508/988455699/2/353/102", Value: "0"},
			{Name: "Fill Color", Key: "9999/988455508/988455699/2/353/113/111", Value: "0 0 0"},
			{Name: "Shape", Key: "9999/988461322/100/988461395/2/100", Value: "4 (Rectangle)"},
			{Name: "Outline", Key: "9999/988461322/100/988464485/2/100", Value: "0"},
		},
		AdjustTransform: &AdjustTransform{Scale: "0 0"},  // Invisible base
		NestedVideos:    nestedVideos,
		NestedTitles:    nestedTitles,
	}
	
	var spineElements []interface{}
	spineElements = append(spineElements, mainVideo)
	fmt.Printf("DEBUG: Created ONE main video with %d nested horizontal lines and %d nested vertical lines\n", 
		len(horizontalPositionOffsets), len(verticalPositionOffsets))

	fmt.Printf("DEBUG: SUMMARY - Total spine elements created: %d\n", len(spineElements))
	
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
			Location: "file:///Users/aa/Movies/Untitled.fcpbundle/",
			Events: []Event{
				{
					Name: "Wikipedia Table",
					UID:  "54E7C4CB-8DAE-4E60-991A-DF2BA5646FF5",
					Projects: []Project{
						{
							Name:    "Wikipedia Table Reveal",
							UID:     "F36A6990-2D89-4815-8065-5EF5D0C71948",
							ModDate: "2025-06-07 08:38:19 -0700",
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

func getResultColor(result string) string {
	switch result {
	case "W":
		return "0.2 0.8 0.2 1" // Green for wins
	case "F":
		return "1 1 0.2 1" // Yellow for finals
	case "SF":
		return "1 0.6 0.2 1" // Orange for semifinals
	case "QF":
		return "0.6 0.8 1 1" // Light blue for quarterfinals
	case "4R", "3R", "2R", "1R":
		return "0.9 0.9 0.9 1" // Light gray for early rounds
	case "A":
		return "0.6 0.6 0.6 1" // Darker gray for absent
	case "NH":
		return "0.4 0.4 0.4 1" // Dark gray for not held
	default:
		return "1 1 1 1" // White for default
	}
}

func getBoldForResult(result string) string {
	switch result {
	case "W":
		return "1" // Bold for wins
	case "F":
		return "1" // Bold for finals
	default:
		return "0" // Normal weight for others
	}
}

func GenerateWikipediaTableFCPXML(tableData *TableData, outputPath string) error {
	return GenerateTableGridFCPXML(tableData, outputPath)
}
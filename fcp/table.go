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
	
	// Use default data if tableData is nil
	if tableData == nil {
		fmt.Printf("DEBUG: tableData is nil, using default data\n")
		tableData = &TableData{
			Headers: []string{"Tournament", "Result"},
			Rows: []TableRow{
				{Cells: []TableCell{{Content: "Grand Slam"}, {Content: "Champion"}}},
				{Cells: []TableCell{{Content: "Masters Cup"}, {Content: "Runner-up"}}},
			},
		}
	} else {
		fmt.Printf("DEBUG: tableData provided with %d headers and %d rows\n", len(tableData.Headers), len(tableData.Rows))
		fmt.Printf("DEBUG: Headers: %v\n", tableData.Headers)
	}

	totalDuration := 15 * time.Second
	
	// Calculate grid dimensions - limit for readability
	maxRows := min(3, len(tableData.Rows))     // Limit to 3 data rows for readability
	maxCols := min(3, len(tableData.Headers))  // Limit to 3 columns for readability
	totalRows := maxRows + 1  // Add 1 for header row
	
	fmt.Printf("DEBUG: Creating %dx%d table (including header)\n", totalRows, maxCols)
	
	// Use exact positioning values from LINES.md for perfect edge-to-edge coverage
	horizontalPositionOffsets := []float64{-100, -46.5928, 48.0135, 100}
	verticalPositionOffsets := []float64{-150, -73.3652, 73.3319, 150}
	
	// Trim to actual number of lines needed (rows+1 lines for rows, cols+1 lines for columns)
	if len(horizontalPositionOffsets) > totalRows+1 {
		horizontalPositionOffsets = horizontalPositionOffsets[:totalRows+1]
	}
	if len(verticalPositionOffsets) > maxCols+1 {
		verticalPositionOffsets = verticalPositionOffsets[:maxCols+1]
	}
	
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
				{Name: "Drop Shadow Opacity", Key: "9999/988455508/1/208/211", Value: "0.7426"},
				{Name: "Feather", Key: "9999/988455508/988455699/2/353/102", Value: "3"},
				{Name: "Fill Color", Key: "9999/988455508/988455699/2/353/113/111", Value: "1.0 0.0 0.0"},
				{Name: "Falloff", Key: "9999/988455508/988455699/2/353/158", Value: "-2"},
				{Name: "Shape", Key: "9999/988461322/100/988461395/2/100", Value: "4 (Rectangle)"},
				{Name: "Outline", Key: "9999/988461322/100/988464485/2/100", Value: "0"},
				{Name: "Outline Width", Key: "9999/988461322/100/988467855/2/100", Value: "0.338788"},
				{Name: "Corners", Key: "9999/988461322/100/988469428/2/100", Value: "1 (Square)"},
			},
			AdjustTransform: &AdjustTransform{Position: fmt.Sprintf("0 %.1f", yOffset), Scale: "1 0.0394"},
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
				{Name: "Drop Shadow Opacity", Key: "9999/988455508/1/208/211", Value: "0.7426"},
				{Name: "Feather", Key: "9999/988455508/988455699/2/353/102", Value: "3"},
				{Name: "Fill Color", Key: "9999/988455508/988455699/2/353/113/111", Value: "1.0 0.0 0.0"},
				{Name: "Falloff", Key: "9999/988455508/988455699/2/353/158", Value: "-2"},
				{Name: "Shape", Key: "9999/988461322/100/988461395/2/100", Value: "4 (Rectangle)"},
				{Name: "Outline", Key: "9999/988461322/100/988464485/2/100", Value: "0"},
				{Name: "Outline Width", Key: "9999/988461322/100/988467855/2/100", Value: "0.338788"},
				{Name: "Corners", Key: "9999/988461322/100/988469428/2/100", Value: "1 (Square)"},
			},
			AdjustTransform: &AdjustTransform{Position: fmt.Sprintf("%.1f 0", xOffset), Scale: "0.0394 1"},
		}
		nestedVideos = append(nestedVideos, verticalLine)
		laneCounter++
	}
	
	// Calculate cell positions for text placement
	// Position text in the center of each cell based on grid lines
	cellTextPositions := calculateCellTextPositions(horizontalPositionOffsets, verticalPositionOffsets)
	
	// Add header text
	for col := 0; col < maxCols && col < len(tableData.Headers); col++ {
		if col < len(cellTextPositions[0]) {
			headerTitle := Title{
				Ref:      "r3",
				Lane:     fmt.Sprintf("%d", laneCounter),
				Offset:   "0s",
				Name:     fmt.Sprintf("Header %d", col+1),
				Start:    "0s",
				Duration: FormatDurationForFCPXML(totalDuration),
				Params: []Param{
					{Name: "Text", Key: "9999/999166631/999166633/1/100/101", Value: tableData.Headers[col]},
					{Name: "Font", Key: "9999/999166631/999166633/2/360", Value: "Helvetica 24"},
					{Name: "Alignment", Key: "9999/999166631/999166633/2/354/999169573/401", Value: "1 (Center)"},
					{Name: "Line Spacing", Key: "9999/999166631/999166633/2/354/19", Value: "1.08"},
					{Name: "Tracking", Key: "9999/999166631/999166633/2/354/999169688/999169690/401", Value: "0"},
					{Name: "Face", Key: "9999/999166631/999166633/2/360/999169588/999169590/401", Value: "Bold"},
					{Name: "Position", Key: "9999/10003/13260/3296672360/1/100/101", Value: fmt.Sprintf("%.0f %.0f", cellTextPositions[0][col].X*10, cellTextPositions[0][col].Y*10)},
				},
			}
			nestedTitles = append(nestedTitles, headerTitle)
			laneCounter++
		}
	}
	
	// Add data cell text
	for row := 0; row < maxRows && row < len(tableData.Rows); row++ {
		for col := 0; col < maxCols && col < len(tableData.Rows[row].Cells); col++ {
			if row+1 < len(cellTextPositions) && col < len(cellTextPositions[row+1]) {
				cellContent := tableData.Rows[row].Cells[col].Content
				if cellContent != "" {
					dataTitle := Title{
						Ref:      "r3",
						Lane:     fmt.Sprintf("%d", laneCounter),
						Offset:   "0s",
						Name:     fmt.Sprintf("Cell %d-%d", row+1, col+1),
						Start:    "0s",
						Duration: FormatDurationForFCPXML(totalDuration),
						Params: []Param{
							{Name: "Text", Key: "9999/999166631/999166633/1/100/101", Value: cellContent},
							{Name: "Font", Key: "9999/999166631/999166633/2/360", Value: "Helvetica 20"},
							{Name: "Alignment", Key: "9999/999166631/999166633/2/354/999169573/401", Value: "1 (Center)"},
							{Name: "Line Spacing", Key: "9999/999166631/999166633/2/354/19", Value: "1.08"},
							{Name: "Tracking", Key: "9999/999166631/999166633/2/354/999169688/999169690/401", Value: "0"},
							{Name: "Face", Key: "9999/999166631/999166633/2/360/999169588/999169590/401", Value: "Regular"},
							{Name: "Position", Key: "9999/10003/13260/3296672360/1/100/101", Value: fmt.Sprintf("%.0f %.0f", cellTextPositions[row+1][col].X*10, cellTextPositions[row+1][col].Y*10)},
						},
					}
					nestedTitles = append(nestedTitles, dataTitle)
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

func GenerateWikipediaTableFCPXML(tableData *TableData, outputPath string) error {
	return GenerateTableGridFCPXML(tableData, outputPath)
}
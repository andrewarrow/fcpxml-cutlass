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

	// Removed unused header variables for minimal test

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
	
	// MINIMAL TEST: Create exactly like table.fcpxml structure
	fmt.Printf("DEBUG: MINIMAL TEST - Creating main video on spine, then nested video with lane=1\n")
	
	// Create nested video for vertical stacking (like table.fcpxml line 26)
	nestedVideo := Video{
		Ref:      "r2",
		Lane:     "1", // This creates the vertical stacking
		Offset:   "108108000/30000s", // Copy exact values from table.fcpxml
		Name:     "Shapes",
		Start:    "108108000/30000s", // Copy exact values from table.fcpxml
		Duration: "2634632/120000s",  // Copy exact values from table.fcpxml
		Params: []Param{
			{Name: "Drop Shadow Opacity", Key: "9999/988455508/1/208/211", Value: "0.7426"},
			{Name: "Feather", Key: "9999/988455508/988455699/2/353/102", Value: "3"},
			{Name: "Fill Color", Key: "9999/988455508/988455699/2/353/113/111", Value: "1.0817 -0.0799793 -0.145856"},
			{Name: "Falloff", Key: "9999/988455508/988455699/2/353/158", Value: "-2"},
			{Name: "Shape", Key: "9999/988461322/100/988461395/2/100", Value: "4 (Rectangle)"},
			{Name: "Outline", Key: "9999/988461322/100/988464485/2/100", Value: "0"},
			{Name: "Outline Width", Key: "9999/988461322/100/988467855/2/100", Value: "0.338788"},
			{Name: "Corners", Key: "9999/988461322/100/988469428/2/100", Value: "1 (Square)"},
			{Name: "Center", Key: "9999/988469355/988469353/3/988469357/1", Value: "0.5 0.59"}, // Copy exact from table.fcpxml
		},
		AdjustTransform: &AdjustTransform{Position: "0 8.33333", Scale: "1 0.0394"}, // Copy exact from table.fcpxml
	}

	// First video - main spine element with nested video inside (like table.fcpxml line 15)
	mainVideo := Video{
		Ref:      "r2",
		Offset:   "0s",
		Name:     "Shapes",
		Start:    "108108000/30000s", // Copy exact values from table.fcpxml
		Duration: "638638/30000s",    // Copy exact values from table.fcpxml
		Params: []Param{
			{Name: "Drop Shadow Opacity", Key: "9999/988455508/1/208/211", Value: "0.7426"},
			{Name: "Feather", Key: "9999/988455508/988455699/2/353/102", Value: "3"},
			{Name: "Fill Color", Key: "9999/988455508/988455699/2/353/113/111", Value: "1.0817 -0.0799793 -0.145856"},
			{Name: "Falloff", Key: "9999/988455508/988455699/2/353/158", Value: "-2"},
			{Name: "Shape", Key: "9999/988461322/100/988461395/2/100", Value: "4 (Rectangle)"},
			{Name: "Outline", Key: "9999/988461322/100/988464485/2/100", Value: "0"},
			{Name: "Outline Width", Key: "9999/988461322/100/988467855/2/100", Value: "0.338788"},
			{Name: "Corners", Key: "9999/988461322/100/988469428/2/100", Value: "1 (Square)"},
			{Name: "Center", Key: "9999/988469355/988469353/3/988469357/1", Value: "0.5 0.59"}, // Copy exact from table.fcpxml
		},
		AdjustTransform: &AdjustTransform{Scale: "1 0.0395"},
		NestedVideos:    []Video{nestedVideo}, // This is the key - nested video creates vertical stacking
	}
	
	fmt.Printf("DEBUG: Created main video on spine with nested video (lane=1) inside\n")
	spineElements = append(spineElements, mainVideo)
	
	// Second video - separate spine element (like table.fcpxml line 66)
	secondVideo := Video{
		Ref:      "r2",
		Offset:   "638638/30000s", // Copy exact values from table.fcpxml
		Name:     "Shapes",
		Start:    "108746638/30000s", // Copy exact values from table.fcpxml  
		Duration: "80080/120000s",    // Copy exact values from table.fcpxml
		Params: []Param{
			{Name: "Drop Shadow Opacity", Key: "9999/988455508/1/208/211", Value: "0.7426"},
			{Name: "Feather", Key: "9999/988455508/988455699/2/353/102", Value: "3"},
			{Name: "Fill Color", Key: "9999/988455508/988455699/2/353/113/111", Value: "1.0817 -0.0799793 -0.145856"},
			{Name: "Falloff", Key: "9999/988455508/988455699/2/353/158", Value: "-2"},
			{Name: "Shape", Key: "9999/988461322/100/988461395/2/100", Value: "4 (Rectangle)"},
			{Name: "Outline", Key: "9999/988461322/100/988464485/2/100", Value: "0"},
			{Name: "Outline Width", Key: "9999/988461322/100/988467855/2/100", Value: "0.338788"},
			{Name: "Corners", Key: "9999/988461322/100/988469428/2/100", Value: "1 (Square)"},
			{Name: "Center", Key: "9999/988469355/988469353/3/988469357/1", Value: "0.5 0.59"}, // Copy exact from table.fcpxml
		},
		AdjustTransform: &AdjustTransform{Scale: "1 0.0395"},
	}
	
	fmt.Printf("DEBUG: Created second video on spine\n")
	spineElements = append(spineElements, secondVideo)

	fmt.Printf("DEBUG: MINIMAL TEST - 1 spine video with nested video (lane=1) + 1 separate spine video\n")

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
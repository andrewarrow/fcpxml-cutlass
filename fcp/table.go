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
	
	// Use default data if tableData is nil
	if tableData == nil {
		tableData = &TableData{
			Headers: []string{"Date", "State(s)", "Magnitude", "Fatalities"},
			Rows: []TableRow{
				{Cells: []TableCell{{Content: "Example 1"}, {Content: "CA"}, {Content: "5.0"}, {Content: "0"}}},
				{Cells: []TableCell{{Content: "Example 2"}, {Content: "AK"}, {Content: "6.2"}, {Content: "1"}}},
			},
		}
	} else {
	}

	// Determine if we have time-based data by looking for year columns OR date data in cells
	var timeColumns []string
	var timeColumnIndices []int
	var staticColumns []string
	var staticColumnIndices []int
	
	// First check headers for 4-digit years (1900-2099)
	headerHasYears := false
	for i, header := range tableData.Headers {
		if len(header) == 4 && header >= "1900" && header <= "2099" {
			timeColumns = append(timeColumns, header)
			timeColumnIndices = append(timeColumnIndices, i)
			headerHasYears = true
		} else {
			staticColumns = append(staticColumns, header)
			staticColumnIndices = append(staticColumnIndices, i)
		}
	}
	
	// If no year headers, check if data contains date information that we can extract years from
	if !headerHasYears && len(tableData.Rows) > 0 {
		// Look for patterns like "{{Dts 1788 07 21}}" in the first column (Date column)
		dateColIndex := -1
		for i, header := range tableData.Headers {
			if strings.Contains(strings.ToLower(header), "date") {
				dateColIndex = i
				break
			}
		}
		
		if dateColIndex >= 0 {
			// Extract years from date entries
			yearSet := make(map[string]bool)
			for _, row := range tableData.Rows {
				if dateColIndex < len(row.Cells) {
					cellContent := row.Cells[dateColIndex].Content
					// Extract year from patterns like "1788-07-21", "{{Dts 1788 07 21}}", or "1788"
					if strings.Contains(cellContent, "-") {
						// Look for date pattern YYYY-MM-DD
						parts := strings.Split(cellContent, "-")
						if len(parts) >= 1 && len(parts[0]) == 4 && parts[0] >= "1500" && parts[0] <= "2100" {
							yearSet[parts[0]] = true
						}
					} else if strings.Contains(cellContent, "Dts") || strings.Contains(cellContent, "{{Dts") {
						parts := strings.Fields(cellContent)
						for _, part := range parts {
							if len(part) == 4 && part >= "1500" && part <= "2100" {
								yearSet[part] = true
								break
							}
						}
					} else if len(cellContent) == 4 && cellContent >= "1500" && cellContent <= "2100" {
						yearSet[cellContent] = true
					}
				}
			}
			
			// Convert year set to sorted list
			if len(yearSet) > 0 {
				for year := range yearSet {
					timeColumns = append(timeColumns, year)
				}
				// Sort years chronologically
				for i := 0; i < len(timeColumns)-1; i++ {
					for j := i + 1; j < len(timeColumns); j++ {
						if timeColumns[i] > timeColumns[j] {
							timeColumns[i], timeColumns[j] = timeColumns[j], timeColumns[i]
						}
					}
				}
				// Set up the date column as the time-based column
				timeColumnIndices = []int{dateColIndex}
				// Remove date column from static columns if it's there
				var newStaticColumns []string
				var newStaticColumnIndices []int
				for i, idx := range staticColumnIndices {
					if idx != dateColIndex {
						newStaticColumns = append(newStaticColumns, staticColumns[i])
						newStaticColumnIndices = append(newStaticColumnIndices, idx)
					}
				}
				staticColumns = newStaticColumns
				staticColumnIndices = newStaticColumnIndices
			}
		}
	}
	
	var totalDuration time.Duration
	if len(timeColumns) > 0 {
		// Time-based table: 3 seconds per time column
		totalDuration = time.Duration(len(timeColumns)*3) * time.Second
	} else {
		// Static table: 15 seconds total
		totalDuration = 15 * time.Second
	}
	
	// Calculate grid dimensions with FCP layer limits in mind
	// FCP has a practical limit of ~50-60 nested elements before performance issues
	// Each row+col creates multiple elements (lines + text), so limit conservatively
	const maxFCPRows = 5    // Maximum 5 data rows + 1 header = 6 total rows
	
	maxRows := min(maxFCPRows, len(tableData.Rows))     // Limit rows for FCP
	
	// For time-based tables: 2 columns (first static column + one time column)
	// For static tables: limit to 4 columns
	var maxCols int
	if len(timeColumns) > 0 {
		maxCols = 2  // Static column + one time column
	} else {
		maxCols = min(4, len(tableData.Headers))  // Regular static table
	}
	totalRows := maxRows + 1  // Add 1 for header row
	
	// Calculate and warn about element counts
	// totalHorizontalLines := totalRows + 1  // rows + 1 for borders
	// totalVerticalLines := maxCols + 1      // cols + 1 for borders  
	// totalLines := totalHorizontalLines + totalVerticalLines
	// totalTextElements := (maxCols) + (maxRows * maxCols) // headers + data cells
	// totalElements := totalLines + totalTextElements
	
	
	
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
	
	// Add static column headers
	if len(timeColumns) > 0 {
		// Time-based table: only show first static column header
		if len(staticColumns) > 0 && len(cellTextPositions[0]) > 0 {
			headerStyleID := "first-static-header-style"
			headerTitle := Title{
				Ref:      "r3",
				Lane:     fmt.Sprintf("%d", laneCounter),
				Offset:   "0s",
				Name:     "First Static Header",
				Start:    "0s",
				Duration: FormatDurationForFCPXML(totalDuration),
				Params: []Param{
					{Name: "Position", Key: "9999/10003/13260/3296672360/1/100/101", Value: fmt.Sprintf("%.0f %.0f", cellTextPositions[0][0].X*10, cellTextPositions[0][0].Y*10)},
				},
				Text: &TitleText{
					TextStyle: TextStyleRef{
						Ref:  headerStyleID,
						Text: staticColumns[0],
					},
				},
				TextStyleDef: &TextStyleDef{
					ID: headerStyleID,
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
			nestedTitles = append(nestedTitles, headerTitle)
			laneCounter++
		}
	} else {
		// Static table: show all headers
		for i, header := range staticColumns {
			if i < len(cellTextPositions[0]) && i < maxCols {
				headerStyleID := fmt.Sprintf("static-header-style-%d", i)
				headerTitle := Title{
					Ref:      "r3",
					Lane:     fmt.Sprintf("%d", laneCounter),
					Offset:   "0s",
					Name:     fmt.Sprintf("Static Header %d", i+1),
					Start:    "0s",
					Duration: FormatDurationForFCPXML(totalDuration),
					Params: []Param{
						{Name: "Position", Key: "9999/10003/13260/3296672360/1/100/101", Value: fmt.Sprintf("%.0f %.0f", cellTextPositions[0][i].X*10, cellTextPositions[0][i].Y*10)},
					},
					Text: &TitleText{
						TextStyle: TextStyleRef{
							Ref:  headerStyleID,
							Text: header,
						},
					},
					TextStyleDef: &TextStyleDef{
						ID: headerStyleID,
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
				nestedTitles = append(nestedTitles, headerTitle)
				laneCounter++
			}
		}
	}

	// Add time-based headers if any (one for each 3-second segment)
	if len(timeColumns) > 0 && len(cellTextPositions[0]) > 1 {
		for i, timeHeader := range timeColumns {
			timeOffset := FormatDurationForFCPXML(time.Duration(i*3) * time.Second)
			timeDuration := FormatDurationForFCPXML(3 * time.Second)
			timeHeaderTitle := Title{
				Ref:      "r3",
				Lane:     fmt.Sprintf("%d", laneCounter),
				Offset:   timeOffset,
				Name:     fmt.Sprintf("Time Header %s", timeHeader),
				Start:    "0s",
				Duration: timeDuration,
				Params: []Param{
					{Name: "Position", Key: "9999/10003/13260/3296672360/1/100/101", Value: fmt.Sprintf("%.0f %.0f", cellTextPositions[0][1].X*10, cellTextPositions[0][1].Y*10)},
				},
				Text: &TitleText{
					TextStyle: TextStyleRef{
						Ref:  fmt.Sprintf("time-header-style-%s", timeHeader),
						Text: timeHeader,
					},
				},
				TextStyleDef: &TextStyleDef{
					ID: fmt.Sprintf("time-header-style-%s", timeHeader),
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
			nestedTitles = append(nestedTitles, timeHeaderTitle)
		}
		laneCounter++
	}
	
	// Add static data cells
	if len(timeColumns) > 0 {
		// Time-based table: only show first static column data
		if len(staticColumnIndices) > 0 {
			firstStaticColIndex := staticColumnIndices[0]
			for row := 0; row < maxRows && row < len(tableData.Rows); row++ {
				if firstStaticColIndex < len(tableData.Rows[row].Cells) && row+1 < len(cellTextPositions) && len(cellTextPositions[row+1]) > 0 {
					cellContent := tableData.Rows[row].Cells[firstStaticColIndex].Content
					if cellContent != "" {
						cellStyleID := fmt.Sprintf("first-static-cell-style-%d", row+1)
						staticCellTitle := Title{
							Ref:      "r3",
							Lane:     fmt.Sprintf("%d", laneCounter),
							Offset:   "0s",
							Name:     fmt.Sprintf("First Static Cell %d", row+1),
							Start:    "0s",
							Duration: FormatDurationForFCPXML(totalDuration),
							Params: []Param{
								{Name: "Position", Key: "9999/10003/13260/3296672360/1/100/101", Value: fmt.Sprintf("%.0f %.0f", cellTextPositions[row+1][0].X*10, cellTextPositions[row+1][0].Y*10)},
							},
							Text: &TitleText{
								TextStyle: TextStyleRef{
									Ref:  cellStyleID,
									Text: cellContent,
								},
							},
							TextStyleDef: &TextStyleDef{
								ID: cellStyleID,
								TextStyle: TextStyle{
									Font:        "Helvetica Neue",
									FontSize:    "120",
									FontColor:   "0.9 0.9 0.9 1",
									Alignment:   "center",
									LineSpacing: "1.08",
								},
							},
						}
						nestedTitles = append(nestedTitles, staticCellTitle)
						laneCounter++
					}
				}
			}
		}
	} else {
		// Static table: show all static data cells
		for row := 0; row < maxRows && row < len(tableData.Rows); row++ {
			for colIdx, staticColIndex := range staticColumnIndices {
				if colIdx < maxCols && staticColIndex < len(tableData.Rows[row].Cells) && row+1 < len(cellTextPositions) && colIdx < len(cellTextPositions[row+1]) {
					cellContent := tableData.Rows[row].Cells[staticColIndex].Content
					if cellContent != "" {
						cellStyleID := fmt.Sprintf("static-cell-style-%d-%d", row+1, colIdx+1)
						staticCellTitle := Title{
							Ref:      "r3",
							Lane:     fmt.Sprintf("%d", laneCounter),
							Offset:   "0s",
							Name:     fmt.Sprintf("Static Cell %d-%d", row+1, colIdx+1),
							Start:    "0s",
							Duration: FormatDurationForFCPXML(totalDuration),
							Params: []Param{
								{Name: "Position", Key: "9999/10003/13260/3296672360/1/100/101", Value: fmt.Sprintf("%.0f %.0f", cellTextPositions[row+1][colIdx].X*10, cellTextPositions[row+1][colIdx].Y*10)},
							},
							Text: &TitleText{
								TextStyle: TextStyleRef{
									Ref:  cellStyleID,
									Text: cellContent,
								},
							},
							TextStyleDef: &TextStyleDef{
								ID: cellStyleID,
								TextStyle: TextStyle{
									Font:        "Helvetica Neue",
									FontSize:    "120",
									FontColor:   "0.9 0.9 0.9 1",
									Alignment:   "center",
									LineSpacing: "1.08",
								},
							},
						}
						nestedTitles = append(nestedTitles, staticCellTitle)
						laneCounter++
					}
				}
			}
		}
	}

	// Add dynamic time-based data (appearing for 3 seconds each)
	if len(timeColumns) > 0 && len(cellTextPositions[0]) > 1 {
		for i, timeHeader := range timeColumns {
			timeOffset := FormatDurationForFCPXML(time.Duration(i*3) * time.Second)
			timeDuration := FormatDurationForFCPXML(3 * time.Second)
			
			if headerHasYears {
				// Original logic for year-based headers (like tennis data)
				timeColIndex := -1
				for idx, colIndex := range timeColumnIndices {
					if idx == i {
						timeColIndex = colIndex
						break
					}
				}
				
				if timeColIndex >= 0 {
					for row := 0; row < maxRows && row < len(tableData.Rows); row++ {
						if timeColIndex < len(tableData.Rows[row].Cells) && row+1 < len(cellTextPositions) && len(cellTextPositions[row+1]) > 1 {
							cellContent := tableData.Rows[row].Cells[timeColIndex].Content
							if cellContent != "" {
								cellStyleID := fmt.Sprintf("time-cell-style-%s-%d", timeHeader, row+1)
								timeCellTitle := Title{
									Ref:      "r3",
									Lane:     fmt.Sprintf("%d", laneCounter),
									Offset:   timeOffset,
									Name:     fmt.Sprintf("Time Cell %s-%d", timeHeader, row+1),
									Start:    "0s",
									Duration: timeDuration,
									Params: []Param{
										{Name: "Position", Key: "9999/10003/13260/3296672360/1/100/101", Value: fmt.Sprintf("%.0f %.0f", cellTextPositions[row+1][1].X*10, cellTextPositions[row+1][1].Y*10)},
									},
									Text: &TitleText{
										TextStyle: TextStyleRef{
											Ref:  cellStyleID,
											Text: cellContent,
										},
									},
									TextStyleDef: &TextStyleDef{
										ID: cellStyleID,
										TextStyle: TextStyle{
											Font:        "Helvetica Neue",
											FontSize:    "120",
											FontColor:   getResultColor(cellContent),
											Bold:        getBoldForResult(cellContent),
											Alignment:   "center",
											LineSpacing: "1.08",
										},
									},
								}
								nestedTitles = append(nestedTitles, timeCellTitle)
								laneCounter++
							}
						}
					}
				}
			} else {
				// New logic for date-based data (like earthquake data)
				// Show only rows that match the current year
				dateColIndex := timeColumnIndices[0]
				rowCounter := 0
				
				for row := 0; row < len(tableData.Rows); row++ {
					if dateColIndex < len(tableData.Rows[row].Cells) {
						dateCellContent := tableData.Rows[row].Cells[dateColIndex].Content
						// Check if this row's date contains the current year
						if strings.Contains(dateCellContent, timeHeader) {
							if rowCounter < maxRows && rowCounter+1 < len(cellTextPositions) {
								// Show all columns for this matching row
								for colIdx, staticColIndex := range staticColumnIndices {
									if colIdx < len(cellTextPositions[rowCounter+1]) && staticColIndex < len(tableData.Rows[row].Cells) {
										cellContent := tableData.Rows[row].Cells[staticColIndex].Content
										if cellContent != "" {
											cellStyleID := fmt.Sprintf("time-data-style-%s-%d-%d", timeHeader, rowCounter+1, colIdx+1)
											timeCellTitle := Title{
												Ref:      "r3",
												Lane:     fmt.Sprintf("%d", laneCounter),
												Offset:   timeOffset,
												Name:     fmt.Sprintf("Time Data %s-R%d-C%d", timeHeader, rowCounter+1, colIdx+1),
												Start:    "0s",
												Duration: timeDuration,
												Params: []Param{
													{Name: "Position", Key: "9999/10003/13260/3296672360/1/100/101", Value: fmt.Sprintf("%.0f %.0f", cellTextPositions[rowCounter+1][colIdx].X*10, cellTextPositions[rowCounter+1][colIdx].Y*10)},
												},
												Text: &TitleText{
													TextStyle: TextStyleRef{
														Ref:  cellStyleID,
														Text: cellContent,
													},
												},
												TextStyleDef: &TextStyleDef{
													ID: cellStyleID,
													TextStyle: TextStyle{
														Font:        "Helvetica Neue",
														FontSize:    "120",
														FontColor:   "0.9 0.9 0.9 1",
														Alignment:   "center",
														LineSpacing: "1.08",
													},
												},
											}
											nestedTitles = append(nestedTitles, timeCellTitle)
											laneCounter++
										}
									}
								}
								rowCounter++
							}
						}
					}
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
	
	spineContent := buildSpineContent(spineElements)

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

// WikiSimpleTable represents a simple table structure (matching wikipedia.SimpleTable)
type WikiSimpleTable struct {
	Headers []string
	Rows    [][]string
}

// GenerateMultiTableFCPXML creates FCPXML with multiple table views showing sequentially for 3 seconds each
func GenerateMultiTableFCPXML(table *WikiSimpleTable, outputPath string) error {
	if table == nil || len(table.Headers) == 0 {
		return fmt.Errorf("no table data provided")
	}

	// Determine table views to show (same logic as ASCII display)
	var tableViews []TableView
	
	// Detect table type: Traditional vs Tennis-style
	isTraditionalTable := detectTraditionalTable(table)
	
	if isTraditionalTable {
		// Traditional table: Each row as a separate view for 3 seconds
		for rowIndex, row := range table.Rows {
			view := TableView{
				Title:    fmt.Sprintf("Row %d/%d", rowIndex+1, len(table.Rows)),
				Duration: 3 * time.Second,
				Headers:  []string{"Field", "Value"},
				Data:     make([][]string, len(table.Headers)),
			}
			
			// Convert row to field-value pairs
			for i, header := range table.Headers {
				value := ""
				if i < len(row) {
					value = row[i]
				}
				view.Data[i] = []string{header, value}
			}
			
			tableViews = append(tableViews, view)
		}
	} else {
		// Tennis-style: Leftmost column + each data column (skipping leftmost)
		leftColIndex := 0
		
		for dataColIndex := 1; dataColIndex < len(table.Headers); dataColIndex++ {
			view := TableView{
				Title:    fmt.Sprintf("Table %d/%d: %s + %s", dataColIndex, len(table.Headers)-1, table.Headers[leftColIndex], table.Headers[dataColIndex]),
				Duration: 3 * time.Second,
				Headers:  []string{table.Headers[leftColIndex], table.Headers[dataColIndex]},
				Data:     make([][]string, len(table.Rows)),
			}
			
			// Extract two-column data
			for rowIndex, row := range table.Rows {
				leftValue := ""
				dataValue := ""
				
				if leftColIndex < len(row) {
					leftValue = row[leftColIndex]
				}
				if dataColIndex < len(row) {
					dataValue = row[dataColIndex]
				}
				
				view.Data[rowIndex] = []string{leftValue, dataValue}
			}
			
			tableViews = append(tableViews, view)
		}
	}

	// Generate FCPXML with sequential table views
	return generateSequentialTableFCPXML(tableViews, outputPath)
}

// TableView represents a single table view to display
type TableView struct {
	Title    string
	Duration time.Duration
	Headers  []string
	Data     [][]string // Each row is a slice of cell values
}

// detectTraditionalTable determines if a table should be displayed in traditional format
func detectTraditionalTable(table *WikiSimpleTable) bool {
	if table == nil || len(table.Headers) < 3 {
		return false
	}
	
	// Check if headers contain year patterns (tennis-style indicator)
	yearCount := 0
	for _, header := range table.Headers[1:] { // Skip first header
		// Check for 4-digit years
		if len(header) == 4 && header >= "1900" && header <= "2100" {
			yearCount++
		}
		// Check for year ranges like "2010-2020"
		if strings.Contains(header, "-") && len(header) >= 4 {
			parts := strings.Split(header, "-")
			if len(parts) == 2 && len(parts[0]) == 4 && parts[0] >= "1900" {
				yearCount++
			}
		}
	}
	
	// If more than half the columns are years, it's likely tennis-style
	if yearCount > len(table.Headers)/2 {
		return false
	}
	
	// Check for traditional table indicators
	headerLower := strings.ToLower(strings.Join(table.Headers, " "))
	traditionalKeywords := []string{
		"date", "state", "magnitude", "location", "name", "type", 
		"fatalities", "casualties", "article", "description", "result",
	}
	
	matchCount := 0
	for _, keyword := range traditionalKeywords {
		if strings.Contains(headerLower, keyword) {
			matchCount++
		}
	}
	
	// If we have traditional keywords and few/no years, it's traditional
	return matchCount >= 2
}

// generateSequentialTableFCPXML creates FCPXML with multiple table views showing sequentially
func generateSequentialTableFCPXML(tableViews []TableView, outputPath string) error {
	if len(tableViews) == 0 {
		return fmt.Errorf("no table views to generate")
	}

	// Calculate total duration
	totalDuration := time.Duration(0)
	for _, view := range tableViews {
		totalDuration += view.Duration
	}

	// Create spine elements for each table view
	var spineElements []interface{}
	currentOffset := time.Duration(0)
	
	for i, view := range tableViews {
		// Create a table grid for this view
		tableData := &TableData{
			Headers: view.Headers,
			Rows:    make([]TableRow, len(view.Data)),
		}
		
		// Convert data to TableRow format
		for j, rowData := range view.Data {
			tableData.Rows[j] = TableRow{
				Cells: make([]TableCell, len(rowData)),
			}
			for k, cellValue := range rowData {
				tableData.Rows[j].Cells[k] = TableCell{
					Content: cellValue,
				}
			}
		}
		
		// Generate table elements for this view
		tableVideo := createTableVideoForView(tableData, view, currentOffset, i+1)
		spineElements = append(spineElements, tableVideo)
		
		currentOffset += view.Duration
	}

	// Build spine content
	spineContent := buildSpineContent(spineElements)

	// Create FCPXML structure
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
					Name: "Wikipedia Tables",
					UID:  "54E7C4CB-8DAE-4E60-991A-DF2BA5646FF5",
					Projects: []Project{
						{
							Name:    fmt.Sprintf("Wikipedia Tables (%d views)", len(tableViews)),
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

	// Write FCPXML
	output, err := xml.MarshalIndent(fcpxml, "", "    ")
	if err != nil {
		return err
	}

	xmlContent := xml.Header + "<!DOCTYPE fcpxml>\n" + string(output)
	return os.WriteFile(outputPath, []byte(xmlContent), 0644)
}

// createTableVideoForView creates a video element containing a complete table view
func createTableVideoForView(tableData *TableData, view TableView, offset time.Duration, viewNumber int) Video {
	// Calculate grid dimensions
	maxRows := min(5, len(tableData.Rows))  // Limit rows for FCP
	maxCols := len(tableData.Headers)       // Use actual column count for 2-column tables
	totalRows := maxRows + 1                // Add 1 for header row
	
	// Create grid lines
	var nestedVideos []Video
	var nestedTitles []Title
	laneCounter := 1
	
	// Calculate grid positions
	horizontalPositionOffsets := make([]float64, totalRows+1)
	startY := -100.0
	endY := 100.0
	stepY := (endY - startY) / float64(totalRows)
	for i := 0; i <= totalRows; i++ {
		horizontalPositionOffsets[i] = startY + float64(i)*stepY
	}
	
	verticalPositionOffsets := make([]float64, maxCols+1)
	startX := -150.0
	endX := 150.0
	stepX := (endX - startX) / float64(maxCols)
	for i := 0; i <= maxCols; i++ {
		verticalPositionOffsets[i] = startX + float64(i)*stepX
	}
	
	// Add horizontal lines
	for i, yOffset := range horizontalPositionOffsets {
		horizontalLine := Video{
			Ref:      "r2",
			Lane:     fmt.Sprintf("%d", laneCounter),
			Offset:   "0s",
			Name:     fmt.Sprintf("H-Line %d-V%d", i+1, viewNumber),
			Start:    "0s",
			Duration: FormatDurationForFCPXML(view.Duration),
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
	
	// Add vertical lines
	for j, xOffset := range verticalPositionOffsets {
		verticalLine := Video{
			Ref:      "r2",
			Lane:     fmt.Sprintf("%d", laneCounter),
			Offset:   "0s",
			Name:     fmt.Sprintf("V-Line %d-V%d", j+1, viewNumber),
			Start:    "0s",
			Duration: FormatDurationForFCPXML(view.Duration),
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
	
	// Calculate cell positions
	cellTextPositions := calculateCellTextPositions(horizontalPositionOffsets, verticalPositionOffsets)
	
	// Add headers
	for i, header := range tableData.Headers {
		if i < len(cellTextPositions[0]) {
			headerStyleID := fmt.Sprintf("header-style-v%d-%d", viewNumber, i)
			headerTitle := Title{
				Ref:      "r3",
				Lane:     fmt.Sprintf("%d", laneCounter),
				Offset:   "0s",
				Name:     fmt.Sprintf("Header %s V%d", header, viewNumber),
				Start:    "0s",
				Duration: FormatDurationForFCPXML(view.Duration),
				Params: []Param{
					{Name: "Position", Key: "9999/10003/13260/3296672360/1/100/101", Value: fmt.Sprintf("%.0f %.0f", cellTextPositions[0][i].X*10, cellTextPositions[0][i].Y*10)},
				},
				Text: &TitleText{
					TextStyle: TextStyleRef{
						Ref:  headerStyleID,
						Text: header,
					},
				},
				TextStyleDef: &TextStyleDef{
					ID: headerStyleID,
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
			nestedTitles = append(nestedTitles, headerTitle)
			laneCounter++
		}
	}
	
	// Add data cells
	for row := 0; row < maxRows && row < len(tableData.Rows); row++ {
		for col := 0; col < len(tableData.Rows[row].Cells) && col < len(cellTextPositions[row+1]); col++ {
			cellContent := tableData.Rows[row].Cells[col].Content
			if cellContent != "" {
				cellStyleID := fmt.Sprintf("cell-style-v%d-%d-%d", viewNumber, row+1, col+1)
				cellTitle := Title{
					Ref:      "r3",
					Lane:     fmt.Sprintf("%d", laneCounter),
					Offset:   "0s",
					Name:     fmt.Sprintf("Cell V%d-R%d-C%d", viewNumber, row+1, col+1),
					Start:    "0s",
					Duration: FormatDurationForFCPXML(view.Duration),
					Params: []Param{
						{Name: "Position", Key: "9999/10003/13260/3296672360/1/100/101", Value: fmt.Sprintf("%.0f %.0f", cellTextPositions[row+1][col].X*10, cellTextPositions[row+1][col].Y*10)},
					},
					Text: &TitleText{
						TextStyle: TextStyleRef{
							Ref:  cellStyleID,
							Text: cellContent,
						},
					},
					TextStyleDef: &TextStyleDef{
						ID: cellStyleID,
						TextStyle: TextStyle{
							Font:        "Helvetica Neue",
							FontSize:    "120",
							FontColor:   "0.9 0.9 0.9 1",
							Alignment:   "center",
							LineSpacing: "1.08",
						},
					},
				}
				nestedTitles = append(nestedTitles, cellTitle)
				laneCounter++
			}
		}
	}
	
	// Create main video with all nested elements
	return Video{
		Ref:      "r2",
		Offset:   FormatDurationForFCPXML(offset),
		Name:     fmt.Sprintf("Table View %d: %s", viewNumber, view.Title),
		Start:    "0s",
		Duration: FormatDurationForFCPXML(view.Duration),
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
}
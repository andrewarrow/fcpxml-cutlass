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

// TableStyle defines the behavior of columns in the table animation
type TableStyle int

const (
	StaticLeftAnimatedRight TableStyle = iota // Tennis style: left column static, right columns animated
	AllColumnsAnimated                       // Traditional style: all columns animated together
)

// TableConfig contains configuration for table generation
type TableConfig struct {
	Style           TableStyle
	StaticColumns   []int // Indices of columns that remain static
	AnimatedColumns []int // Indices of columns that animate
	TimeSegments    []string // Time segments for animation (years, dates, etc.)
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
	}

	// Determine table configuration based on data structure
	config := analyzeTableStructure(tableData)
	
	return generateTableWithConfig(tableData, outputPath, config)
}

// analyzeTableStructure determines the appropriate table style and configuration
func analyzeTableStructure(tableData *TableData) TableConfig {
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
	
	// Determine table style based on structure
	if headerHasYears {
		// Tennis style: static left column, animated right columns
		return TableConfig{
			Style:           StaticLeftAnimatedRight,
			StaticColumns:   staticColumnIndices,
			AnimatedColumns: timeColumnIndices,
			TimeSegments:    timeColumns,
		}
	} else if len(timeColumns) > 0 {
		// Earthquake style: first column static, remaining columns animated
		return TableConfig{
			Style:           StaticLeftAnimatedRight,
			StaticColumns:   []int{0}, // First column static
			AnimatedColumns: staticColumnIndices[1:], // Remaining columns animated
			TimeSegments:    timeColumns,
		}
	} else {
		// Traditional static table
		return TableConfig{
			Style:           AllColumnsAnimated,
			StaticColumns:   staticColumnIndices,
			AnimatedColumns: []int{},
			TimeSegments:    []string{},
		}
	}
}

func generateTableWithConfig(tableData *TableData, outputPath string, config TableConfig) error {
	
	var totalDuration time.Duration
	if len(config.TimeSegments) > 0 {
		// Time-based table: 3 seconds per time segment
		totalDuration = time.Duration(len(config.TimeSegments)*3) * time.Second
	} else {
		// Static table: 15 seconds total
		totalDuration = 15 * time.Second
	}
	
	// Calculate grid dimensions with FCP layer limits in mind
	// FCP has a practical limit of ~50-60 nested elements before performance issues
	// Each row+col creates multiple elements (lines + text), so limit conservatively
	const maxFCPRows = 5    // Maximum 5 data rows + 1 header = 6 total rows
	
	maxRows := min(maxFCPRows, len(tableData.Rows))     // Limit rows for FCP
	
	// Determine column count based on table style
	var maxCols int
	switch config.Style {
	case StaticLeftAnimatedRight:
		maxCols = 2  // Static column + one animated column
	case AllColumnsAnimated:
		maxCols = min(4, len(tableData.Headers))  // Regular static table
	default:
		maxCols = min(4, len(tableData.Headers))
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
	startY := 100.0  // Start from top (positive Y in FCP)
	endY := -100.0   // End at bottom (negative Y in FCP)
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
	
	// Add static column headers based on configuration
	switch config.Style {
	case StaticLeftAnimatedRight:
		// Static-left style: only show first static column header
		if len(config.StaticColumns) > 0 && len(cellTextPositions[0]) > 0 {
			staticColIndex := config.StaticColumns[0]
			if staticColIndex < len(tableData.Headers) {
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
							Text: tableData.Headers[staticColIndex],
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
	case AllColumnsAnimated:
		// Static table: show all headers (for traditional tables like earthquakes)
		for i := 0; i < len(tableData.Headers) && i < maxCols && i < len(cellTextPositions[0]); i++ {
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
						Text: tableData.Headers[i],
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

	// Add time-based headers if any (one for each 3-second segment)
	if config.Style == StaticLeftAnimatedRight && len(config.TimeSegments) > 0 && len(cellTextPositions[0]) > 1 {
		for i, timeHeader := range config.TimeSegments {
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
					{Name: "Opacity", Key: "9999/10003/1/100/101", Value: "0", KeyframeAnimation: &KeyframeAnimation{
						Keyframes: []Keyframe{
							{Time: "0s", Value: "0"},
							{Time: "15/30000s", Value: "1"},
							{Time: "75/30000s", Value: "1"},
							{Time: "90/30000s", Value: "0"},
						},
					}},
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
	
	// Add static data cells based on configuration
	switch config.Style {
	case StaticLeftAnimatedRight:
		// Static-left style: only show first static column data
		if len(config.StaticColumns) > 0 {
			firstStaticColIndex := config.StaticColumns[0]
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
	case AllColumnsAnimated:
		// For AllColumnsAnimated (traditional tables), don't show static data cells
		// All data will be shown via animation in generateTraditionalAnimatedData
	}

	// Add dynamic time-based data for appropriate table styles
	if config.Style == StaticLeftAnimatedRight && len(config.TimeSegments) > 0 && len(cellTextPositions[0]) > 1 {
		err := generateAnimatedData(tableData, config, cellTextPositions, &nestedTitles, &laneCounter, maxRows)
		if err != nil {
			return err
		}
	} else if config.Style == AllColumnsAnimated && len(config.TimeSegments) > 0 {
		// For traditional tables with time segments (like earthquakes), show each row sequentially
		err := generateTraditionalAnimatedData(tableData, config, cellTextPositions, &nestedTitles, &laneCounter, maxRows, maxCols)
		if err != nil {
			return err
		}
	}
	
	return generateFinalFCPXML(tableData, outputPath, totalDuration, nestedVideos, nestedTitles)
}

// generateAnimatedData handles the animated column data for StaticLeftAnimatedRight style
func generateAnimatedData(tableData *TableData, config TableConfig, cellTextPositions [][]Position, nestedTitles *[]Title, laneCounter *int, maxRows int) error {
	for i, timeHeader := range config.TimeSegments {
		timeOffset := FormatDurationForFCPXML(time.Duration(i*3) * time.Second)
		timeDuration := FormatDurationForFCPXML(3 * time.Second)
		
		// Check if we're dealing with year headers (tennis style) or date-based data (earthquake style)
		if len(config.AnimatedColumns) > 0 && config.AnimatedColumns[0] > 0 {
			// Tennis style: year headers as columns
			timeColIndex := -1
			if i < len(config.AnimatedColumns) {
				timeColIndex = config.AnimatedColumns[i]
			}
			
			if timeColIndex >= 0 {
				for row := 0; row < maxRows && row < len(tableData.Rows); row++ {
					// Reverse the row index for data access to match ASCII order
					reversedRow := maxRows - 1 - row
					if reversedRow >= 0 && reversedRow < len(tableData.Rows) && timeColIndex < len(tableData.Rows[reversedRow].Cells) && row+1 < len(cellTextPositions) && len(cellTextPositions[row+1]) > 1 {
						cellContent := tableData.Rows[reversedRow].Cells[timeColIndex].Content
						if cellContent != "" {
							cellStyleID := fmt.Sprintf("time-cell-style-%s-%d", timeHeader, reversedRow+1)
							timeCellTitle := Title{
								Ref:      "r3",
								Lane:     fmt.Sprintf("%d", *laneCounter),
								Offset:   timeOffset,
								Name:     fmt.Sprintf("Time Cell %s-%d", timeHeader, reversedRow+1),
								Start:    "0s",
								Duration: timeDuration,
								Params: []Param{
									{Name: "Position", Key: "9999/10003/13260/3296672360/1/100/101", Value: fmt.Sprintf("%.0f %.0f", cellTextPositions[row+1][1].X*10, cellTextPositions[row+1][1].Y*10)},
									{Name: "Opacity", Key: "9999/10003/1/100/101", Value: "0", KeyframeAnimation: &KeyframeAnimation{
										Keyframes: []Keyframe{
											{Time: "0s", Value: "0"},
											{Time: "15/30000s", Value: "1"},
											{Time: "75/30000s", Value: "1"},
											{Time: "90/30000s", Value: "0"},
										},
									}},
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
							*nestedTitles = append(*nestedTitles, timeCellTitle)
							(*laneCounter)++
						}
					}
				}
			}
		} else {
			// Earthquake style: date-based data filtering
			// Show only rows that match the current year
			dateColIndex := 0 // First column is typically the date column
			rowCounter := 0
			
			for row := 0; row < len(tableData.Rows); row++ {
				if dateColIndex < len(tableData.Rows[row].Cells) {
					dateCellContent := tableData.Rows[row].Cells[dateColIndex].Content
					// Check if this row's date contains the current year
					if strings.Contains(dateCellContent, timeHeader) {
						if rowCounter < maxRows && rowCounter+1 < len(cellTextPositions) {
							// Show animated columns for this matching row (all except first static column)
							for colIdx := 1; colIdx < len(tableData.Headers) && colIdx < len(cellTextPositions[rowCounter+1]); colIdx++ {
								if colIdx < len(tableData.Rows[row].Cells) {
									cellContent := tableData.Rows[row].Cells[colIdx].Content
									if cellContent != "" {
										cellStyleID := fmt.Sprintf("time-data-style-%s-%d-%d", timeHeader, rowCounter+1, colIdx+1)
										// All columns except the first (static) column get fade animation
										params := []Param{
											{Name: "Position", Key: "9999/10003/13260/3296672360/1/100/101", Value: fmt.Sprintf("%.0f %.0f", cellTextPositions[rowCounter+1][1].X*10, cellTextPositions[rowCounter+1][1].Y*10)},
											{Name: "Opacity", Key: "9999/10003/1/100/101", Value: "0", KeyframeAnimation: &KeyframeAnimation{
												Keyframes: []Keyframe{
													{Time: "0s", Value: "0"},
													{Time: "15/30000s", Value: "1"},
													{Time: "75/30000s", Value: "1"},
													{Time: "90/30000s", Value: "0"},
												},
											}},
										}
										
										timeCellTitle := Title{
											Ref:      "r3",
											Lane:     fmt.Sprintf("%d", *laneCounter),
											Offset:   timeOffset,
											Name:     fmt.Sprintf("Time Data %s-R%d-C%d", timeHeader, rowCounter+1, colIdx+1),
											Start:    "0s",
											Duration: timeDuration,
											Params:   params,
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
										*nestedTitles = append(*nestedTitles, timeCellTitle)
										(*laneCounter)++
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
	return nil
}

// generateTraditionalAnimatedData handles animated data for traditional tables (like earthquakes)
// Shows each complete row for 3 seconds with all columns visible
func generateTraditionalAnimatedData(tableData *TableData, config TableConfig, cellTextPositions [][]Position, nestedTitles *[]Title, laneCounter *int, maxRows, maxCols int) error {
	for i, timeHeader := range config.TimeSegments {
		timeOffset := FormatDurationForFCPXML(time.Duration(i*3) * time.Second)
		timeDuration := FormatDurationForFCPXML(3 * time.Second)
		
		// Find the row that corresponds to this time segment (year)
		rowIndex := -1
		for r, row := range tableData.Rows {
			if r < len(tableData.Rows) && len(row.Cells) > 0 {
				// Check if the first cell (date) contains this year
				dateContent := row.Cells[0].Content
				if strings.Contains(dateContent, timeHeader) {
					rowIndex = r
					break
				}
			}
		}
		
		if rowIndex >= 0 && rowIndex < len(tableData.Rows) {
			// Display all columns for this row in the first data row position (traditional table style)
			for colIdx := 0; colIdx < len(tableData.Headers) && colIdx < maxCols; colIdx++ {
				if colIdx < len(tableData.Rows[rowIndex].Cells) && len(cellTextPositions) > 1 && colIdx < len(cellTextPositions[1]) {
					cellContent := tableData.Rows[rowIndex].Cells[colIdx].Content
					if cellContent != "" {
						cellStyleID := fmt.Sprintf("traditional-cell-style-%s-%d", timeHeader, colIdx)
						traditionalCellTitle := Title{
							Ref:      "r3",
							Lane:     fmt.Sprintf("%d", *laneCounter),
							Offset:   timeOffset,
							Name:     fmt.Sprintf("Traditional Cell %s-C%d", timeHeader, colIdx),
							Start:    "0s",
							Duration: timeDuration,
							Params: []Param{
								{Name: "Position", Key: "9999/10003/13260/3296672360/1/100/101", Value: fmt.Sprintf("%.0f %.0f", cellTextPositions[1][colIdx].X*10, cellTextPositions[1][colIdx].Y*10)},
								{Name: "Opacity", Key: "9999/10003/1/100/101", Value: "0", KeyframeAnimation: &KeyframeAnimation{
									Keyframes: []Keyframe{
										{Time: "0s", Value: "0"},
										{Time: "15/30000s", Value: "1"},
										{Time: "75/30000s", Value: "1"},
										{Time: "90/30000s", Value: "0"},
									},
								}},
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
						*nestedTitles = append(*nestedTitles, traditionalCellTitle)
						(*laneCounter)++
					}
				}
			}
		}
	}
	return nil
}

// generateFinalFCPXML creates the final FCPXML structure and writes it to file
func generateFinalFCPXML(tableData *TableData, outputPath string, totalDuration time.Duration, nestedVideos []Video, nestedTitles []Title) error {
	
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

// GenerateMultiTableFCPXML creates FCPXML with multiple table views - now uses unified system
func GenerateMultiTableFCPXML(table *WikiSimpleTable, outputPath string) error {
	if table == nil || len(table.Headers) == 0 {
		return fmt.Errorf("no table data provided")
	}

	// Convert WikiSimpleTable to TableData format for unified system
	tableData := &TableData{
		Headers: table.Headers,
		Rows:    make([]TableRow, len(table.Rows)),
	}

	for i, row := range table.Rows {
		tableData.Rows[i] = TableRow{
			Cells: make([]TableCell, len(row)),
		}
		for j, cell := range row {
			tableData.Rows[i].Cells[j] = TableCell{
				Content: cell,
			}
		}
	}

	// Use the unified table generation system
	return GenerateTableGridFCPXML(tableData, outputPath)
}

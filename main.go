package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"cutlass/fcp"
	"cutlass/vtt"
	"cutlass/wikipedia"
	"cutlass/youtube"
)

func main() {
	var inputFile string
	var segmentMode bool
	var wikipediaMode bool
	var parseMode bool
	var tableMode bool
	var tableNumber int
	flag.StringVar(&inputFile, "i", "", "Input file (required)")
	flag.BoolVar(&segmentMode, "s", false, "Segment mode: break into logical clips with title cards")
	flag.BoolVar(&wikipediaMode, "w", false, "Wikipedia mode: create FCPXML from Wikipedia article tables")
	flag.BoolVar(&parseMode, "p", false, "Parse mode: parse and display existing FCPXML file")
	flag.BoolVar(&tableMode, "t", false, "Table mode: parse and display Wikipedia table data")
	flag.IntVar(&tableNumber, "table", 0, "Table number to display (0 for all, 1-N for specific table)")
	flag.Parse()

	args := flag.Args()
	if inputFile == "" {
		fmt.Fprintf(os.Stderr, "Usage: %s -i <input_file> [output_file]\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  -s: Segment mode - break video into logical clips with title cards\n")
		fmt.Fprintf(os.Stderr, "  -w: Wikipedia mode - create FCPXML from Wikipedia article tables\n")
		fmt.Fprintf(os.Stderr, "  -p: Parse mode - parse and display existing FCPXML file\n")
		fmt.Fprintf(os.Stderr, "  -t: Table mode - parse and display Wikipedia table data\n")
		fmt.Fprintf(os.Stderr, "  -table N: Display specific table number in ASCII format (use with -t)\n")
		os.Exit(1)
	}

	outputFile := "test.fcpxml"
	if len(args) > 0 {
		outputFile = args[0]
	}
	if !strings.HasSuffix(strings.ToLower(outputFile), ".fcpxml") {
		outputFile += ".fcpxml"
	}

	// Handle parse mode
	if parseMode {
		if err := parseFCPXML(inputFile); err != nil {
			fmt.Fprintf(os.Stderr, "Error parsing FCPXML: %v\n", err)
			os.Exit(1)
		}
		return
	}

	// Handle table mode
	if tableMode {
		if err := parseWikipediaTables(inputFile, tableNumber); err != nil {
			fmt.Fprintf(os.Stderr, "Error parsing Wikipedia tables: %v\n", err)
			os.Exit(1)
		}
		return
	}

	// Handle Wikipedia mode
	if wikipediaMode {
		fmt.Printf("Using Wikipedia mode to create FCPXML from article tables...\n")
		if err := generateFromWikipedia(inputFile, outputFile); err != nil {
			fmt.Fprintf(os.Stderr, "Error generating from Wikipedia: %v\n", err)
			os.Exit(1)
		}
		return
	}

	// Check if input looks like a YouTube ID
	youtubeID := ""
	if youtube.IsYouTubeID(inputFile) {
		youtubeID = inputFile
		videoFile, err := youtube.DownloadVideo(inputFile)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error downloading YouTube video: %v\n", err)
			os.Exit(1)
		}

		if err := youtube.DownloadSubtitles(inputFile); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: Could not download subtitles: %v\n", err)
		}

		inputFile = videoFile
	}

	if _, err := os.Stat(inputFile); os.IsNotExist(err) {
		fmt.Fprintf(os.Stderr, "Error: Input file '%s' does not exist\n", inputFile)
		os.Exit(1)
	}

	// Use segment mode if requested
	if segmentMode {
		fmt.Printf("Using segment mode to break video into logical clips...\n")
		if youtubeID != "" {
			if err := breakIntoLogicalParts(youtubeID); err != nil {
				fmt.Fprintf(os.Stderr, "Error breaking into logical parts: %v\n", err)
				os.Exit(1)
			}
		} else {
			// Handle local files in segment mode
			baseID := strings.TrimSuffix(filepath.Base(inputFile), filepath.Ext(inputFile))
			if err := breakIntoLogicalParts(baseID); err != nil {
				fmt.Fprintf(os.Stderr, "Error breaking into logical parts: %v\n", err)
				os.Exit(1)
			}
		}
		return
	}

	// Standard mode - generate simple FCPXML
	if err := fcp.GenerateStandard(inputFile, outputFile); err != nil {
		fmt.Fprintf(os.Stderr, "Error generating FCPXML: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Successfully converted '%s' to '%s'\n", inputFile, outputFile)
}

func breakIntoLogicalParts(youtubeID string) error {
	vttPath := fmt.Sprintf("%s.en.vtt", youtubeID)
	videoPath := fmt.Sprintf("%s.mov", youtubeID)
	outputPath := fmt.Sprintf("%s_clips.fcpxml", youtubeID)

	// Check if files exist
	if _, err := os.Stat(vttPath); os.IsNotExist(err) {
		return fmt.Errorf("VTT file not found: %s", vttPath)
	}
	if _, err := os.Stat(videoPath); os.IsNotExist(err) {
		return fmt.Errorf("video file not found: %s", videoPath)
	}

	// Parse VTT file
	fmt.Printf("Parsing VTT file: %s\n", vttPath)
	segments, err := vtt.ParseFile(vttPath)
	if err != nil {
		return fmt.Errorf("error parsing VTT file: %v", err)
	}

	fmt.Printf("Found %d VTT segments\n", len(segments))

	// Segment into logical clips (6-18 seconds)
	minDuration := 6 * time.Second
	maxDuration := 18 * time.Second
	clips := vtt.SegmentIntoClips(segments, minDuration, maxDuration)

	fmt.Printf("Generated %d clips\n", len(clips))
	for i, clip := range clips {
		fmt.Printf("Clip %d: %v - %v (%.1fs) - %s\n",
			i+1, clip.StartTime, clip.EndTime, clip.Duration.Seconds(),
			clip.Text[:min(50, len(clip.Text))])
	}

	// Generate FCPXML
	fmt.Printf("Generating FCPXML: %s\n", outputPath)
	err = fcp.GenerateClipFCPXML(clips, videoPath, outputPath)
	if err != nil {
		return fmt.Errorf("error generating FCPXML: %v", err)
	}

	fmt.Printf("Successfully generated %s with %d clips\n", outputPath, len(clips))
	return nil
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func parseFCPXML(filePath string) error {
	fcpxml, err := fcp.ParseFCPXML(filePath)
	if err != nil {
		return err
	}

	fcp.DisplayFCPXML(fcpxml)
	return nil
}

func displaySingleColumnPair(table *wikipedia.SimpleTable, leftColIndex, dataColIndex int) {
	if table == nil || len(table.Headers) == 0 {
		fmt.Printf("No table data to display\n")
		return
	}

	// Only display two columns: leftmost + one data column
	leftHeader := table.Headers[leftColIndex]
	dataHeader := table.Headers[dataColIndex]
	
	// Calculate column widths for these two columns
	leftWidth := len(leftHeader)
	dataWidth := len(dataHeader)
	
	// Check row data for max widths
	for _, row := range table.Rows {
		if leftColIndex < len(row) && len(row[leftColIndex]) > leftWidth {
			leftWidth = len(row[leftColIndex])
		}
		if dataColIndex < len(row) && len(row[dataColIndex]) > dataWidth {
			dataWidth = len(row[dataColIndex])
		}
	}
	
	// Limit column width to reasonable max (40 chars) for readability
	if leftWidth > 40 {
		leftWidth = 40
	}
	if dataWidth > 40 {
		dataWidth = 40
	}
	if leftWidth < 3 {
		leftWidth = 3
	}
	if dataWidth < 3 {
		dataWidth = 3
	}
	
	// Print top border
	fmt.Printf("+%s+%s+\n", 
		strings.Repeat("-", leftWidth+2), 
		strings.Repeat("-", dataWidth+2))
	
	// Print headers
	leftTruncated := leftHeader
	if len(leftTruncated) > leftWidth {
		leftTruncated = leftTruncated[:leftWidth-3] + "..."
	}
	dataTruncated := dataHeader
	if len(dataTruncated) > dataWidth {
		dataTruncated = dataTruncated[:dataWidth-3] + "..."
	}
	fmt.Printf("| %-*s | %-*s |\n", leftWidth, leftTruncated, dataWidth, dataTruncated)
	
	// Print header separator
	fmt.Printf("+%s+%s+\n", 
		strings.Repeat("=", leftWidth+2), 
		strings.Repeat("=", dataWidth+2))
	
	// Print rows
	for _, row := range table.Rows {
		leftCell := ""
		dataCell := ""
		
		if leftColIndex < len(row) {
			leftCell = row[leftColIndex]
		}
		if dataColIndex < len(row) {
			dataCell = row[dataColIndex]
		}
		
		// Truncate if too long
		if len(leftCell) > leftWidth {
			leftCell = leftCell[:leftWidth-3] + "..."
		}
		if len(dataCell) > dataWidth {
			dataCell = dataCell[:dataWidth-3] + "..."
		}
		
		fmt.Printf("| %-*s | %-*s |\n", leftWidth, leftCell, dataWidth, dataCell)
	}
	
	// Print bottom border
	fmt.Printf("+%s+%s+\n", 
		strings.Repeat("-", leftWidth+2), 
		strings.Repeat("-", dataWidth+2))
}

// detectTraditionalTable determines if a table should be displayed in traditional format
// Traditional tables have diverse column types and don't repeat the same data structure
// Tennis-style tables have years/dates as columns with similar tournament data
func detectTraditionalTable(table *wikipedia.SimpleTable) bool {
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

// displayTraditionalTable displays each row as a separate 2-column table
func displayTraditionalTable(table *wikipedia.SimpleTable) {
	if table == nil || len(table.Headers) == 0 || len(table.Rows) == 0 {
		fmt.Printf("No data to display\n")
		return
	}
	
	for rowIndex, row := range table.Rows {
		fmt.Printf("--- ROW %d/%d ---\n", rowIndex+1, len(table.Rows))
		
		// Calculate max width for headers and data
		headerWidth := 0
		dataWidth := 0
		
		for i, header := range table.Headers {
			if len(header) > headerWidth {
				headerWidth = len(header)
			}
			if i < len(row) && len(row[i]) > dataWidth {
				dataWidth = len(row[i])
			}
		}
		
		// Set reasonable limits
		if headerWidth > 25 {
			headerWidth = 25
		}
		if dataWidth > 50 {
			dataWidth = 50
		}
		if headerWidth < 10 {
			headerWidth = 10
		}
		if dataWidth < 10 {
			dataWidth = 10
		}
		
		// Print top border
		fmt.Printf("+%s+%s+\n", 
			strings.Repeat("-", headerWidth+2), 
			strings.Repeat("-", dataWidth+2))
		
		// Print header row
		fmt.Printf("| %-*s | %-*s |\n", headerWidth, "Field", dataWidth, "Value")
		
		// Print separator
		fmt.Printf("+%s+%s+\n", 
			strings.Repeat("=", headerWidth+2), 
			strings.Repeat("=", dataWidth+2))
		
		// Print each field-value pair
		for i, header := range table.Headers {
			value := ""
			if i < len(row) {
				value = row[i]
			}
			
			// Truncate if too long
			truncatedHeader := header
			if len(truncatedHeader) > headerWidth {
				truncatedHeader = truncatedHeader[:headerWidth-3] + "..."
			}
			
			truncatedValue := value
			if len(truncatedValue) > dataWidth {
				truncatedValue = truncatedValue[:dataWidth-3] + "..."
			}
			
			fmt.Printf("| %-*s | %-*s |\n", headerWidth, truncatedHeader, dataWidth, truncatedValue)
		}
		
		// Print bottom border
		fmt.Printf("+%s+%s+\n", 
			strings.Repeat("-", headerWidth+2), 
			strings.Repeat("-", dataWidth+2))
		
		// Add spacing between rows (except after the last one)
		if rowIndex < len(table.Rows)-1 {
			fmt.Println()
		}
	}
}

func displayTableASCII(table *wikipedia.SimpleTable) {
	if table == nil || len(table.Headers) == 0 {
		fmt.Printf("No table data to display\n")
		return
	}

	// If table has 2 or fewer columns, display normally
	if len(table.Headers) <= 2 {
		displaySingleColumnPair(table, 0, len(table.Headers)-1)
		return
	}

	// Detect table type: Traditional vs Tennis-style
	isTraditionalTable := detectTraditionalTable(table)
	
	if isTraditionalTable {
		fmt.Printf("=== TRADITIONAL TABLE: Each Row as 2-Column Format ===\n\n")
		displayTraditionalTable(table)
	} else {
		// Tennis-style: Display leftmost column + each data column (skipping leftmost)
		leftColIndex := 0
		totalDataCols := len(table.Headers) - 1
		
		fmt.Printf("=== TENNIS-STYLE TABLE: %d COLUMN PAIRS (Leftmost + Each Data Column) ===\n\n", totalDataCols)
		
		for dataColIndex := 1; dataColIndex < len(table.Headers); dataColIndex++ {
			fmt.Printf("--- TABLE %d/%d: %s + %s ---\n", 
				dataColIndex, totalDataCols, table.Headers[leftColIndex], table.Headers[dataColIndex])
			
			displaySingleColumnPair(table, leftColIndex, dataColIndex)
			
			// Add spacing between tables (except after the last one)
			if dataColIndex < len(table.Headers)-1 {
				fmt.Println()
			}
		}
	}
}

func parseWikipediaTables(articleTitle string, tableNumber int) error {
	// Fetch Wikipedia source
	fmt.Printf("Fetching Wikipedia source for: %s\n", articleTitle)
	source, err := wikipedia.FetchWikipediaSource(articleTitle)
	if err != nil {
		return fmt.Errorf("failed to fetch Wikipedia source: %v", err)
	}

	// Parse the source to extract tables
	fmt.Printf("Parsing Wikipedia source for tables...\n")
	tables, err := wikipedia.ParseWikitableFromSource(source)
	if err != nil {
		return fmt.Errorf("failed to parse Wikipedia source: %v", err)
	}

	if len(tables) == 0 {
		fmt.Printf("No tables found in Wikipedia article '%s'\n", articleTitle)
		return nil
	}

	// If specific table number requested
	if tableNumber > 0 {
		if tableNumber > len(tables) {
			return fmt.Errorf("table %d not found. Article has %d tables", tableNumber, len(tables))
		}
		
		selectedTable := &tables[tableNumber-1]
		fmt.Printf("\n=== TABLE %d FROM WIKIPEDIA ARTICLE '%s' ===\n", tableNumber, articleTitle)
		fmt.Printf("Headers: %d, Rows: %d\n\n", len(selectedTable.Headers), len(selectedTable.Rows))
		
		displayTableASCII(selectedTable)
		return nil
	}

	// Display all tables found (summary mode)
	fmt.Printf("\n=== FOUND %d TABLES IN WIKIPEDIA ARTICLE '%s' ===\n\n", len(tables), articleTitle)
	
	for i, table := range tables {
		fmt.Printf("TABLE %d:\n", i+1)
		fmt.Printf("--------\n")
		fmt.Printf("Headers (%d): %v\n", len(table.Headers), table.Headers)
		fmt.Printf("Rows: %d\n", len(table.Rows))
		
		if len(table.Rows) > 0 {
			fmt.Printf("\nFirst 5 rows:\n")
			for j, row := range table.Rows {
				if j >= 5 {
					break
				}
				fmt.Printf("  Row %d: %v\n", j+1, row)
			}
			
			if len(table.Rows) > 5 {
				fmt.Printf("  ... (and %d more rows)\n", len(table.Rows)-5)
			}
		}
		fmt.Printf("\n")
	}

	// Show best table selection
	bestTable := wikipedia.SelectBestTable(tables)
	if bestTable != nil {
		fmt.Printf("=== BEST TABLE FOR FCPXML GENERATION ===\n")
		fmt.Printf("Headers: %v\n", bestTable.Headers)
		fmt.Printf("Total rows: %d\n", len(bestTable.Rows))
		fmt.Printf("Table data is ready for FCPXML generation\n")
		fmt.Printf("\nTo view a specific table in ASCII format, use: -table N (where N is 1-%d)\n", len(tables))
	}

	return nil
}

func generateFromWikipedia(articleTitle, outputFile string) error {
	// Fetch Wikipedia source
	fmt.Printf("Fetching Wikipedia source for: %s\n", articleTitle)
	source, err := wikipedia.FetchWikipediaSource(articleTitle)
	if err != nil {
		return fmt.Errorf("failed to fetch Wikipedia source: %v", err)
	}

	// Parse the source to extract tables
	fmt.Printf("Parsing Wikipedia source for tables...\n")
	tables, err := wikipedia.ParseWikitableFromSource(source)
	if err != nil {
		return fmt.Errorf("failed to parse Wikipedia source: %v", err)
	}

	if len(tables) == 0 {
		return fmt.Errorf("no tables found in Wikipedia article")
	}

	// Select best table
	bestTable := wikipedia.SelectBestTable(tables)
	if bestTable == nil {
		return fmt.Errorf("no suitable table found")
	}

	fmt.Printf("Table headers: %v\n", bestTable.Headers)
	fmt.Printf("Table has %d rows\n", len(bestTable.Rows))

	// Convert the selected table to the structured TableData format
	tableData := &fcp.TableData{
		Headers: bestTable.Headers,
		Rows:    make([]fcp.TableRow, len(bestTable.Rows)),
	}

	for i, row := range bestTable.Rows {
		tableData.Rows[i] = fcp.TableRow{
			Cells: make([]fcp.TableCell, len(row)),
		}
		for j, cell := range row {
			tableData.Rows[i].Cells[j] = fcp.TableCell{
				Content: cell,
			}
		}
	}

	// Convert to fcp-compatible format
	fcpTable := &fcp.WikiSimpleTable{
		Headers: bestTable.Headers,
		Rows:    bestTable.Rows,
	}

	// Generate FCPXML with multiple table views
	fmt.Printf("Generating FCPXML: %s\n", outputFile)
	err = fcp.GenerateMultiTableFCPXML(fcpTable, outputFile)
	if err != nil {
		return fmt.Errorf("failed to generate FCPXML: %v", err)
	}

	fmt.Printf("Successfully generated Wikipedia table FCPXML: %s\n", outputFile)
	return nil
}

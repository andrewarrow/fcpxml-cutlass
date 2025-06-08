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

func displayTableASCII(table *wikipedia.SimpleTable) {
	if table == nil || len(table.Headers) == 0 {
		fmt.Printf("No table data to display\n")
		return
	}

	// Calculate column widths
	colWidths := make([]int, len(table.Headers))
	
	// Initialize with header widths
	for i, header := range table.Headers {
		colWidths[i] = len(header)
	}
	
	// Check row data for max widths
	for _, row := range table.Rows {
		for i, cell := range row {
			if i < len(colWidths) {
				if len(cell) > colWidths[i] {
					colWidths[i] = len(cell)
				}
			}
		}
	}
	
	// Limit column width to reasonable max (40 chars) for readability
	for i := range colWidths {
		if colWidths[i] > 40 {
			colWidths[i] = 40
		}
		if colWidths[i] < 3 {
			colWidths[i] = 3
		}
	}
	
	// Print top border
	fmt.Print("+")
	for _, width := range colWidths {
		fmt.Print(strings.Repeat("-", width+2) + "+")
	}
	fmt.Println()
	
	// Print headers
	fmt.Print("|")
	for i, header := range table.Headers {
		truncated := header
		if len(truncated) > colWidths[i] {
			truncated = truncated[:colWidths[i]-3] + "..."
		}
		fmt.Printf(" %-*s |", colWidths[i], truncated)
	}
	fmt.Println()
	
	// Print header separator
	fmt.Print("+")
	for _, width := range colWidths {
		fmt.Print(strings.Repeat("=", width+2) + "+")
	}
	fmt.Println()
	
	// Print rows
	for _, row := range table.Rows {
		fmt.Print("|")
		for i := 0; i < len(colWidths); i++ {
			cell := ""
			if i < len(row) {
				cell = row[i]
			}
			
			// Truncate if too long
			if len(cell) > colWidths[i] {
				cell = cell[:colWidths[i]-3] + "..."
			}
			
			fmt.Printf(" %-*s |", colWidths[i], cell)
		}
		fmt.Println()
	}
	
	// Print bottom border
	fmt.Print("+")
	for _, width := range colWidths {
		fmt.Print(strings.Repeat("-", width+2) + "+")
	}
	fmt.Println()
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

	// Generate FCPXML
	fmt.Printf("Generating FCPXML: %s\n", outputFile)
	err = fcp.GenerateTableGridFCPXML(tableData, outputFile)
	if err != nil {
		return fmt.Errorf("failed to generate FCPXML: %v", err)
	}

	fmt.Printf("Successfully generated Wikipedia table FCPXML: %s\n", outputFile)
	return nil
}

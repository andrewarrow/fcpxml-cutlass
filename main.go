package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"cutalyst/fcp"
	"cutalyst/vtt"
	"cutalyst/wikipedia"
	"cutalyst/youtube"
)

func main() {
	var inputFile string
	var segmentMode bool
	var wikipediaMode bool
	flag.StringVar(&inputFile, "i", "", "Input file (required)")
	flag.BoolVar(&segmentMode, "s", false, "Segment mode: break into logical clips with title cards")
	flag.BoolVar(&wikipediaMode, "w", false, "Wikipedia mode: create FCPXML from Wikipedia article tables")
	flag.Parse()

	args := flag.Args()
	if inputFile == "" {
		fmt.Fprintf(os.Stderr, "Usage: %s -i <input_file> [output_file]\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  -s: Segment mode - break video into logical clips with title cards\n")
		fmt.Fprintf(os.Stderr, "  -w: Wikipedia mode - create FCPXML from Wikipedia article tables\n")
		os.Exit(1)
	}

	outputFile := "test.fcpxml"
	if len(args) > 0 {
		outputFile = args[0]
	}
	if !strings.HasSuffix(strings.ToLower(outputFile), ".fcpxml") {
		outputFile += ".fcpxml"
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

func generateFromWikipedia(articleTitle, outputFile string) error {
	// Fetch Wikipedia source
	fmt.Printf("Fetching Wikipedia source for: %s\n", articleTitle)
	source, err := wikipedia.FetchSource(articleTitle)
	if err != nil {
		return fmt.Errorf("failed to fetch Wikipedia source: %v", err)
	}
	
	// Parse the source to extract tables
	fmt.Printf("Parsing Wikipedia source for tables...\n")
	data, err := wikipedia.ParseWikiSource(source)
	if err != nil {
		return fmt.Errorf("failed to parse Wikipedia source: %v", err)
	}
	
	if len(data.Tables) == 0 {
		return fmt.Errorf("no tables found in Wikipedia article")
	}
	
	fmt.Printf("Found %d tables, selecting the best one for FCPXML generation\n", len(data.Tables))
	
	// Find the table with tournament data (look for year headers like 1986)
	bestTableIndex := 0
	maxScore := 0
	for i, table := range data.Tables {
		fmt.Printf("Table %d: %d headers, %d rows\n", i+1, len(table.Headers), len(table.Rows))
		fmt.Printf("  Headers: %v\n", table.Headers)
		
		score := 0
		// Score based on headers containing years and tournaments
		for _, header := range table.Headers {
			if strings.Contains(header, "1986") || strings.Contains(header, "1987") || 
			   strings.Contains(header, "Tournament") || strings.Contains(header, "Grand") {
				score += 10
			}
		}
		// Score based on number of headers (more headers = likely the main table)
		score += len(table.Headers)
		
		fmt.Printf("  Score: %d\n", score)
		if score > maxScore {
			maxScore = score
			bestTableIndex = i
		}
	}
	
	table := data.Tables[bestTableIndex]
	fmt.Printf("Table headers: %v\n", table.Headers)
	fmt.Printf("Table has %d rows\n", len(table.Rows))
	
	// Convert ONLY the selected table to the loosely typed format expected by
	// the generator.  Passing the whole slice previously meant that the
	// generator always processed the first table instead of the best-scoring
	// one.
	var tableData []interface{}

	{
		t := table // the best table selected above
		rows := make([]interface{}, len(t.Rows))
		for j, row := range t.Rows {
			cells := make([]interface{}, len(row.Cells))
			for k, cell := range row.Cells {
				cells[k] = map[string]interface{}{
					"Content":    cell.Content,
					"Style":      cell.Style,
					"Class":      cell.Class,
					"ColSpan":    cell.ColSpan,
					"RowSpan":    cell.RowSpan,
					"Attributes": cell.Attributes,
				}
			}
			rows[j] = map[string]interface{}{"Cells": cells}
		}

		tableData = append(tableData, map[string]interface{}{
			"Headers": t.Headers,
			"Rows":    rows,
		})
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
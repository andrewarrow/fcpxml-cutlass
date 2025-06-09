package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"cutlass/fcp"
	"cutlass/vtt"
	"cutlass/wikipedia"
	"cutlass/youtube"
)

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	command := os.Args[1]
	args := os.Args[2:]

	switch command {
	case "video":
		handleVideoCommand(args)
	case "youtube":
		youtube.HandleYouTubeCommand(args)
	case "youtube-bulk":
		youtube.HandleYouTubeBulkCommand(args)
	case "youtube-bulk-assemble":
		youtube.HandleYouTubeBulkAssembleCommand(args)
	case "wikipedia":
		wikipedia.HandleWikipediaCommand(args)
	case "parse":
		handleParseCommand(args)
	case "table":
		wikipedia.HandleTableCommand(args)
	case "vtt":
		vtt.HandleVTTCommand(args)
	case "vtt-clips":
		vtt.HandleVTTClipsCommand(args)
	case "help", "-h", "--help":
		printUsage()
	default:
		fmt.Fprintf(os.Stderr, "Unknown command: %s\n\n", command)
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Fprintf(os.Stderr, "Usage: %s <command> [options]\n\n", os.Args[0])
	fmt.Fprintf(os.Stderr, "Commands:\n")
	fmt.Fprintf(os.Stderr, "  video <file>              Generate FCPXML from video file\n")
	fmt.Fprintf(os.Stderr, "  youtube <video-id>        Download YouTube video and generate FCPXML\n")
	fmt.Fprintf(os.Stderr, "  youtube-bulk <ids-file>   Download multiple YouTube videos from file\n")
	fmt.Fprintf(os.Stderr, "  youtube-bulk-assemble <ids-file> <name> Create top5.fcpxml from downloaded videos\n")
	fmt.Fprintf(os.Stderr, "  wikipedia <article-title> Generate FCPXML from Wikipedia tables\n")
	fmt.Fprintf(os.Stderr, "  parse <fcpxml-file>       Parse and display FCPXML contents\n")
	fmt.Fprintf(os.Stderr, "  table <article-title>     Display Wikipedia table data\n")
	fmt.Fprintf(os.Stderr, "  vtt <file>                Parse VTT file and display cleaned text\n")
	fmt.Fprintf(os.Stderr, "  vtt-clips <vtt-file> <timecodes> Create FCPXML clips from VTT file at specified timecodes\n")
	fmt.Fprintf(os.Stderr, "            Timecodes can be MM:SS or MM:SS_duration format\n")
	fmt.Fprintf(os.Stderr, "            Example: 01:21_6,02:20_3,03:34_9,05:07_18\n")
	fmt.Fprintf(os.Stderr, "  help                      Show this help message\n\n")
	fmt.Fprintf(os.Stderr, "Options:\n")
	fmt.Fprintf(os.Stderr, "  -s, --segments           Break into logical clips with title cards (video/youtube)\n")
	fmt.Fprintf(os.Stderr, "  -o, --output <file>      Output file (default: test.fcpxml)\n")
	fmt.Fprintf(os.Stderr, "  --table-num <N>          Display specific table number (table command)\n")
}

func handleParseCommand(args []string) {
	if len(args) == 0 {
		fmt.Fprintf(os.Stderr, "Error: FCPXML file required\n")
		fmt.Fprintf(os.Stderr, "Usage: %s parse <fcpxml-file>\n", os.Args[0])
		os.Exit(1)
	}

	inputFile := args[0]
	if err := parseFCPXML(inputFile); err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing FCPXML: %v\n", err)
		os.Exit(1)
	}
}

func handleVideoCommand(args []string) {
	fs := flag.NewFlagSet("video", flag.ExitOnError)
	var segmentMode bool
	var outputFile string

	fs.BoolVar(&segmentMode, "s", false, "Break into logical clips with title cards")
	fs.BoolVar(&segmentMode, "segments", false, "Break into logical clips with title cards")
	fs.StringVar(&outputFile, "o", "test.fcpxml", "Output file")
	fs.StringVar(&outputFile, "output", "test.fcpxml", "Output file")

	if err := fs.Parse(args); err != nil {
		os.Exit(1)
	}

	if fs.NArg() == 0 {
		fmt.Fprintf(os.Stderr, "Error: video file required\n")
		fmt.Fprintf(os.Stderr, "Usage: %s video <file> [options]\n", os.Args[0])
		os.Exit(1)
	}

	inputFile := fs.Arg(0)
	if !strings.HasSuffix(strings.ToLower(outputFile), ".fcpxml") {
		outputFile += ".fcpxml"
	}

	if _, err := os.Stat(inputFile); os.IsNotExist(err) {
		fmt.Fprintf(os.Stderr, "Error: Input file '%s' does not exist\n", inputFile)
		os.Exit(1)
	}

	if segmentMode {
		fmt.Printf("Using segment mode to break video into logical clips...\n")
		baseID := strings.TrimSuffix(filepath.Base(inputFile), filepath.Ext(inputFile))
		clips, videoPath, outputPath, err := vtt.BreakIntoLogicalParts(baseID)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error breaking into logical parts: %v\n", err)
			os.Exit(1)
		}
		
		// Generate FCPXML
		fmt.Printf("Generating FCPXML: %s\n", outputPath)
		err = fcp.GenerateClipFCPXML(clips, videoPath, outputPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error generating FCPXML: %v\n", err)
			os.Exit(1)
		}
		
		fmt.Printf("Successfully generated %s with %d clips\n", outputPath, len(clips))
		return
	}

	if err := fcp.GenerateStandard(inputFile, outputFile); err != nil {
		fmt.Fprintf(os.Stderr, "Error generating FCPXML: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Successfully converted '%s' to '%s'\n", inputFile, outputFile)
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
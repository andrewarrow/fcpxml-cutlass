package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"cutlass/fcp"
	"cutlass/segments"
	"cutlass/speech"
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
	case "segments":
		segments.HandleSegmentsCommand(args)
	case "speech":
		speech.HandleSpeechCommand(args)
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
	fmt.Fprintf(os.Stderr, "  parse <fcpxml-file>       Parse and display FCPXML contents (use 'parse help' for details)\n")
	fmt.Fprintf(os.Stderr, "  table <article-title>     Display Wikipedia table data\n")
	fmt.Fprintf(os.Stderr, "  vtt <file>                Parse VTT file and display cleaned text\n")
	fmt.Fprintf(os.Stderr, "  vtt-clips <vtt-file> <timecodes> Create FCPXML clips from VTT file at specified timecodes\n")
	fmt.Fprintf(os.Stderr, "            Timecodes can be MM:SS or MM:SS_duration format\n")
	fmt.Fprintf(os.Stderr, "            Example: 01:21_6,02:20_3,03:34_9,05:07_18\n")
	fmt.Fprintf(os.Stderr, "  segments <video-id> <timecodes> Create FCPXML clips from video ID in ./data/ at specified timecodes\n")
	fmt.Fprintf(os.Stderr, "            Similar to vtt-clips but looks for video_id in ./data/id.mov\n")
	fmt.Fprintf(os.Stderr, "  speech <text-file>        Generate FCPXML with multiple text elements appearing over time\n")
	fmt.Fprintf(os.Stderr, "            Creates slide animation with each line from text file\n")
	fmt.Fprintf(os.Stderr, "  help                      Show this help message\n\n")
	fmt.Fprintf(os.Stderr, "Options:\n")
	fmt.Fprintf(os.Stderr, "  -s, --segments           Break into logical clips with title cards (video/youtube)\n")
	fmt.Fprintf(os.Stderr, "  -o, --output <file>      Output file (default: test.fcpxml)\n")
	fmt.Fprintf(os.Stderr, "  --table-num <N>          Display specific table number (table command)\n")
}

func handleParseCommand(args []string) {
	if len(args) == 0 {
		printParseUsage()
		os.Exit(1)
	}

	if args[0] == "help" || args[0] == "-h" || args[0] == "--help" {
		printParseHelp()
		return
	}

	fs := flag.NewFlagSet("parse", flag.ExitOnError)
	var tier int
	var showElements, showParams, showAnimation, showResources, showStructure bool

	fs.IntVar(&tier, "tier", 1, "Display tier (1=core, 2=advanced, 3=detailed)")
	fs.IntVar(&tier, "t", 1, "Display tier (1=core, 2=advanced, 3=detailed)")
	fs.BoolVar(&showElements, "elements", false, "Show story elements (clips, titles, effects)")
	fs.BoolVar(&showElements, "e", false, "Show story elements (clips, titles, effects)")
	fs.BoolVar(&showParams, "params", false, "Show parameters and keyframes")
	fs.BoolVar(&showParams, "p", false, "Show parameters and keyframes")
	fs.BoolVar(&showAnimation, "animation", false, "Show animation and keyframe details")
	fs.BoolVar(&showAnimation, "a", false, "Show animation and keyframe details")
	fs.BoolVar(&showResources, "resources", false, "Show detailed resource information")
	fs.BoolVar(&showResources, "r", false, "Show detailed resource information")
	fs.BoolVar(&showStructure, "structure", false, "Show XML structure hierarchy")
	fs.BoolVar(&showStructure, "s", false, "Show XML structure hierarchy")

	if err := fs.Parse(args); err != nil {
		os.Exit(1)
	}

	if fs.NArg() == 0 {
		fmt.Fprintf(os.Stderr, "Error: FCPXML file required\n")
		printParseUsage()
		os.Exit(1)
	}

	inputFile := fs.Arg(0)
	
	// Check if file exists
	if _, err := os.Stat(inputFile); os.IsNotExist(err) {
		fmt.Fprintf(os.Stderr, "Error: File '%s' does not exist\n", inputFile)
		os.Exit(1)
	}

	parseOptions := fcp.ParseOptions{
		Tier:          tier,
		ShowElements:  showElements,
		ShowParams:    showParams,
		ShowAnimation: showAnimation,
		ShowResources: showResources,
		ShowStructure: showStructure,
	}

	if err := parseFCPXMLWithOptions(inputFile, parseOptions); err != nil {
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

func parseFCPXMLWithOptions(filePath string, options fcp.ParseOptions) error {
	fcpxml, err := fcp.ParseFCPXML(filePath)
	if err != nil {
		return err
	}

	fcp.DisplayFCPXMLWithOptions(fcpxml, options)
	return nil
}

func printParseUsage() {
	fmt.Fprintf(os.Stderr, "Usage: %s parse <fcpxml-file> [options]\n", os.Args[0])
	fmt.Fprintf(os.Stderr, "       %s parse help\n\n", os.Args[0])
	fmt.Fprintf(os.Stderr, "Options:\n")
	fmt.Fprintf(os.Stderr, "  -t, --tier <N>           Display tier (1=core, 2=advanced, 3=detailed)\n")
	fmt.Fprintf(os.Stderr, "  -e, --elements           Show story elements (clips, titles, effects)\n")
	fmt.Fprintf(os.Stderr, "  -p, --params             Show parameters and keyframes\n")
	fmt.Fprintf(os.Stderr, "  -a, --animation          Show animation and keyframe details\n")
	fmt.Fprintf(os.Stderr, "  -r, --resources          Show detailed resource information\n")
	fmt.Fprintf(os.Stderr, "  -s, --structure          Show XML structure hierarchy\n")
	fmt.Fprintf(os.Stderr, "  help                     Show detailed help and examples\n")
}

func printParseHelp() {
	fmt.Printf("=== FCPXML Parser: Understanding Video's Hidden Language ===\n\n")
	
	fmt.Printf("FCPXML is the instruction manual that tells Final Cut Pro how to assemble\n")
	fmt.Printf("all your video pieces into a finished film. Think of it as digital LEGO\n")
	fmt.Printf("instructions, but for Hollywood movies.\n\n")
	
	fmt.Printf("=== TIER 1: The Foundation (Every Video Needs These) ===\n\n")
	fmt.Printf("These are the DNA of digital video - present in every project:\n\n")
	
	fmt.Printf("• fcpxml root element - Your project's birth certificate\n")
	fmt.Printf("  Declares the FCPXML version and contains everything else\n\n")
	
	fmt.Printf("• resources section - Your digital warehouse\n")
	fmt.Printf("  Catalogs every asset: video files, audio, images, effects\n")
	fmt.Printf("  Each gets a unique ID (r1, r2, r3) for precise referencing\n\n")
	
	fmt.Printf("• sequence element - Your movie timeline\n")
	fmt.Printf("  Contains the 'spine' where clips are arranged in time\n")
	fmt.Printf("  Defines duration, frame rate, and audio layout\n\n")
	
	fmt.Printf("Examples:\n")
	fmt.Printf("  %s parse video.fcpxml                    # Show tier 1 (default)\n", os.Args[0])
	fmt.Printf("  %s parse video.fcpxml --tier 1           # Explicitly show core elements\n", os.Args[0])
	fmt.Printf("  %s parse video.fcpxml --resources        # Focus on resources section\n\n", os.Args[0])
	
	fmt.Printf("=== TIER 2: When Things Get Interesting ===\n\n")
	fmt.Printf("Moving beyond basic editing to engaging content:\n\n")
	
	fmt.Printf("• asset-clips - Your actual video/audio pieces on timeline\n")
	fmt.Printf("  Instructions for 'take this part of this file, put it here for this long'\n\n")
	
	fmt.Printf("• title elements - All text overlays and graphics\n")
	fmt.Printf("  From simple captions to complex animated 3D text\n\n")
	
	fmt.Printf("• effects and filters - Transform your footage\n")
	fmt.Printf("  Reference Motion templates with dozens of animatable parameters\n\n")
	
	fmt.Printf("• audio elements - Separate sound control\n")
	fmt.Printf("  Dialogue, music, effects on different tracks with individual mixing\n\n")
	
	fmt.Printf("Examples:\n")
	fmt.Printf("  %s parse video.fcpxml --tier 2           # Show tier 1 + 2 elements\n", os.Args[0])
	fmt.Printf("  %s parse video.fcpxml --elements         # Focus on timeline elements\n", os.Args[0])
	fmt.Printf("  %s parse video.fcpxml -t 2 -e            # Tier 2 + detailed elements\n\n", os.Args[0])
	
	fmt.Printf("=== TIER 3: Where Magic Happens ===\n\n")
	fmt.Printf("The genuinely mind-bending complexity:\n\n")
	
	fmt.Printf("• keyframe animations - Static properties become moving, changing\n")
	fmt.Printf("  'Text starts red at 0s, becomes purple at 2s, ends blue at 4s'\n\n")
	
	fmt.Printf("• parameter hierarchies - Nested layers of control\n")
	fmt.Printf("  Position contains X/Y, each with keyframes and interpolation curves\n\n")
	
	fmt.Printf("• lane systems - Vertical stacking in time\n")
	fmt.Printf("  Lane 0=main, Lane 1=above, Lane -1=below for complex compositions\n\n")
	
	fmt.Printf("• timing systems - Frame-perfect rational numbers\n")
	fmt.Printf("  '5000/2000s' instead of '2.5s' prevents rounding errors\n\n")
	
	fmt.Printf("Examples:\n")
	fmt.Printf("  %s parse video.fcpxml --tier 3           # Show everything\n", os.Args[0])
	fmt.Printf("  %s parse video.fcpxml --animation        # Focus on keyframes/timing\n", os.Args[0])
	fmt.Printf("  %s parse video.fcpxml --params           # Show all parameters\n", os.Args[0])
	fmt.Printf("  %s parse video.fcpxml -t 3 -a -p -s      # Maximum detail view\n\n", os.Args[0])
	
	fmt.Printf("=== Understanding the Complexity ===\n\n")
	fmt.Printf("Why is FCPXML 'insanely complicated'? It's not just describing what you see,\n")
	fmt.Printf("it's describing how to RECREATE what you see with perfect fidelity.\n\n")
	
	fmt.Printf("A simple 'Hello World' title floating over video actually describes:\n")
	fmt.Printf("• Exact font, size, and 3D lighting model creating subtle shadows\n")
	fmt.Printf("• Material properties making it look metallic\n")
	fmt.Printf("• Keyframe animation sliding it in from the left\n")
	fmt.Printf("• Color space ensuring it looks right on different monitors\n")
	fmt.Printf("• Audio synchronization keeping it in time with music\n")
	fmt.Printf("• Mathematical transforms positioning it precisely in 3D space\n\n")
	
	fmt.Printf("This complexity exists so creativity can be simple. Directors work with\n")
	fmt.Printf("intuitive interfaces while FCPXML handles mathematical precision underneath.\n\n")
	
	fmt.Printf("=== Quick Start Guide ===\n\n")
	fmt.Printf("Start simple, then drill down:\n")
	fmt.Printf("1. %s parse myproject.fcpxml              # See the foundation\n", os.Args[0])
	fmt.Printf("2. %s parse myproject.fcpxml -t 2          # Add story elements\n", os.Args[0])
	fmt.Printf("3. %s parse myproject.fcpxml -t 3 -a       # Dive into animations\n", os.Args[0])
	fmt.Printf("4. %s parse myproject.fcpxml -s            # See XML structure\n\n", os.Args[0])
	
	fmt.Printf("FCPXML is the hidden language transforming human creativity into digital\n")
	fmt.Printf("reality, one precisely timed, mathematically perfect frame at a time.\n")
}
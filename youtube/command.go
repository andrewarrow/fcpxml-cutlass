package youtube

import (
	"cutlass/fcp"
	"cutlass/vtt"
	"flag"
	"fmt"
	"os"
)

func HandleYouTubeCommand(args []string) {
	fs := flag.NewFlagSet("youtube", flag.ExitOnError)
	var segmentMode bool

	fs.BoolVar(&segmentMode, "s", false, "Break into logical clips with title cards")
	fs.BoolVar(&segmentMode, "segments", false, "Break into logical clips with title cards")

	if err := fs.Parse(args); err != nil {
		os.Exit(1)
	}

	if fs.NArg() == 0 {
		fmt.Fprintf(os.Stderr, "Error: YouTube video ID required\n")
		fmt.Fprintf(os.Stderr, "Usage: %s youtube <video-id> [options]\n", os.Args[0])
		os.Exit(1)
	}

	youtubeID := fs.Arg(0)
	if !IsYouTubeID(youtubeID) {
		fmt.Fprintf(os.Stderr, "Error: Invalid YouTube video ID: %s\n", youtubeID)
		os.Exit(1)
	}

	_, err := DownloadVideo(youtubeID)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error downloading YouTube video: %v\n", err)
		os.Exit(1)
	}

	if err := DownloadSubtitles(youtubeID); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: Could not download subtitles: %v\n", err)
	}

	if segmentMode {
		fmt.Printf("Using segment mode to break video into logical clips...\n")
		clips, videoPath, outputPath, err := vtt.BreakIntoLogicalParts(youtubeID)
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
}

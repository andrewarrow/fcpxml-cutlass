package vtt

import (
	"flag"
	"fmt"
	"os"
)

func HandleVTTCommand(args []string) {
	if len(args) == 0 {
		fmt.Fprintf(os.Stderr, "Error: VTT file required\n")
		fmt.Fprintf(os.Stderr, "Usage: %s vtt <file>\n", os.Args[0])
		os.Exit(1)
	}

	inputFile := args[0]
	if _, err := os.Stat(inputFile); os.IsNotExist(err) {
		fmt.Fprintf(os.Stderr, "Error: VTT file '%s' does not exist\n", inputFile)
		os.Exit(1)
	}

	if err := ParseAndDisplayCleanText(inputFile); err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing VTT file: %v\n", err)
		os.Exit(1)
	}
}

func HandleSegmentsCommand(args []string) {
	fs := flag.NewFlagSet("segments", flag.ExitOnError)
	var outputFile string

	fs.StringVar(&outputFile, "o", "", "Output file (default: <video-id>_segments.fcpxml)")
	fs.StringVar(&outputFile, "output", "", "Output file (default: <video-id>_segments.fcpxml)")

	if err := fs.Parse(args); err != nil {
		os.Exit(1)
	}

	if fs.NArg() < 2 {
		fmt.Fprintf(os.Stderr, "Error: video ID and timecodes required\n")
		fmt.Fprintf(os.Stderr, "Usage: %s segments <video-id> <timecodes>\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "Example: %s segments IBnNedMh4Pg 01:21_6,02:20_3,03:34_9,05:07_18\n", os.Args[0])
		os.Exit(1)
	}

	videoID := fs.Arg(0)
	timecodesStr := fs.Arg(1)

	if err := GenerateSegments(videoID, timecodesStr, outputFile); err != nil {
		fmt.Fprintf(os.Stderr, "Error generating segments: %v\n", err)
		os.Exit(1)
	}
}

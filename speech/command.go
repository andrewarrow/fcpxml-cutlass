package speech

import (
	"flag"
	"fmt"
	"os"
)

func HandleSpeechCommand(args []string) {
	fs := flag.NewFlagSet("speech", flag.ExitOnError)
	var outputFile, videoFile string

	fs.StringVar(&outputFile, "o", "data/test_speech.fcpxml", "Output file")
	fs.StringVar(&outputFile, "output", "data/test_speech.fcpxml", "Output file")
	fs.StringVar(&videoFile, "v", "", "Video file path (required)")
	fs.StringVar(&videoFile, "video", "", "Video file path (required)")

	if err := fs.Parse(args); err != nil {
		os.Exit(1)
	}

	if fs.NArg() == 0 {
		fmt.Fprintf(os.Stderr, "Error: text file required\n")
		fmt.Fprintf(os.Stderr, "Usage: %s speech <text-file> -v <video-file> [options]\n", os.Args[0])
		os.Exit(1)
	}

	if videoFile == "" {
		fmt.Fprintf(os.Stderr, "Error: video file required\n")
		fmt.Fprintf(os.Stderr, "Usage: %s speech <text-file> -v <video-file> [options]\n", os.Args[0])
		os.Exit(1)
	}

	inputFile := fs.Arg(0)
	if _, err := os.Stat(inputFile); os.IsNotExist(err) {
		fmt.Fprintf(os.Stderr, "Error: Input file '%s' does not exist\n", inputFile)
		os.Exit(1)
	}

	if _, err := os.Stat(videoFile); os.IsNotExist(err) {
		fmt.Fprintf(os.Stderr, "Error: Video file '%s' does not exist\n", videoFile)
		os.Exit(1)
	}

	if err := GenerateSpeechFCPXML(inputFile, outputFile, videoFile); err != nil {
		fmt.Fprintf(os.Stderr, "Error generating speech FCPXML: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Successfully generated '%s' from '%s' using video '%s'\n", outputFile, inputFile, videoFile)
}
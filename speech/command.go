package speech

import (
	"flag"
	"fmt"
	"os"
)

func HandleSpeechCommand(args []string) {
	fs := flag.NewFlagSet("speech", flag.ExitOnError)
	var outputFile string

	fs.StringVar(&outputFile, "o", "data/test_speech.fcpxml", "Output file")
	fs.StringVar(&outputFile, "output", "data/test_speech.fcpxml", "Output file")

	if err := fs.Parse(args); err != nil {
		os.Exit(1)
	}

	if fs.NArg() < 2 {
		fmt.Fprintf(os.Stderr, "Error: text file and video/image file required\n")
		fmt.Fprintf(os.Stderr, "Usage: %s speech <text-file> <video-or-image-file> [options]\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "       The video or image file will be used as background media for the text animations\n")
		os.Exit(1)
	}

	inputFile := fs.Arg(0)
	videoFile := fs.Arg(1)

	if _, err := os.Stat(inputFile); os.IsNotExist(err) {
		fmt.Fprintf(os.Stderr, "Error: Input file '%s' does not exist\n", inputFile)
		os.Exit(1)
	}

	if _, err := os.Stat(videoFile); os.IsNotExist(err) {
		fmt.Fprintf(os.Stderr, "Error: Video or image file '%s' does not exist\n", videoFile)
		os.Exit(1)
	}

	if err := GenerateSpeechFCPXML(inputFile, outputFile, videoFile); err != nil {
		fmt.Fprintf(os.Stderr, "Error generating speech FCPXML: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Successfully generated '%s' from '%s' using media '%s'\n", outputFile, inputFile, videoFile)
}
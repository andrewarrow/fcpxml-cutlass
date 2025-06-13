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

func HandleResumeCommandWithOutput(args []string, outputFile string) {
	if len(args) < 1 {
		fmt.Fprintf(os.Stderr, "Error: resume file required\n")
		fmt.Fprintf(os.Stderr, "Usage: %s content resume <resume-file> [options]\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "       The resume file should contain PNG filenames followed by their associated text lines\n")
		os.Exit(1)
	}

	resumeFile := args[0]

	if _, err := os.Stat(resumeFile); os.IsNotExist(err) {
		fmt.Fprintf(os.Stderr, "Error: Resume file '%s' does not exist\n", resumeFile)
		os.Exit(1)
	}

	if err := GenerateResumeFCPXML(resumeFile, outputFile); err != nil {
		fmt.Fprintf(os.Stderr, "Error generating resume FCPXML: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Successfully generated '%s' from '%s'\n", outputFile, resumeFile)
}
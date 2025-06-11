package time

import (
	"flag"
	"fmt"
	"os"
)

func HandleTimeCommand(args []string) {
	fs := flag.NewFlagSet("time", flag.ExitOnError)
	var outputFile string

	fs.StringVar(&outputFile, "o", "test_output.fcpxml", "Output file")
	fs.StringVar(&outputFile, "output", "test_output.fcpxml", "Output file")

	if err := fs.Parse(args); err != nil {
		os.Exit(1)
	}

	if fs.NArg() == 0 {
		fmt.Fprintf(os.Stderr, "Error: .time file required\n")
		fmt.Fprintf(os.Stderr, "Usage: %s time [options] <time-file>\n", os.Args[0])
		os.Exit(1)
	}

	inputFile := fs.Arg(0)
	if _, err := os.Stat(inputFile); os.IsNotExist(err) {
		fmt.Fprintf(os.Stderr, "Error: Input file '%s' does not exist\n", inputFile)
		os.Exit(1)
	}

	if err := GenerateTimeFCPXML(inputFile, outputFile); err != nil {
		fmt.Fprintf(os.Stderr, "Error generating time FCPXML: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Successfully generated '%s' from '%s'\n", outputFile, inputFile)
}
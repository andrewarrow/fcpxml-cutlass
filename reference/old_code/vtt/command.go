package vtt

import (
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

package vtt

import (
	"flag"
	"fmt"
	"os"
)

func HandleVTTClipsCommand(args []string) {
	fs := flag.NewFlagSet("vtt-clips", flag.ExitOnError)
	var outputFile string

	fs.StringVar(&outputFile, "o", "", "Output file (default: <vtt-basename>_clips.fcpxml)")
	fs.StringVar(&outputFile, "output", "", "Output file (default: <vtt-basename>_clips.fcpxml)")

	if err := fs.Parse(args); err != nil {
		os.Exit(1)
	}

	if fs.NArg() < 2 {
		fmt.Fprintf(os.Stderr, "Error: VTT file and timecodes required\n")
		fmt.Fprintf(os.Stderr, "Usage: %s vtt-clips <vtt-file> <timecodes>\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "Example: %s vtt-clips IBnNedMh4Pg.en.vtt 01:21_6,02:20_3,03:34_9,05:07_18\n", os.Args[0])
		os.Exit(1)
	}

	vttFile := fs.Arg(0)
	timecodesStr := fs.Arg(1)

	if _, err := os.Stat(vttFile); os.IsNotExist(err) {
		fmt.Fprintf(os.Stderr, "Error: VTT file '%s' does not exist\n", vttFile)
		os.Exit(1)
	}

	if err := GenerateVTTClips(vttFile, timecodesStr, outputFile); err != nil {
		fmt.Fprintf(os.Stderr, "Error generating VTT clips: %v\n", err)
		os.Exit(1)
	}
}

package wikipedia

import (
	"flag"
	"fmt"
	"os"
	"strings"
)

func HandleWikipediaCommand(args []string) {
	fs := flag.NewFlagSet("wikipedia", flag.ExitOnError)
	var outputFile string

	fs.StringVar(&outputFile, "o", "", "Output file")
	fs.StringVar(&outputFile, "output", "", "Output file")

	if err := fs.Parse(args); err != nil {
		os.Exit(1)
	}

	if fs.NArg() == 0 {
		fmt.Fprintf(os.Stderr, "Error: Wikipedia article title required\n")
		fmt.Fprintf(os.Stderr, "Usage: %s wikipedia <article-title> [options]\n", os.Args[0])
		os.Exit(1)
	}

	articleTitle := fs.Arg(0)

	// If no output file specified, use article title as filename
	if outputFile == "" {
		outputFile = articleTitle + ".fcpxml"
	} else if !strings.HasSuffix(strings.ToLower(outputFile), ".fcpxml") {
		outputFile += ".fcpxml"
	}

	fmt.Printf("Using Wikipedia mode to create FCPXML from article tables...\n")
	if err := GenerateFromWikipedia(articleTitle, outputFile); err != nil {
		fmt.Fprintf(os.Stderr, "Error generating from Wikipedia: %v\n", err)
		os.Exit(1)
	}
}

func HandleTableCommand(args []string) {
	fs := flag.NewFlagSet("table", flag.ExitOnError)
	var tableNumber int

	fs.IntVar(&tableNumber, "table-num", 0, "Table number to display (0 for all, 1-N for specific table)")

	if err := fs.Parse(args); err != nil {
		os.Exit(1)
	}

	if fs.NArg() == 0 {
		fmt.Fprintf(os.Stderr, "Error: Wikipedia article title required\n")
		fmt.Fprintf(os.Stderr, "Usage: %s table <article-title> [--table-num N]\n", os.Args[0])
		os.Exit(1)
	}

	articleTitle := fs.Arg(0)
	if err := ParseWikipediaTables(articleTitle, tableNumber); err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing Wikipedia tables: %v\n", err)
		os.Exit(1)
	}
}

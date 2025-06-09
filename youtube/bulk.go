package youtube

import (
	"fmt"
	"os"
)

func HandleYouTubeBulkCommand(args []string) {
	if len(args) == 0 {
		fmt.Fprintf(os.Stderr, "Error: Video IDs file required\n")
		fmt.Fprintf(os.Stderr, "Usage: %s youtube-bulk <ids-file>\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "File should contain one video ID per line\n")
		os.Exit(1)
	}

	idsFile := args[0]
	if _, err := os.Stat(idsFile); os.IsNotExist(err) {
		fmt.Fprintf(os.Stderr, "Error: IDs file '%s' does not exist\n", idsFile)
		os.Exit(1)
	}

	if err := DownloadMultipleVideos(idsFile); err != nil {
		fmt.Fprintf(os.Stderr, "Error downloading videos: %v\n", err)
		os.Exit(1)
	}
}

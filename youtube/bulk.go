package youtube

import (
	"fmt"
	"os"
	"strings"

	"cutlass/fcp"
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
func DownloadMultipleVideos(idsFile string) error {
	// Read video IDs from file
	videoIDs, err := readVideoIDsFromFile(idsFile)
	if err != nil {
		return fmt.Errorf("failed to read video IDs: %v", err)
	}

	if len(videoIDs) == 0 {
		return fmt.Errorf("no video IDs found in file")
	}

	fmt.Printf("Found %d video IDs to download\n", len(videoIDs))

	// Download each video
	for i, videoID := range videoIDs {
		fmt.Printf("\n=== Downloading video %d/%d: %s ===\n", i+1, len(videoIDs), videoID)

		DownloadVideo(videoID)
		DownloadSubtitles(videoID)

		fmt.Printf("Successfully downloaded video %s\n", videoID)
	}

	fmt.Printf("\nBulk download completed\n")
	return nil
}

func HandleYouTubeBulkAssembleCommand(args []string) {
	if len(args) < 2 {
		fmt.Fprintf(os.Stderr, "Error: IDs file and name required\n")
		fmt.Fprintf(os.Stderr, "Usage: %s youtube-bulk-assemble <ids-file> <name>\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "Example: %s youtube-bulk-assemble ids.txt \"Brain Smith\"\n", os.Args[0])
		os.Exit(1)
	}

	idsFile := args[0]
	name := args[1]

	if _, err := os.Stat(idsFile); os.IsNotExist(err) {
		fmt.Fprintf(os.Stderr, "Error: IDs file '%s' does not exist\n", idsFile)
		os.Exit(1)
	}

	if err := AssembleTop5FCPXML(idsFile, name); err != nil {
		fmt.Fprintf(os.Stderr, "Error assembling top5.fcpxml: %v\n", err)
		os.Exit(1)
	}

	outputFile := strings.ReplaceAll(name, " ", "_") + "_top5.fcpxml"
	fmt.Printf("Successfully created %s\n", outputFile)
}

func AssembleTop5FCPXML(idsFile, name string) error {
	// Read video IDs from file
	videoIDs, err := readVideoIDsFromFile(idsFile)
	if err != nil {
		return fmt.Errorf("failed to read video IDs: %v", err)
	}

	if len(videoIDs) == 0 {
		return fmt.Errorf("no video IDs found in file")
	}

	// Read the template top5.fcpxml
	templatePath := "top5.fcpxml"
	if _, err := os.Stat(templatePath); os.IsNotExist(err) {
		return fmt.Errorf("template file %s does not exist", templatePath)
	}

	// Create output filename from name
	outputFile := strings.ReplaceAll(name, " ", "_") + "_top5.fcpxml"
	
	// Generate the FCPXML with the new name and random clips
	err = fcp.GenerateTop5FCPXML(templatePath, videoIDs, name, outputFile)
	if err != nil {
		return fmt.Errorf("failed to generate FCPXML: %v", err)
	}

	return nil
}

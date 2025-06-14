package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"cutlass/fcp"
	"cutlass/vtt"

	"github.com/spf13/cobra"
)

var mediaCmd = &cobra.Command{
	Use:   "media",
	Short: "Video processing and conversion commands",
	Long:  "Commands for processing video files and converting them to FCPXML format.",
}

var videoCmd = &cobra.Command{
	Use:   "video <file>",
	Short: "Generate FCPXML from video file",
	Long:  "Convert a video file to FCPXML format with optional segment breaking.",
	Args:  cobra.ExactArgs(1),
	RunE:  runVideoCommand,
}

var keyframesCmd = &cobra.Command{
	Use:   "keyframes <video-id>",
	Short: "Extract keyframes from video file",
	Long:  "Extract all keyframes from a video file as JPEG images using ffmpeg.",
	Args:  cobra.ExactArgs(1),
	RunE:  runKeyframesCommand,
}

var segmentMode bool
var outputFile string

func init() {
	mediaCmd.AddCommand(videoCmd)
	mediaCmd.AddCommand(keyframesCmd)
	
	videoCmd.Flags().BoolVarP(&segmentMode, "segments", "s", false, "Break into logical clips with title cards")
	videoCmd.Flags().StringVarP(&outputFile, "output", "o", "test.fcpxml", "Output file")
}

func runVideoCommand(cmd *cobra.Command, args []string) error {
	inputFile := args[0]
	
	if !strings.HasSuffix(strings.ToLower(outputFile), ".fcpxml") {
		outputFile += ".fcpxml"
	}

	if _, err := os.Stat(inputFile); os.IsNotExist(err) {
		return fmt.Errorf("input file '%s' does not exist", inputFile)
	}

	if segmentMode {
		fmt.Printf("Using segment mode to break video into logical clips...\n")
		baseID := strings.TrimSuffix(filepath.Base(inputFile), filepath.Ext(inputFile))
		clips, videoPath, outputPath, err := vtt.BreakIntoLogicalParts(baseID)
		if err != nil {
			return fmt.Errorf("error breaking into logical parts: %v", err)
		}
		
		// Generate FCPXML
		fmt.Printf("Generating FCPXML: %s\n", outputPath)
		err = fcp.GenerateClipFCPXML(clips, videoPath, outputPath)
		if err != nil {
			return fmt.Errorf("error generating FCPXML: %v", err)
		}
		
		fmt.Printf("Successfully generated %s with %d clips\n", outputPath, len(clips))
		return nil
	}

	if err := fcp.GenerateStandard(inputFile, outputFile); err != nil {
		return fmt.Errorf("error generating FCPXML: %v", err)
	}

	fmt.Printf("Successfully converted '%s' to '%s'\n", inputFile, outputFile)
	return nil
}

func runKeyframesCommand(cmd *cobra.Command, args []string) error {
	videoID := args[0]
	
	if err := vtt.ExtractKeyframes(videoID); err != nil {
		return fmt.Errorf("error extracting keyframes: %v", err)
	}
	
	fmt.Printf("Successfully extracted keyframes for video ID: %s\n", videoID)
	return nil
}
package cmd

import (
	"cutlass/fcp"
	"fmt"
	"strconv"
	"time"

	"github.com/spf13/cobra"
)

var fcpCmd = &cobra.Command{
	Use:   "fcp",
	Short: "FCPXML generation tools",
	Long: `FCPXML generation tools for creating Final Cut Pro XML files.

This command provides various subcommands for generating and working with FCPXML files.
Use 'cutlass fcp --help' to see all available subcommands.`,
	Run: func(cmd *cobra.Command, args []string) {
		// Show help when called without subcommands
		cmd.Help()
	},
}

var createEmptyCmd = &cobra.Command{
	Use:   "create-empty [filename]",
	Short: "Generate an empty FCPXML file from structs",
	Long:  `Generate a basic empty FCPXML file structure using the fcp package structs.`,
	Args:  cobra.RangeArgs(0, 1),
	Run: func(cmd *cobra.Command, args []string) {
		// Get output filename from flag or generate default
		output, _ := cmd.Flags().GetString("output")
		var filename string
		
		if output != "" {
			filename = output
		} else if len(args) > 0 {
			filename = args[0]
		} else {
			// Generate default filename with unix timestamp
			timestamp := time.Now().Unix()
			filename = fmt.Sprintf("cutlass_%d.fcpxml", timestamp)
		}
		
		_, err := fcp.GenerateEmpty(filename)
		if err != nil {
			fmt.Printf("Error generating FCPXML: %v\n", err)
			return
		}
		fmt.Printf("Generated empty FCPXML: %s\n", filename)
	},
}

var addVideoCmd = &cobra.Command{
	Use:   "add-video [video-file]",
	Short: "Add a video to an FCPXML file using structs",
	Long:  `Add a video asset and asset-clip to an FCPXML file using the fcp package structs.`,
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		videoFile := args[0]
		
		// Get output filename from flag or generate default
		output, _ := cmd.Flags().GetString("output")
		var filename string
		
		if output != "" {
			filename = output
		} else {
			// Generate default filename with unix timestamp
			timestamp := time.Now().Unix()
			filename = fmt.Sprintf("cutlass_%d.fcpxml", timestamp)
		}
		
		// Generate empty FCPXML structure
		fcpxml, err := fcp.GenerateEmpty("")
		if err != nil {
			fmt.Printf("Error creating FCPXML structure: %v\n", err)
			return
		}
		
		// Add video to the structure
		err = fcp.AddVideo(fcpxml, videoFile)
		if err != nil {
			fmt.Printf("Error adding video: %v\n", err)
			return
		}
		
		// Write to file
		err = fcp.WriteToFile(fcpxml, filename)
		if err != nil {
			fmt.Printf("Error writing FCPXML: %v\n", err)
			return
		}
		
		fmt.Printf("Generated FCPXML with video: %s\n", filename)
	},
}

var addImageCmd = &cobra.Command{
	Use:   "add-image [image-file]",
	Short: "Add an image to an FCPXML file using structs",
	Long:  `Add an image asset and asset-clip to an FCPXML file using the fcp package structs. Supports PNG, JPG, and JPEG files.`,
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		imageFile := args[0]
		
		// Get duration from flag (default 9 seconds)
		durationStr, _ := cmd.Flags().GetString("duration")
		duration, err := strconv.ParseFloat(durationStr, 64)
		if err != nil {
			fmt.Printf("Error parsing duration '%s': %v\n", durationStr, err)
			return
		}
		
		// Get output filename from flag or generate default
		output, _ := cmd.Flags().GetString("output")
		var filename string
		
		if output != "" {
			filename = output
		} else {
			// Generate default filename with unix timestamp
			timestamp := time.Now().Unix()
			filename = fmt.Sprintf("cutlass_%d.fcpxml", timestamp)
		}
		
		// Generate empty FCPXML structure
		fcpxml, err := fcp.GenerateEmpty("")
		if err != nil {
			fmt.Printf("Error creating FCPXML structure: %v\n", err)
			return
		}
		
		// Add image to the structure
		err = fcp.AddImage(fcpxml, imageFile, duration)
		if err != nil {
			fmt.Printf("Error adding image: %v\n", err)
			return
		}
		
		// Write to file
		err = fcp.WriteToFile(fcpxml, filename)
		if err != nil {
			fmt.Printf("Error writing FCPXML: %v\n", err)
			return
		}
		
		fmt.Printf("Generated FCPXML with image: %s (duration: %.1fs)\n", filename, duration)
	},
}

func init() {
	// Add output flag to create-empty subcommand
	createEmptyCmd.Flags().StringP("output", "o", "", "Output filename (defaults to cutlass_unixtime.fcpxml)")
	
	// Add output flag to add-video subcommand
	addVideoCmd.Flags().StringP("output", "o", "", "Output filename (defaults to cutlass_unixtime.fcpxml)")
	
	// Add flags to add-image subcommand
	addImageCmd.Flags().StringP("output", "o", "", "Output filename (defaults to cutlass_unixtime.fcpxml)")
	addImageCmd.Flags().StringP("duration", "d", "9", "Duration in seconds (default 9)")
	
	fcpCmd.AddCommand(createEmptyCmd)
	fcpCmd.AddCommand(addVideoCmd)
	fcpCmd.AddCommand(addImageCmd)
}
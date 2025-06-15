package cmd

import (
	"cutlass/fcp"
	"fmt"

	"github.com/spf13/cobra"
)

var fcpCmd = &cobra.Command{
	Use:   "fcp [filename.fcpxml]",
	Short: "Generate an empty FCPXML file from structs",
	Long:  `Generate a basic empty FCPXML file structure using the fcp package structs.`,
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		filename := args[0]
		_, err := fcp.GenerateEmpty(filename)
		if err != nil {
			fmt.Printf("Error generating FCPXML: %v\n", err)
			return
		}
		fmt.Printf("Generated empty FCPXML: %s\n", filename)
	},
}

var addVideoCmd = &cobra.Command{
	Use:   "add-video [filename.fcpxml] [video-file]",
	Short: "Add a video to an FCPXML file using structs",
	Long:  `Add a video asset and asset-clip to an FCPXML file using the fcp package structs.`,
	Args:  cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		filename := args[0]
		videoFile := args[1]
		
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

func init() {
	fcpCmd.AddCommand(addVideoCmd)
}
package build2

import (
	"fmt"
	"os"
	"strings"

	"cutlass/build2/api"
	"github.com/spf13/cobra"
)

var BuildCmd = &cobra.Command{
	Use:   "build2 [filename] [add-video] [media-file]",
	Short: "Build a blank FCP project or add media to existing project (Refactored Build2)",
	Long:  "Create a blank Final Cut Pro project from empty.fcpxml template, or add a video/image to it using the new build2 architecture",
	Args:  cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		filename := args[0]
		if !strings.HasSuffix(filename, ".fcpxml") {
			filename += ".fcpxml"
		}
		
		// Check if this is an add-video command
		if len(args) >= 3 && args[1] == "add-video" {
			mediaFile := args[2]
			
			// Get the --with-text, --with-sound, --with-duration, and --with-slide flag values
			withText, _ := cmd.Flags().GetString("with-text")
			withSound, _ := cmd.Flags().GetString("with-sound")
			withDuration, _ := cmd.Flags().GetString("with-duration")
			withSlide, _ := cmd.Flags().GetBool("with-slide")
			
			// Add media to the project using build2 API
			err := api.AddVideoToProjectWithSlide(filename, mediaFile, withText, withSound, withDuration, withSlide)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error adding media to project: %v\n", err)
				os.Exit(1)
			}
			
			fmt.Printf("Added media %s to project %s\n", mediaFile, filename)
			if withText != "" {
				fmt.Printf("Added text overlay: %s\n", withText)
			}
			if withSound != "" {
				fmt.Printf("Added audio file: %s\n", withSound)
			}
		} else {
			// Just create a blank project using build2 API
			err := api.CreateBlankProject(filename)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error creating blank project: %v\n", err)
				os.Exit(1)
			}
			
			fmt.Printf("Created blank project: %s\n", filename)
		}
	},
}

func init() {
	BuildCmd.Flags().String("with-text", "", "Add text overlay on top of the video")
	BuildCmd.Flags().String("with-sound", "", "Add audio file (WAV) to create compound clip")
	BuildCmd.Flags().String("with-duration", "", "Set custom duration in seconds (e.g., 900)")
	BuildCmd.Flags().Bool("with-slide", false, "Add slide animation that moves video to the right at 2 seconds")
}
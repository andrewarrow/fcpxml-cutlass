package build

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
)

var BuildCmd = &cobra.Command{
	Use:   "build [filename] [add-video] [media-file]",
	Short: "Build a blank FCP project or add media to existing project",
	Long:  "Create a blank Final Cut Pro project from empty.fcpxml template, or add a video/image to it",
	Args:  cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		filename := args[0]
		if !strings.HasSuffix(filename, ".fcpxml") {
			filename += ".fcpxml"
		}
		
		// Check if this is an add-video command
		if len(args) >= 3 && args[1] == "add-video" {
			mediaFile := args[2]
			
			// Get the --with-text flag value
			withText, _ := cmd.Flags().GetString("with-text")
			
			// First ensure the project exists, create if it doesn't
			if _, err := os.Stat(filename); os.IsNotExist(err) {
				err := createBlankProject(filename)
				if err != nil {
					fmt.Fprintf(os.Stderr, "Error creating blank project: %v\n", err)
					os.Exit(1)
				}
				fmt.Printf("Created blank project: %s\n", filename)
			}
			
			// Add media to the project
			err := addVideoToProject(filename, mediaFile, withText)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error adding media to project: %v\n", err)
				os.Exit(1)
			}
			
			fmt.Printf("Added media %s to project %s\n", mediaFile, filename)
			if withText != "" {
				fmt.Printf("Added text overlay: %s\n", withText)
			}
		} else {
			// Just create a blank project
			err := createBlankProject(filename)
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
}


